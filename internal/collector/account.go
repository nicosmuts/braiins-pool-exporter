package collector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

const accountEndpointProfile = "profile"

// AccountClient fetches verified account-level Braiins API data.
type AccountClient interface {
	Profile(context.Context, string) (braiins.ProfileResponse, error)
}

// AccountOptions configures account polling and metric conversion.
type AccountOptions struct {
	Client       AccountClient
	Coin         string
	PollInterval time.Duration
	Clock        Clock
}

// Clock provides testable time.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// AccountMetrics owns the cached account snapshot and account-related
// Prometheus metrics. API calls are made only by Poll or Run, never by Collect.
type AccountMetrics struct {
	client AccountClient
	coin   string
	clock  Clock

	mu          sync.RWMutex
	lastGood    *accountSnapshot
	lastSuccess time.Time
	lastError   string
	requests    map[string]float64
}

type accountSnapshot struct {
	collectedAt time.Time
	hashrates   map[string]float64
	balanceBTC  *float64
	workers     map[string]float64
}

// NewAccountMetrics constructs a cache-backed account collector.
func NewAccountMetrics(options AccountOptions) (*AccountMetrics, error) {
	if options.Client == nil {
		return nil, errors.New("account client is required")
	}
	coin := strings.ToLower(strings.TrimSpace(options.Coin))
	if coin == "" {
		coin = "btc"
	}
	clock := options.Clock
	if clock == nil {
		clock = realClock{}
	}
	return &AccountMetrics{
		client:   options.Client,
		coin:     coin,
		clock:    clock,
		requests: make(map[string]float64),
	}, nil
}

// RegisterAccountMetrics registers account data and account API self-metrics.
func RegisterAccountMetrics(registry *prometheus.Registry, metrics *AccountMetrics) {
	registry.MustRegister(metrics)
}

// Poll performs one bounded account profile request and updates the last-good
// cache only after a complete, valid account snapshot is built.
func (m *AccountMetrics) Poll(ctx context.Context) error {
	profile, err := m.client.Profile(ctx, m.coin)
	if err != nil {
		category := categorizeError(err)
		m.recordRequest(category)
		return fmt.Errorf("poll Braiins account profile: %w", err)
	}
	snapshot, err := buildAccountSnapshot(profile, m.coin, m.clock.Now())
	if err != nil {
		m.recordRequest("invalid_data")
		return err
	}
	m.recordRequest("success")
	m.mu.Lock()
	m.lastGood = snapshot
	m.lastSuccess = snapshot.collectedAt
	m.lastError = ""
	m.mu.Unlock()
	return nil
}

// Run polls immediately and then on the configured interval until ctx is done.
func (m *AccountMetrics) Run(ctx context.Context, interval time.Duration) {
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

// Ready reports whether at least one valid account snapshot has been accepted.
func (m *AccountMetrics) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastGood != nil
}

// LastSuccess reports the last accepted snapshot timestamp.
func (m *AccountMetrics) LastSuccess() (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.lastSuccess.IsZero() {
		return time.Time{}, false
	}
	return m.lastSuccess, true
}

// DataAge reports the age of the latest accepted snapshot.
func (m *AccountMetrics) DataAge() (time.Duration, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.lastSuccess.IsZero() {
		return 0, false
	}
	return m.clock.Now().Sub(m.lastSuccess), true
}

func (m *AccountMetrics) recordRequest(category string) {
	m.mu.Lock()
	m.lastError = category
	m.requests[category]++
	m.mu.Unlock()
}

// Describe sends deterministic metric descriptors.
func (m *AccountMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- accountHashrateDesc
	ch <- accountBalanceDesc
	ch <- accountWorkersDesc
	ch <- apiLastSuccessDesc
	ch <- dataAgeDesc
	ch <- apiRequestsTotalDesc
}

// Collect exposes only the cached last-good snapshot.
func (m *AccountMetrics) Collect(ch chan<- prometheus.Metric) {
	m.mu.RLock()
	snapshot := m.lastGood
	requests := copyCounters(m.requests)
	m.mu.RUnlock()
	for _, result := range sortedCounterKeys(requests) {
		ch <- prometheus.MustNewConstMetric(apiRequestsTotalDesc, prometheus.CounterValue, requests[result], accountEndpointProfile, result)
	}
	if snapshot == nil {
		return
	}
	for _, window := range []string{"5m", "60m", "24h", "yesterday"} {
		value, ok := snapshot.hashrates[window]
		if !ok {
			continue
		}
		ch <- prometheus.MustNewConstMetric(accountHashrateDesc, prometheus.GaugeValue, value, window)
	}
	if snapshot.balanceBTC != nil {
		ch <- prometheus.MustNewConstMetric(accountBalanceDesc, prometheus.GaugeValue, *snapshot.balanceBTC)
	}
	for _, state := range []string{"ok", "low", "off", "dis"} {
		ch <- prometheus.MustNewConstMetric(accountWorkersDesc, prometheus.GaugeValue, snapshot.workers[state], state)
	}
	ch <- prometheus.MustNewConstMetric(apiLastSuccessDesc, prometheus.GaugeValue, float64(snapshot.collectedAt.Unix()), accountEndpointProfile)
	ch <- prometheus.MustNewConstMetric(dataAgeDesc, prometheus.GaugeValue, m.clock.Now().Sub(snapshot.collectedAt).Seconds(), accountEndpointProfile)
}

var (
	accountHashrateDesc = prometheus.NewDesc(
		"braiins_pool_account_hashrate_ghs",
		"Account hashrate reported by Braiins Pool profile windows in Gh/s.",
		[]string{"window"},
		nil,
	)
	accountBalanceDesc = prometheus.NewDesc(
		"braiins_pool_account_balance_btc",
		"Current Braiins Pool account balance in BTC.",
		nil,
		nil,
	)
	accountWorkersDesc = prometheus.NewDesc(
		"braiins_pool_account_workers",
		"Account worker count reported by Braiins Pool profile state.",
		[]string{"state"},
		nil,
	)
	apiLastSuccessDesc = prometheus.NewDesc(
		"braiins_pool_api_last_success_timestamp_seconds",
		"Unix timestamp of the last successful Braiins Pool API poll by bounded endpoint.",
		[]string{"endpoint"},
		nil,
	)
	dataAgeDesc = prometheus.NewDesc(
		"braiins_pool_data_age_seconds",
		"Age in seconds of the latest accepted Braiins Pool account snapshot by bounded endpoint.",
		[]string{"endpoint"},
		nil,
	)
	apiRequestsTotalDesc = prometheus.NewDesc(
		"braiins_pool_api_requests_total",
		"Total Braiins Pool API requests by bounded endpoint and result.",
		[]string{"endpoint", "result"},
		nil,
	)
)

func buildAccountSnapshot(response braiins.ProfileResponse, coin string, collectedAt time.Time) (*accountSnapshot, error) {
	profile, ok := response.Coins[strings.ToLower(coin)]
	if !ok {
		return nil, errors.New("Braiins account profile missing configured coin")
	}
	if strings.TrimSpace(profile.HashRateUnit) != "Gh/s" {
		return nil, errors.New("Braiins account profile has unsupported hashrate unit")
	}
	hashrates := make(map[string]float64, 4)
	for window, value := range map[string]braiins.Decimal{
		"5m":        profile.HashRate5m,
		"60m":       profile.HashRate60m,
		"24h":       profile.HashRate24h,
		"yesterday": profile.HashRateYesterday,
	} {
		if value == "" {
			continue
		}
		parsed, err := decimalToFloat(value)
		if err != nil {
			return nil, fmt.Errorf("Braiins account profile has invalid hashrate value")
		}
		hashrates[window] = parsed
	}
	var balance *float64
	if profile.CurrentBalance != "" {
		parsed, err := decimalToFloat(profile.CurrentBalance)
		if err != nil {
			return nil, errors.New("Braiins account profile has invalid current balance")
		}
		balance = &parsed
	}
	return &accountSnapshot{
		collectedAt: collectedAt,
		hashrates:   hashrates,
		balanceBTC:  balance,
		workers: map[string]float64{
			"ok":  float64(profile.OKWorkers),
			"low": float64(profile.LowWorkers),
			"off": float64(profile.OffWorkers),
			"dis": float64(profile.DisabledWorkers),
		},
	}, nil
}

func decimalToFloat(value braiins.Decimal) (float64, error) {
	parsed, err := strconv.ParseFloat(value.String(), 64)
	if err != nil {
		return 0, err
	}
	if math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0, errors.New("decimal must be finite")
	}
	return parsed, nil
}

func categorizeError(err error) string {
	if err == nil {
		return "success"
	}
	var status braiins.StatusError
	if errors.As(err, &status) {
		switch status.StatusCode {
		case 401:
			return "unauthorized"
		case 403:
			return "forbidden"
		case 429:
			return "rate_limited"
		default:
			if status.StatusCode >= 500 && status.StatusCode <= 599 {
				return "server"
			}
			return "http_error"
		}
	}
	var decode braiins.DecodeError
	if errors.As(err, &decode) {
		return "decode"
	}
	var tooLarge braiins.ResponseTooLargeError
	if errors.As(err, &tooLarge) {
		return "invalid_data"
	}
	var transport braiins.TransportError
	if errors.As(err, &transport) {
		if transport.Timeout {
			return "timeout"
		}
		return "transport"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "error"
}
