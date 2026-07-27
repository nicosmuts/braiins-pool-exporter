package collector

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

func TestPoolMetricsExposeApprovedStatsMetrics(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1785080300, 0)}
	metrics := newTestPoolMetrics(t, fakePoolClient{
		stats: []braiins.CoinEnvelope[braiins.PoolStats]{testPoolStats()},
	}, clock)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}

	registry := prometheus.NewRegistry()
	RegisterPoolMetrics(registry, metrics)
	families := gatherFamilies(t, registry)

	assertMetric(t, families, "braiins_pool_hashrate_ghs", map[string]string{"window": "5m"}, 5727000000)
	assertMetric(t, families, "braiins_pool_hashrate_ghs", map[string]string{"window": "60m"}, 5719000000)
	assertMetric(t, families, "braiins_pool_hashrate_ghs", map[string]string{"window": "24h"}, 5701000000)
	assertMetric(t, families, "braiins_pool_active_workers", nil, 123456)
	assertMetric(t, families, "braiins_pool_stats_update_timestamp_seconds", nil, 1785080000)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "pool_stats", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_last_success_timestamp_seconds", map[string]string{"endpoint": "pool_stats"}, float64(clock.now.Unix()))
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "pool_stats"}, 0)

	assertMetricType(t, families, "braiins_pool_hashrate_ghs", dto.MetricType_GAUGE)
	assertMetricType(t, families, "braiins_pool_active_workers", dto.MetricType_GAUGE)
	assertMetricType(t, families, "braiins_pool_api_requests_total", dto.MetricType_COUNTER)
}

func TestPoolMetricsKeepLastGoodSnapshotAfterFailure(t *testing.T) {
	t.Parallel()

	client := &sequencePoolClient{
		stats:  []braiins.CoinEnvelope[braiins.PoolStats]{testPoolStats()},
		errors: []error{braiins.StatusError{StatusCode: http.StatusBadGateway}},
	}
	clock := &fakeClock{now: time.Unix(100, 0)}
	metrics := newTestPoolMetrics(t, client, clock)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("first Poll() error = %v", err)
	}
	clock.now = time.Unix(160, 0)
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("second Poll() error = nil, want error")
	}

	registry := prometheus.NewRegistry()
	RegisterPoolMetrics(registry, metrics)
	families := gatherFamilies(t, registry)
	assertMetric(t, families, "braiins_pool_active_workers", nil, 123456)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "pool_stats", "result": "server"}, 1)
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "pool_stats"}, 60)
}

func TestPoolMetricsFirstFailureOmitsPoolMetrics(t *testing.T) {
	t.Parallel()

	metrics := newTestPoolMetrics(t, fakePoolClient{err: context.DeadlineExceeded}, &fakeClock{now: time.Unix(1, 0)})
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("Poll() error = nil, want error")
	}

	registry := prometheus.NewRegistry()
	RegisterPoolMetrics(registry, metrics)
	families := gatherFamilies(t, registry)
	if _, ok := families["braiins_pool_active_workers"]; ok {
		t.Fatal("active worker metric present before first success")
	}
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "pool_stats", "result": "timeout"}, 1)
}

func TestBuildPoolSnapshotRejectsUnsupportedOrMissingFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutate func(*braiins.PoolStats)
	}{
		"missing active workers": {mutate: func(stats *braiins.PoolStats) { stats.PoolActiveWorkers = nil }},
		"negative active workers": {mutate: func(stats *braiins.PoolStats) {
			workers := int64(-1)
			stats.PoolActiveWorkers = &workers
		}},
		"unsupported hashrate unit": {mutate: func(stats *braiins.PoolStats) { stats.HashRateUnit = "Th/s" }},
		"malformed hashrate":        {mutate: func(stats *braiins.PoolStats) { stats.PoolHashRate5m = "not-a-number" }},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			envelope := testPoolStats()
			stats := envelope["btc"]
			tt.mutate(&stats)
			envelope["btc"] = stats
			if _, err := buildPoolSnapshot(envelope, "btc", time.Unix(1, 0)); err == nil {
				t.Fatal("buildPoolSnapshot() error = nil, want error")
			}
		})
	}
}

func TestBuildPoolSnapshotOmitsOptionalHashrateAndUpdateTimestamp(t *testing.T) {
	t.Parallel()

	envelope := testPoolStats()
	stats := envelope["btc"]
	stats.PoolHashRate24h = ""
	stats.UpdateTS = 0
	envelope["btc"] = stats

	snapshot, err := buildPoolSnapshot(envelope, "btc", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("buildPoolSnapshot() error = %v", err)
	}
	if _, ok := snapshot.hashrates["24h"]; ok {
		t.Fatal("24h hashrate produced a metric, want omitted")
	}
	if snapshot.sourceUpdateTS != nil {
		t.Fatal("source update timestamp present, want omitted")
	}
}

func assertMetricType(t *testing.T, families map[string]*dto.MetricFamily, name string, want dto.MetricType) {
	t.Helper()
	family, ok := families[name]
	if !ok {
		t.Fatalf("metric family %q not found", name)
	}
	if family.GetType() != want {
		t.Fatalf("metric family %q type = %s, want %s", name, family.GetType(), want)
	}
}

type fakePoolClient struct {
	stats []braiins.CoinEnvelope[braiins.PoolStats]
	err   error
}

func (c fakePoolClient) PoolStats(context.Context, string) (braiins.CoinEnvelope[braiins.PoolStats], error) {
	if c.err != nil {
		return nil, c.err
	}
	if len(c.stats) == 0 {
		return testPoolStats(), nil
	}
	return c.stats[0], nil
}

type sequencePoolClient struct {
	stats  []braiins.CoinEnvelope[braiins.PoolStats]
	errors []error
	calls  int
}

func (c *sequencePoolClient) PoolStats(context.Context, string) (braiins.CoinEnvelope[braiins.PoolStats], error) {
	call := c.calls
	c.calls++
	if call < len(c.stats) {
		return c.stats[call], nil
	}
	errIndex := call - len(c.stats)
	if errIndex < len(c.errors) {
		return nil, c.errors[errIndex]
	}
	return nil, errors.New("unexpected call")
}

func newTestPoolMetrics(t *testing.T, client PoolClient, clock Clock) *PoolMetrics {
	t.Helper()
	metrics, err := NewPoolMetrics(PoolOptions{Client: client, Coin: "btc", Clock: clock})
	if err != nil {
		t.Fatalf("NewPoolMetrics() error = %v", err)
	}
	return metrics
}

func testPoolStats() braiins.CoinEnvelope[braiins.PoolStats] {
	workers := int64(123456)
	return braiins.CoinEnvelope[braiins.PoolStats]{
		"btc": {
			HashRateUnit:      "Gh/s",
			PoolActiveWorkers: &workers,
			PoolHashRate5m:    "5727000000",
			PoolHashRate60m:   "5719000000",
			PoolHashRate24h:   "5701000000",
			UpdateTS:          1785080000,
		},
	}
}
