package collector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

const (
	historyEndpointRewards = "rewards"
	historyEndpointPayouts = "payouts"
	defaultHistoryDays     = 7
)

var (
	rewardComponents = []string{"total", "mining", "bos_plus", "referral_bonus", "referral_reward"}
	payoutRails      = []string{"onchain", "lightning"}
	payoutStatuses   = []string{"queued", "confirmed", "failed", "unknown"}
)

// HistoryClient fetches verified bounded-history Braiins API data.
type HistoryClient interface {
	Rewards(context.Context, string, string, string) (braiins.CoinEnvelope[braiins.RewardsResponse], error)
	Payouts(context.Context, string, string, string) (braiins.PayoutsResponse, error)
}

// HistoryOptions configures rewards and payouts polling.
type HistoryOptions struct {
	Client         HistoryClient
	Coin           string
	HistoryDays    int
	RewardsEnabled bool
	PayoutsEnabled bool
	Clock          Clock
}

// HistoryMetrics owns independently cached rewards and payouts snapshots.
// API calls are made only by PollRewards, PollPayouts, Poll, or Run.
type HistoryMetrics struct {
	client         HistoryClient
	coin           string
	historyDays    int
	rewardsEnabled bool
	payoutsEnabled bool
	clock          Clock

	mu       sync.RWMutex
	rewards  *rewardSnapshot
	payouts  *payoutSnapshot
	requests map[string]map[string]float64
}

type rewardSnapshot struct {
	collectedAt time.Time
	components  map[string]*big.Rat
}

type payoutSnapshot struct {
	collectedAt time.Time
	amounts     map[string]map[string]int64
	fees        map[string]map[string]int64
}

// NewHistoryMetrics constructs a cache-backed rewards and payouts collector.
func NewHistoryMetrics(options HistoryOptions) (*HistoryMetrics, error) {
	if options.Client == nil {
		return nil, errors.New("history client is required")
	}
	coin := strings.ToLower(strings.TrimSpace(options.Coin))
	if coin == "" {
		coin = "btc"
	}
	historyDays := options.HistoryDays
	if historyDays <= 0 {
		historyDays = defaultHistoryDays
	}
	clock := options.Clock
	if clock == nil {
		clock = realClock{}
	}
	return &HistoryMetrics{
		client:         options.Client,
		coin:           coin,
		historyDays:    historyDays,
		rewardsEnabled: options.RewardsEnabled,
		payoutsEnabled: options.PayoutsEnabled,
		clock:          clock,
		requests: map[string]map[string]float64{
			historyEndpointRewards: {},
			historyEndpointPayouts: {},
		},
	}, nil
}

// RegisterHistoryMetrics registers rewards and payouts data metrics.
func RegisterHistoryMetrics(registry *prometheus.Registry, metrics *HistoryMetrics) {
	registry.MustRegister(metrics)
}

// RewardsEnabled reports whether rewards polling is enabled.
func (m *HistoryMetrics) RewardsEnabled() bool { return m.rewardsEnabled }

// PayoutsEnabled reports whether payouts polling is enabled.
func (m *HistoryMetrics) PayoutsEnabled() bool { return m.payoutsEnabled }

// PollRewards performs one bounded rewards request and updates only the
// rewards last-good cache after a complete valid snapshot is built.
func (m *HistoryMetrics) PollRewards(ctx context.Context) error {
	from, to, start, end := m.dateWindow()
	envelope, err := m.client.Rewards(ctx, m.coin, from, to)
	if err != nil {
		category := categorizeError(err)
		m.recordHistoryRequest(historyEndpointRewards, category)
		return fmt.Errorf("poll Braiins rewards failed: %s", category)
	}
	snapshot, err := buildRewardSnapshot(envelope, m.coin, m.clock.Now(), start, end)
	if err != nil {
		m.recordHistoryRequest(historyEndpointRewards, "malformed")
		return err
	}
	m.recordHistoryRequest(historyEndpointRewards, "success")
	m.mu.Lock()
	m.rewards = snapshot
	m.mu.Unlock()
	return nil
}

// PollPayouts performs one bounded payouts request and updates only the payouts
// last-good cache after a complete valid snapshot is built.
func (m *HistoryMetrics) PollPayouts(ctx context.Context) error {
	from, to, start, end := m.dateWindow()
	response, err := m.client.Payouts(ctx, m.coin, from, to)
	if err != nil {
		category := categorizeError(err)
		m.recordHistoryRequest(historyEndpointPayouts, category)
		return fmt.Errorf("poll Braiins payouts failed: %s", category)
	}
	snapshot, err := buildPayoutSnapshot(response, m.clock.Now(), start, end)
	if err != nil {
		m.recordHistoryRequest(historyEndpointPayouts, "malformed")
		return err
	}
	m.recordHistoryRequest(historyEndpointPayouts, "success")
	m.mu.Lock()
	m.payouts = snapshot
	m.mu.Unlock()
	return nil
}

// Poll performs all enabled bounded-history requests. Each endpoint keeps an
// independent last-known-good cache.
func (m *HistoryMetrics) Poll(ctx context.Context) error {
	var errs []error
	if m.rewardsEnabled {
		if err := m.PollRewards(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if m.payoutsEnabled {
		if err := m.PollPayouts(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Run polls immediately and then on the configured interval until ctx is done.
func (m *HistoryMetrics) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	_ = m.Poll(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.Poll(ctx)
		}
	}
}

func (m *HistoryMetrics) dateWindow() (string, string, time.Time, time.Time) {
	now := m.clock.Now().UTC()
	toDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	fromDate := toDate.AddDate(0, 0, -(m.historyDays - 1))
	end := toDate.Add(24*time.Hour - time.Nanosecond)
	return fromDate.Format(time.DateOnly), toDate.Format(time.DateOnly), fromDate, end
}

func (m *HistoryMetrics) recordHistoryRequest(endpoint, category string) {
	m.mu.Lock()
	if m.requests[endpoint] == nil {
		m.requests[endpoint] = map[string]float64{}
	}
	m.requests[endpoint][category]++
	m.mu.Unlock()
}

// Describe sends deterministic metric descriptors.
func (m *HistoryMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- rewardDailyDesc
	ch <- payoutAmountDesc
	ch <- payoutFeeDesc
}

// Collect exposes only cached last-good rewards and payouts snapshots.
func (m *HistoryMetrics) Collect(ch chan<- prometheus.Metric) {
	m.mu.RLock()
	rewards := cloneRewardSnapshot(m.rewards)
	payouts := clonePayoutSnapshot(m.payouts)
	requests := make(map[string]map[string]float64, len(m.requests))
	for endpoint, counters := range m.requests {
		requests[endpoint] = copyCounters(counters)
	}
	m.mu.RUnlock()

	for _, endpoint := range sortedEndpointKeys(requests) {
		for _, result := range sortedCounterKeys(requests[endpoint]) {
			ch <- prometheus.MustNewConstMetric(apiRequestsTotalDesc, prometheus.CounterValue, requests[endpoint][result], endpoint, result)
		}
	}
	if rewards != nil {
		for _, component := range rewardComponents {
			ch <- prometheus.MustNewConstMetric(rewardDailyDesc, prometheus.GaugeValue, ratToFloat64(rewards.components[component]), component)
		}
		ch <- prometheus.MustNewConstMetric(apiLastSuccessDesc, prometheus.GaugeValue, float64(rewards.collectedAt.Unix()), historyEndpointRewards)
		ch <- prometheus.MustNewConstMetric(dataAgeDesc, prometheus.GaugeValue, m.clock.Now().Sub(rewards.collectedAt).Seconds(), historyEndpointRewards)
	}
	if payouts != nil {
		for _, rail := range payoutRails {
			for _, status := range payoutStatuses {
				ch <- prometheus.MustNewConstMetric(payoutAmountDesc, prometheus.GaugeValue, float64(payouts.amounts[rail][status]), rail, status)
				ch <- prometheus.MustNewConstMetric(payoutFeeDesc, prometheus.GaugeValue, float64(payouts.fees[rail][status]), rail, status)
			}
		}
		ch <- prometheus.MustNewConstMetric(apiLastSuccessDesc, prometheus.GaugeValue, float64(payouts.collectedAt.Unix()), historyEndpointPayouts)
		ch <- prometheus.MustNewConstMetric(dataAgeDesc, prometheus.GaugeValue, m.clock.Now().Sub(payouts.collectedAt).Seconds(), historyEndpointPayouts)
	}
}

var (
	rewardDailyDesc = prometheus.NewDesc(
		"braiins_pool_reward_daily_btc",
		"BTC rewards aggregated over the configured bounded daily rewards window by safe reward component.",
		[]string{"component"},
		nil,
	)
	payoutAmountDesc = prometheus.NewDesc(
		"braiins_pool_payout_amount_sats",
		"Payout amount in satoshis aggregated over the configured bounded payouts window by safe rail and status.",
		[]string{"rail", "status"},
		nil,
	)
	payoutFeeDesc = prometheus.NewDesc(
		"braiins_pool_payout_fee_sats",
		"Payout fee in satoshis aggregated over the configured bounded payouts window by safe rail and status.",
		[]string{"rail", "status"},
		nil,
	)
)

func buildRewardSnapshot(envelope braiins.CoinEnvelope[braiins.RewardsResponse], coin string, collectedAt, start, end time.Time) (*rewardSnapshot, error) {
	response, ok := envelope[strings.ToLower(coin)]
	if !ok {
		return nil, errors.New("Braiins rewards response missing configured coin")
	}
	components := zeroRewardComponents()
	seen := map[string]struct{}{}
	for _, reward := range response.DailyRewards {
		if !timestampInWindow(rewardTimestamp(reward), start, end) {
			continue
		}
		key := rewardDedupKey(reward)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		for component, value := range map[string]braiins.Decimal{
			"total":           reward.TotalReward,
			"mining":          reward.MiningReward,
			"bos_plus":        reward.BOSPlusReward,
			"referral_bonus":  reward.ReferralBonus,
			"referral_reward": reward.ReferralReward,
		} {
			if value == "" {
				continue
			}
			parsed, err := decimalToRat(value)
			if err != nil {
				return nil, errors.New("Braiins rewards response has invalid reward decimal")
			}
			components[component].Add(components[component], parsed)
		}
	}
	return &rewardSnapshot{collectedAt: collectedAt, components: components}, nil
}

func buildPayoutSnapshot(response braiins.PayoutsResponse, collectedAt, start, end time.Time) (*payoutSnapshot, error) {
	snapshot := &payoutSnapshot{
		collectedAt: collectedAt,
		amounts:     zeroPayoutTotals(),
		fees:        zeroPayoutTotals(),
	}
	seen := map[string]struct{}{}
	for _, rail := range payoutRails {
		var payouts []braiins.Payout
		switch rail {
		case "onchain":
			payouts = response.Onchain
		case "lightning":
			payouts = response.Lightning
		}
		for _, payout := range payouts {
			if !timestampInWindow(payoutTimestamp(payout), start, end) {
				continue
			}
			key := payoutDedupKey(rail, payout)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			status := normalizePayoutStatus(payout.Status)
			amount, err := safeAddInt64(snapshot.amounts[rail][status], payout.AmountSats)
			if err != nil {
				return nil, errors.New("Braiins payouts response amount exceeds supported range")
			}
			fee, err := safeAddInt64(snapshot.fees[rail][status], payout.FeeSats)
			if err != nil {
				return nil, errors.New("Braiins payouts response fee exceeds supported range")
			}
			snapshot.amounts[rail][status] = amount
			snapshot.fees[rail][status] = fee
		}
	}
	return snapshot, nil
}

func zeroRewardComponents() map[string]*big.Rat {
	values := make(map[string]*big.Rat, len(rewardComponents))
	for _, component := range rewardComponents {
		values[component] = new(big.Rat)
	}
	return values
}

func zeroPayoutTotals() map[string]map[string]int64 {
	values := make(map[string]map[string]int64, len(payoutRails))
	for _, rail := range payoutRails {
		values[rail] = make(map[string]int64, len(payoutStatuses))
		for _, status := range payoutStatuses {
			values[rail][status] = 0
		}
	}
	return values
}

func rewardTimestamp(reward braiins.DailyReward) int64 {
	if reward.Date != 0 {
		return reward.Date
	}
	return reward.CalculationDate
}

func payoutTimestamp(payout braiins.Payout) int64 {
	if payout.RequestedAtTS != 0 {
		return payout.RequestedAtTS
	}
	return payout.ResolvedAtTS
}

func timestampInWindow(unix int64, start, end time.Time) bool {
	if unix == 0 {
		return true
	}
	ts := time.Unix(unix, 0).UTC()
	return !ts.Before(start) && !ts.After(end)
}

func rewardDedupKey(reward braiins.DailyReward) string {
	parts := []string{
		fmt.Sprintf("%d", reward.Date),
		fmt.Sprintf("%d", reward.CalculationDate),
		reward.TotalReward.String(),
		reward.MiningReward.String(),
		reward.BOSPlusReward.String(),
		reward.ReferralBonus.String(),
		reward.ReferralReward.String(),
		reward.Shares.String(),
	}
	for _, price := range reward.SharePrices {
		parts = append(parts, price.String())
	}
	return strings.Join(parts, "\x00")
}

func payoutDedupKey(rail string, payout braiins.Payout) string {
	for _, candidate := range []*string{payout.TxID, payout.Invoice, payout.Preimage} {
		if candidate != nil && strings.TrimSpace(*candidate) != "" {
			return rail + "\x00id\x00" + strings.TrimSpace(*candidate)
		}
	}
	return strings.Join([]string{
		rail,
		payout.Status,
		fmt.Sprintf("%d", payout.RequestedAtTS),
		fmt.Sprintf("%d", payout.ResolvedAtTS),
		fmt.Sprintf("%d", payout.AmountSats),
		fmt.Sprintf("%d", payout.FeeSats),
		payout.TriggerType,
		payout.FinancialAccountName,
		payout.Destination,
	}, "\x00")
}

func normalizePayoutStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued":
		return "queued"
	case "confirmed":
		return "confirmed"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func decimalToRat(value braiins.Decimal) (*big.Rat, error) {
	text := strings.TrimSpace(value.String())
	if text == "" {
		return nil, errors.New("decimal is empty")
	}
	sign := 1
	if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		if text[0] == '-' {
			sign = -1
		}
		text = text[1:]
	}
	exponent := 0
	if idx := strings.IndexAny(text, "eE"); idx >= 0 {
		parsed, err := parseBase10Int(text[idx+1:])
		if err != nil {
			return nil, err
		}
		exponent = parsed
		text = text[:idx]
	}
	scale := 0
	if idx := strings.IndexByte(text, '.'); idx >= 0 {
		scale = len(text) - idx - 1
		text = text[:idx] + text[idx+1:]
	}
	if text == "" || strings.Trim(text, "0123456789") != "" {
		return nil, errors.New("decimal contains invalid digits")
	}
	numerator := new(big.Int)
	numerator.SetString(text, 10)
	if sign < 0 {
		numerator.Neg(numerator)
	}
	scale -= exponent
	if scale <= 0 {
		numerator.Mul(numerator, pow10(-scale))
		return new(big.Rat).SetInt(numerator), nil
	}
	return new(big.Rat).SetFrac(numerator, pow10(scale)), nil
}

func parseBase10Int(text string) (int, error) {
	if text == "" {
		return 0, errors.New("exponent is empty")
	}
	sign := 1
	if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		if text[0] == '-' {
			sign = -1
		}
		text = text[1:]
	}
	if text == "" || strings.Trim(text, "0123456789") != "" {
		return 0, errors.New("exponent contains invalid digits")
	}
	value := 0
	for _, digit := range text {
		value = value*10 + int(digit-'0')
		if value > 1000 {
			return 0, errors.New("decimal exponent is too large")
		}
	}
	return sign * value, nil
}

func pow10(exp int) *big.Int {
	result := big.NewInt(1)
	if exp <= 0 {
		return result
	}
	return result.Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}

func ratToFloat64(value *big.Rat) float64 {
	if value == nil {
		return 0
	}
	parsed, _ := value.Float64()
	if math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0
	}
	return parsed
}

func safeAddInt64(left, right int64) (int64, error) {
	if (right > 0 && left > math.MaxInt64-right) || (right < 0 && left < math.MinInt64-right) {
		return 0, errors.New("int64 overflow")
	}
	return left + right, nil
}

func cloneRewardSnapshot(source *rewardSnapshot) *rewardSnapshot {
	if source == nil {
		return nil
	}
	clone := &rewardSnapshot{collectedAt: source.collectedAt, components: make(map[string]*big.Rat, len(source.components))}
	for key, value := range source.components {
		clone.components[key] = new(big.Rat).Set(value)
	}
	return clone
}

func clonePayoutSnapshot(source *payoutSnapshot) *payoutSnapshot {
	if source == nil {
		return nil
	}
	clone := &payoutSnapshot{collectedAt: source.collectedAt, amounts: zeroPayoutTotals(), fees: zeroPayoutTotals()}
	for rail, byStatus := range source.amounts {
		for status, value := range byStatus {
			clone.amounts[rail][status] = value
		}
	}
	for rail, byStatus := range source.fees {
		for status, value := range byStatus {
			clone.fees[rail][status] = value
		}
	}
	return clone
}

func sortedEndpointKeys(counters map[string]map[string]float64) []string {
	keys := make([]string, 0, len(counters))
	for key := range counters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
