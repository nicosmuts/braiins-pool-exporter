package collector

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
	"github.com/nicosmuts/braiins-pool-exporter/internal/version"
)

func TestAccountMetricsExposeApprovedProfileMetrics(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1785080000, 0)}
	metrics := newTestAccountMetrics(t, fakeAccountClient{
		profiles: []braiins.ProfileResponse{testProfile()},
	}, clock)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}

	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, metrics)
	families := gatherFamilies(t, registry)

	assertMetric(t, families, "braiins_pool_account_hashrate_ghs", map[string]string{"window": "5m"}, 27978)
	assertMetric(t, families, "braiins_pool_account_hashrate_ghs", map[string]string{"window": "60m"}, 28191)
	assertMetric(t, families, "braiins_pool_account_hashrate_ghs", map[string]string{"window": "24h"}, 28357)
	assertMetric(t, families, "braiins_pool_account_hashrate_ghs", map[string]string{"window": "yesterday"}, 28197)
	assertMetric(t, families, "braiins_pool_account_balance_btc", nil, 0.01)
	assertMetric(t, families, "braiins_pool_account_workers", map[string]string{"state": "ok"}, 2)
	assertMetric(t, families, "braiins_pool_account_workers", map[string]string{"state": "low"}, 0)
	assertMetric(t, families, "braiins_pool_account_workers", map[string]string{"state": "off"}, 1)
	assertMetric(t, families, "braiins_pool_account_workers", map[string]string{"state": "dis"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_last_success_timestamp_seconds", map[string]string{"endpoint": "profile"}, float64(clock.now.Unix()))
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "profile"}, 0)
}

func TestAccountMetricsDoNotPollDuringGather(t *testing.T) {
	t.Parallel()

	client := &countingAccountClient{profile: testProfile()}
	metrics := newTestAccountMetrics(t, client, &fakeClock{now: time.Unix(1, 0)})
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, metrics)
	_ = gatherFamilies(t, registry)
	_ = gatherFamilies(t, registry)
	if got := client.calls; got != 1 {
		t.Fatalf("Profile calls after gathers = %d, want 1", got)
	}
}

func TestAccountMetricsKeepLastGoodSnapshotAfterFailure(t *testing.T) {
	t.Parallel()

	client := &sequenceAccountClient{
		profiles: []braiins.ProfileResponse{testProfile()},
		errors:   []error{braiins.StatusError{StatusCode: http.StatusForbidden, ContentType: "text/plain"}},
	}
	clock := &fakeClock{now: time.Unix(100, 0)}
	metrics := newTestAccountMetrics(t, client, clock)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("first Poll() error = %v", err)
	}
	clock.now = time.Unix(160, 0)
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("second Poll() error = nil, want error")
	}
	if !metrics.Ready() {
		t.Fatal("Ready() = false, want last-good snapshot to remain ready")
	}

	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, metrics)
	families := gatherFamilies(t, registry)
	assertMetric(t, families, "braiins_pool_account_balance_btc", nil, 0.01)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "forbidden"}, 1)
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "profile"}, 60)
}

func TestAccountMetricsFirstFailureOmitsAccountMetrics(t *testing.T) {
	t.Parallel()

	metrics := newTestAccountMetrics(t, fakeAccountClient{
		err: context.DeadlineExceeded,
	}, &fakeClock{now: time.Unix(1, 0)})
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("Poll() error = nil, want error")
	}
	if metrics.Ready() {
		t.Fatal("Ready() = true, want false without first snapshot")
	}
	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, metrics)
	families := gatherFamilies(t, registry)
	if _, ok := families["braiins_pool_account_balance_btc"]; ok {
		t.Fatal("account balance metric present before first success")
	}
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "timeout"}, 1)
}

func TestBuildAccountSnapshotOmitsOptionalDecimalFields(t *testing.T) {
	t.Parallel()

	profile := testProfile()
	btc := profile.Coins["btc"]
	btc.HashRateYesterday = ""
	btc.CurrentBalance = ""
	profile.Coins["btc"] = btc

	snapshot, err := buildAccountSnapshot(profile, "btc", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("buildAccountSnapshot() error = %v", err)
	}
	if _, ok := snapshot.hashrates["yesterday"]; ok {
		t.Fatal("hash_rate_yesterday produced a metric, want omitted")
	}
	if snapshot.balanceBTC != nil {
		t.Fatal("CurrentBalance metric present, want omitted")
	}
}

func TestBuildAccountSnapshotRejectsMalformedValues(t *testing.T) {
	t.Parallel()

	profile := testProfile()
	btc := profile.Coins["btc"]
	btc.HashRate5m = "not-a-number"
	profile.Coins["btc"] = btc
	if _, err := buildAccountSnapshot(profile, "btc", time.Unix(1, 0)); err == nil {
		t.Fatal("buildAccountSnapshot() error = nil, want error")
	}
}

func TestCategorizeAccountErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want string
	}{
		"unauthorized": {err: braiins.StatusError{StatusCode: http.StatusUnauthorized}, want: "unauthorized"},
		"forbidden":    {err: braiins.StatusError{StatusCode: http.StatusForbidden}, want: "forbidden"},
		"rate limited": {err: braiins.StatusError{StatusCode: http.StatusTooManyRequests}, want: "rate_limited"},
		"server":       {err: braiins.StatusError{StatusCode: http.StatusBadGateway}, want: "server"},
		"http":         {err: braiins.StatusError{StatusCode: http.StatusNotFound}, want: "http_error"},
		"canceled":     {err: context.Canceled, want: "canceled"},
		"timeout":      {err: context.DeadlineExceeded, want: "timeout"},
		"decode":       {err: braiins.DecodeError{Err: errors.New("unexpected EOF")}, want: "decode"},
		"too large":    {err: braiins.ResponseTooLargeError{}, want: "invalid_data"},
		"transport":    {err: braiins.TransportError{Err: errors.New("transport failed")}, want: "transport"},
		"other":        {err: errors.New("transport failed"), want: "error"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := categorizeError(tt.err); got != tt.want {
				t.Fatalf("categorizeError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAccountPollErrorDoesNotLeakTokenOrBody(t *testing.T) {
	t.Parallel()

	const secret = "distinctive-account-secret"
	client, err := braiins.NewClient(braiins.Config{
		BaseURL: "https://pool.braiins.com",
		Token:   braiins.Secret(secret),
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader(secret)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	metrics := newTestAccountMetrics(t, client, &fakeClock{now: time.Unix(1, 0)})
	err = metrics.Poll(context.Background())
	if err == nil {
		t.Fatal("Poll() error = nil, want error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("Poll() leaked secret: %v", err)
	}
}

func TestSelfMetricsCanRequireAccountReadiness(t *testing.T) {
	t.Parallel()

	_, self := NewRegistry(testBuildInfo())
	self.SetReady(true)
	accountReady := false
	self.RequireAccountReady(func() bool { return accountReady })
	if self.Ready() {
		t.Fatal("Ready() = true, want false while account is not ready")
	}
	accountReady = true
	if !self.Ready() {
		t.Fatal("Ready() = false, want true after account is ready")
	}
}

type fakeAccountClient struct {
	profiles []braiins.ProfileResponse
	err      error
}

func (c fakeAccountClient) Profile(context.Context, string) (braiins.ProfileResponse, error) {
	if c.err != nil {
		return braiins.ProfileResponse{}, c.err
	}
	if len(c.profiles) == 0 {
		return testProfile(), nil
	}
	return c.profiles[0], nil
}

type sequenceAccountClient struct {
	mu       sync.Mutex
	profiles []braiins.ProfileResponse
	errors   []error
	calls    int
}

func (c *sequenceAccountClient) Profile(context.Context, string) (braiins.ProfileResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.calls
	c.calls++
	if call < len(c.profiles) {
		return c.profiles[call], nil
	}
	errIndex := call - len(c.profiles)
	if errIndex < len(c.errors) {
		return braiins.ProfileResponse{}, c.errors[errIndex]
	}
	return braiins.ProfileResponse{}, errors.New("unexpected call")
}

type countingAccountClient struct {
	profile braiins.ProfileResponse
	calls   int
}

func (c *countingAccountClient) Profile(context.Context, string) (braiins.ProfileResponse, error) {
	c.calls++
	return c.profile, nil
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testBuildInfo() version.Info {
	return version.Info{
		Version:   "test",
		Commit:    "abc123",
		BuildDate: "2026-07-26T00:00:00Z",
		GoVersion: "go1.test",
	}
}

func newTestAccountMetrics(t *testing.T, client AccountClient, clock Clock) *AccountMetrics {
	t.Helper()
	metrics, err := NewAccountMetrics(AccountOptions{Client: client, Coin: "btc", Clock: clock})
	if err != nil {
		t.Fatalf("NewAccountMetrics() error = %v", err)
	}
	return metrics
}

func testProfile() braiins.ProfileResponse {
	return braiins.ProfileResponse{
		Username: "example-user",
		Coins: map[string]braiins.Profile{
			"btc": {
				AllTimeReward:     "0.15000000",
				HashRateUnit:      "Gh/s",
				HashRate5m:        "27978",
				HashRate60m:       "28191",
				HashRate24h:       "28357",
				HashRateYesterday: "28197",
				LowWorkers:        0,
				OffWorkers:        1,
				OKWorkers:         2,
				DisabledWorkers:   1,
				CurrentBalance:    "0.01000000",
				TodayReward:       "0.000166667",
				EstimatedReward:   "0.00011940",
				Shares5m:          "123",
				Shares60m:         "1476",
				Shares24h:         "35424",
				SharesYesterday:   "0",
			},
		},
	}
}

func gatherFamilies(t *testing.T, registry *prometheus.Registry) map[string]*dto.MetricFamily {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		byName[family.GetName()] = family
	}
	return byName
}

func assertMetric(t *testing.T, families map[string]*dto.MetricFamily, name string, labels map[string]string, want float64) {
	t.Helper()
	family, ok := families[name]
	if !ok {
		t.Fatalf("metric family %q not found", name)
	}
	for _, metric := range family.Metric {
		if labelsMatch(metric, labels) {
			got := metricValue(metric)
			if math.Abs(got-want) > 0.000000001 {
				t.Fatalf("%s%v = %v, want %v", name, labels, got, want)
			}
			return
		}
	}
	t.Fatalf("metric %q with labels %v not found", name, labels)
}

func labelsMatch(metric *dto.Metric, want map[string]string) bool {
	if len(metric.Label) != len(want) {
		return false
	}
	for _, label := range metric.Label {
		if want[label.GetName()] != label.GetValue() {
			return false
		}
	}
	return true
}

func metricValue(metric *dto.Metric) float64 {
	if metric.Gauge != nil {
		return metric.Gauge.GetValue()
	}
	if metric.Counter != nil {
		return metric.Counter.GetValue()
	}
	return math.NaN()
}
