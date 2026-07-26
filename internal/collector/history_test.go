package collector

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

func TestHistoryMetricsExposeRewardsAndPayouts(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)}
	client := &recordingHistoryClient{
		rewards: []braiins.CoinEnvelope[braiins.RewardsResponse]{testRewards(
			testReward(unixDate(2026, 7, 25), "0.00000001", "0.00000001"),
			testReward(unixDate(2026, 7, 26), "2e-8", "0.00000002"),
		)},
		payouts: []braiins.PayoutsResponse{{
			Onchain:   []braiins.Payout{testPayout("confirmed", 1000, 12, unixDate(2026, 7, 25))},
			Lightning: []braiins.Payout{testLightningPayout("queued", 2000, 0, unixDate(2026, 7, 26))},
		}},
	}
	metrics := newTestHistoryMetrics(t, client, clock, 7, true, true)
	if err := metrics.PollRewards(context.Background()); err != nil {
		t.Fatalf("PollRewards() error = %v", err)
	}
	if err := metrics.PollPayouts(context.Background()); err != nil {
		t.Fatalf("PollPayouts() error = %v", err)
	}
	if got := strings.Join(client.rewardWindows, ","); got != "2026-07-20/2026-07-26" {
		t.Fatalf("reward window = %q", got)
	}
	if got := strings.Join(client.payoutWindows, ","); got != "2026-07-20/2026-07-26" {
		t.Fatalf("payout window = %q", got)
	}

	families := gatherRegisteredHistoryFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "total"}, 0.00000003)
	assertMetric(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "mining"}, 0.00000003)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 1000)
	assertMetric(t, families, "braiins_pool_payout_fee_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 12)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "lightning", "status": "queued"}, 2000)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "rewards", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "payouts", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_last_success_timestamp_seconds", map[string]string{"endpoint": "rewards"}, float64(clock.now.Unix()))
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "payouts"}, 0)
	if hasLabelValue(families["braiins_pool_payout_amount_sats"], "status", "tx-confirmed-1000-12") {
		t.Fatal("sensitive payout identifier appeared as a label")
	}
}

func TestHistoryMetricsDeduplicateAndFilterWindow(t *testing.T) {
	t.Parallel()

	inside := testReward(unixDate(2026, 7, 26), "0.00000005", "0.00000005")
	duplicate := inside
	outside := testReward(unixDate(2026, 7, 19), "0.5", "0.5")
	onchain := testPayout("confirmed", 100, 1, unixDate(2026, 7, 26))
	onchainDuplicate := onchain
	outsidePayout := testPayout("confirmed", 900, 9, unixDate(2026, 7, 19))
	metrics := newTestHistoryMetrics(t, fakeHistoryClient{
		reward: testRewards(inside, duplicate, outside),
		payout: testPayouts(
			onchain,
			onchainDuplicate,
			outsidePayout,
			testPayout("unexpected-status", 25, 0, unixDate(2026, 7, 26)),
		),
	}, &fakeClock{now: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC)}, 7, true, true)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}

	families := gatherRegisteredHistoryFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "total"}, 0.00000005)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 100)
	assertMetric(t, families, "braiins_pool_payout_fee_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 1)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "unknown"}, 25)
	if hasLabelValue(families["braiins_pool_payout_amount_sats"], "status", "unexpected-status") {
		t.Fatal("raw unknown payout status was emitted as a label")
	}
}

func TestHistoryMetricsEndpointFailuresAreIndependent(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(100, 0)}
	client := &sequenceHistoryClient{
		rewards:      []braiins.CoinEnvelope[braiins.RewardsResponse]{testRewards(testReward(100, "0.1", "0.1"))},
		payouts:      []braiins.PayoutsResponse{testPayouts(testPayout("confirmed", 10, 1, 100))},
		rewardErrors: []error{context.DeadlineExceeded},
		payoutsMore:  []braiins.PayoutsResponse{testPayouts(testPayout("confirmed", 20, 2, 130))},
	}
	metrics := newTestHistoryMetrics(t, client, clock, 7, true, true)
	if err := metrics.PollRewards(context.Background()); err != nil {
		t.Fatalf("first PollRewards() error = %v", err)
	}
	if err := metrics.PollPayouts(context.Background()); err != nil {
		t.Fatalf("first PollPayouts() error = %v", err)
	}
	clock.now = time.Unix(130, 0)
	if err := metrics.PollRewards(context.Background()); err == nil {
		t.Fatal("second PollRewards() error = nil, want error")
	}
	if err := metrics.PollPayouts(context.Background()); err != nil {
		t.Fatalf("second PollPayouts() error = %v", err)
	}

	families := gatherRegisteredHistoryFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "total"}, 0.1)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 20)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "rewards", "result": "timeout"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "payouts", "result": "success"}, 2)
}

func TestHistoryMetricsEmptySuccessClearsPriorData(t *testing.T) {
	t.Parallel()

	client := &sequenceHistoryClient{
		rewards: []braiins.CoinEnvelope[braiins.RewardsResponse]{
			testRewards(testReward(100, "0.1", "0.1")),
			testRewards(),
		},
		payouts: []braiins.PayoutsResponse{
			testPayouts(testPayout("confirmed", 10, 1, 100)),
			testPayouts(),
		},
	}
	metrics := newTestHistoryMetrics(t, client, &fakeClock{now: time.Unix(100, 0)}, 7, true, true)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("first Poll() error = %v", err)
	}
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("second Poll() error = %v", err)
	}
	families := gatherRegisteredHistoryFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "total"}, 0)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 0)
}

func TestHistoryMetricsFirstFailureOmitsDataMetrics(t *testing.T) {
	t.Parallel()

	metrics := newTestHistoryMetrics(t, fakeHistoryClient{
		rewardErr: errors.New("transport failed"),
		payoutErr: context.DeadlineExceeded,
	}, &fakeClock{now: time.Unix(100, 0)}, 7, true, true)
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("Poll() error = nil, want error")
	}
	families := gatherRegisteredHistoryFamilies(t, metrics)
	assertMetricAbsent(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "total"})
	assertMetricAbsent(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "confirmed"})
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "payouts", "result": "timeout"}, 1)
}

func TestHistoryMetricsRejectMalformedRewardDecimalAndPayoutOverflow(t *testing.T) {
	t.Parallel()

	rewards := newTestHistoryMetrics(t, fakeHistoryClient{
		reward: testRewards(testReward(100, "not-a-decimal", "0")),
	}, &fakeClock{now: time.Unix(100, 0)}, 7, true, false)
	if err := rewards.PollRewards(context.Background()); err == nil {
		t.Fatal("PollRewards() error = nil, want malformed decimal error")
	}

	payouts := newTestHistoryMetrics(t, fakeHistoryClient{
		payout: testPayouts(
			testPayout("confirmed", 9223372036854775807, 0, 100),
			testPayout("confirmed", 1, 0, 101),
		),
	}, &fakeClock{now: time.Unix(100, 0)}, 7, false, true)
	if err := payouts.PollPayouts(context.Background()); err == nil {
		t.Fatal("PollPayouts() error = nil, want overflow error")
	}
}

func TestHistoryMetricsRejectRecordLimitsAndPreserveLastGood(t *testing.T) {
	t.Parallel()

	rewards := make([]braiins.DailyReward, maxRewardRecords+1)
	for i := range rewards {
		rewards[i] = testReward(100+int64(i), "0.00000001", "0.00000001")
	}
	payouts := make([]braiins.Payout, maxPayoutRecords+1)
	for i := range payouts {
		payouts[i] = testPayout("confirmed", 1, 0, 100+int64(i))
	}
	client := &sequenceHistoryClient{
		rewards: []braiins.CoinEnvelope[braiins.RewardsResponse]{
			testRewards(testReward(100, "0.1", "0.1")),
			testRewards(rewards...),
		},
		payouts: []braiins.PayoutsResponse{
			testPayouts(testPayout("confirmed", 10, 1, 100)),
			testPayouts(payouts...),
		},
	}
	metrics := newTestHistoryMetrics(t, client, &fakeClock{now: time.Unix(100, 0)}, 7, true, true)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("first Poll() error = %v", err)
	}
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("second Poll() error = nil, want record-limit errors")
	}
	families := gatherRegisteredHistoryFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_reward_daily_btc", map[string]string{"component": "total"}, 0.1)
	assertMetric(t, families, "braiins_pool_payout_amount_sats", map[string]string{"rail": "onchain", "status": "confirmed"}, 10)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "rewards", "result": "invalid_data"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "payouts", "result": "invalid_data"}, 1)
}

func TestHistoryAndAccountAndWorkerMetricsRegisterTogether(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(100, 0)}
	account := newTestAccountMetrics(t, fakeAccountClient{profiles: []braiins.ProfileResponse{testProfile()}}, clock)
	worker := newTestWorkerMetrics(t, fakeWorkerClient{responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{testWorkers("worker-a")}}, clock, 100)
	history := newTestHistoryMetrics(t, fakeHistoryClient{
		reward: testRewards(testReward(100, "0.1", "0.1")),
		payout: testPayouts(testPayout("confirmed", 10, 1, 100)),
	}, clock, 7, true, true)
	if err := account.Poll(context.Background()); err != nil {
		t.Fatalf("account Poll() error = %v", err)
	}
	if err := worker.Poll(context.Background()); err != nil {
		t.Fatalf("worker Poll() error = %v", err)
	}
	if err := history.Poll(context.Background()); err != nil {
		t.Fatalf("history Poll() error = %v", err)
	}
	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, account)
	RegisterWorkerMetrics(registry, worker)
	RegisterHistoryMetrics(registry, history)
	families := gatherFamilies(t, registry)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "rewards", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "payouts", "result": "success"}, 1)
}

func gatherRegisteredHistoryFamilies(t *testing.T, metrics *HistoryMetrics) map[string]*dto.MetricFamily {
	t.Helper()
	registry := prometheus.NewRegistry()
	RegisterHistoryMetrics(registry, metrics)
	return gatherFamilies(t, registry)
}

func newTestHistoryMetrics(t *testing.T, client HistoryClient, clock Clock, days int, rewards, payouts bool) *HistoryMetrics {
	t.Helper()
	metrics, err := NewHistoryMetrics(HistoryOptions{
		Client:         client,
		Coin:           "btc",
		HistoryDays:    days,
		RewardsEnabled: rewards,
		PayoutsEnabled: payouts,
		Clock:          clock,
	})
	if err != nil {
		t.Fatalf("NewHistoryMetrics() error = %v", err)
	}
	return metrics
}

func testRewards(rewards ...braiins.DailyReward) braiins.CoinEnvelope[braiins.RewardsResponse] {
	return braiins.CoinEnvelope[braiins.RewardsResponse]{"btc": {DailyRewards: rewards}}
}

func testReward(date int64, total, mining string) braiins.DailyReward {
	return braiins.DailyReward{
		Date:           date,
		TotalReward:    braiins.Decimal(total),
		MiningReward:   braiins.Decimal(mining),
		BOSPlusReward:  "0",
		ReferralBonus:  "0",
		ReferralReward: "0",
		Shares:         "1",
	}
}

func testPayouts(payouts ...braiins.Payout) braiins.PayoutsResponse {
	return braiins.PayoutsResponse{Onchain: payouts}
}

func testPayout(status string, amount, fee, requestedAt int64) braiins.Payout {
	txID := "tx-" + status + "-" + fmtInt(amount) + "-" + fmtInt(requestedAt)
	return braiins.Payout{
		RequestedAtTS: requestedAt,
		Status:        status,
		AmountSats:    amount,
		FeeSats:       fee,
		Destination:   "private-address",
		TxID:          &txID,
		TriggerType:   "manual",
	}
}

func fmtInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func testLightningPayout(status string, amount, fee, requestedAt int64) braiins.Payout {
	invoice := "private-invoice"
	return braiins.Payout{
		RequestedAtTS: requestedAt,
		Status:        status,
		AmountSats:    amount,
		FeeSats:       fee,
		Invoice:       &invoice,
		TriggerType:   "triggered",
	}
}

func unixDate(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC).Unix()
}

type fakeHistoryClient struct {
	reward    braiins.CoinEnvelope[braiins.RewardsResponse]
	payout    braiins.PayoutsResponse
	rewardErr error
	payoutErr error
}

func (c fakeHistoryClient) Rewards(context.Context, string, string, string) (braiins.CoinEnvelope[braiins.RewardsResponse], error) {
	if c.rewardErr != nil {
		return nil, c.rewardErr
	}
	return c.reward, nil
}

func (c fakeHistoryClient) Payouts(context.Context, string, string, string) (braiins.PayoutsResponse, error) {
	if c.payoutErr != nil {
		return braiins.PayoutsResponse{}, c.payoutErr
	}
	return c.payout, nil
}

type recordingHistoryClient struct {
	rewards       []braiins.CoinEnvelope[braiins.RewardsResponse]
	payouts       []braiins.PayoutsResponse
	rewardWindows []string
	payoutWindows []string
}

func (c *recordingHistoryClient) Rewards(_ context.Context, _ string, from, to string) (braiins.CoinEnvelope[braiins.RewardsResponse], error) {
	c.rewardWindows = append(c.rewardWindows, from+"/"+to)
	if len(c.rewards) == 0 {
		return testRewards(), nil
	}
	return c.rewards[0], nil
}

func (c *recordingHistoryClient) Payouts(_ context.Context, _ string, from, to string) (braiins.PayoutsResponse, error) {
	c.payoutWindows = append(c.payoutWindows, from+"/"+to)
	if len(c.payouts) == 0 {
		return testPayouts(), nil
	}
	return c.payouts[0], nil
}

type sequenceHistoryClient struct {
	rewards      []braiins.CoinEnvelope[braiins.RewardsResponse]
	payouts      []braiins.PayoutsResponse
	payoutsMore  []braiins.PayoutsResponse
	rewardErrors []error
	rewardCalls  int
	payoutCalls  int
}

func (c *sequenceHistoryClient) Rewards(context.Context, string, string, string) (braiins.CoinEnvelope[braiins.RewardsResponse], error) {
	call := c.rewardCalls
	c.rewardCalls++
	if call < len(c.rewards) {
		return c.rewards[call], nil
	}
	errIndex := call - len(c.rewards)
	if errIndex < len(c.rewardErrors) {
		return nil, c.rewardErrors[errIndex]
	}
	return testRewards(), nil
}

func (c *sequenceHistoryClient) Payouts(context.Context, string, string, string) (braiins.PayoutsResponse, error) {
	call := c.payoutCalls
	c.payoutCalls++
	if call < len(c.payouts) {
		return c.payouts[call], nil
	}
	next := call - len(c.payouts)
	if next < len(c.payoutsMore) {
		return c.payoutsMore[next], nil
	}
	return testPayouts(), nil
}
