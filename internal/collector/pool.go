package collector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

const poolEndpointStats = "pool_stats"

// PoolClient fetches verified pool-wide Braiins API data.
type PoolClient interface {
	PoolStats(context.Context, string) (braiins.CoinEnvelope[braiins.PoolStats], error)
}

// PoolOptions configures pool-wide telemetry polling and metric conversion.
type PoolOptions struct {
	Client PoolClient
	Coin   string
	Clock  Clock
}

// PoolMetrics owns cached pool-wide telemetry. API calls are made only by Poll
// or Run, never by Collect.
type PoolMetrics struct {
	client PoolClient
	coin   string
	clock  Clock

	mu          sync.RWMutex
	lastGood    *poolSnapshot
	lastSuccess time.Time
	lastError   string
	requests    map[string]float64
}

type poolSnapshot struct {
	collectedAt    time.Time
	hashrates      map[string]float64
	activeWorkers  float64
	sourceUpdateTS *float64
}

// NewPoolMetrics constructs a cache-backed pool-wide telemetry collector.
func NewPoolMetrics(options PoolOptions) (*PoolMetrics, error) {
	if options.Client == nil {
		return nil, errors.New("pool client is required")
	}
	coin := strings.ToLower(strings.TrimSpace(options.Coin))
	if coin == "" {
		coin = "btc"
	}
	clock := options.Clock
	if clock == nil {
		clock = realClock{}
	}
	return &PoolMetrics{
		client:   options.Client,
		coin:     coin,
		clock:    clock,
		requests: make(map[string]float64),
	}, nil
}

// RegisterPoolMetrics registers pool-wide data and pool API self-metrics.
func RegisterPoolMetrics(registry *prometheus.Registry, metrics *PoolMetrics) {
	registry.MustRegister(metrics)
}

// Poll performs one bounded pool stats request and updates the last-good cache
// only after a complete, valid pool snapshot is built.
func (m *PoolMetrics) Poll(ctx context.Context) error {
	envelope, err := m.client.PoolStats(ctx, m.coin)
	if err != nil {
		category := categorizeError(err)
		m.recordRequest(category)
		return fmt.Errorf("poll Braiins pool stats failed: %s", category)
	}
	snapshot, err := buildPoolSnapshot(envelope, m.coin, m.clock.Now())
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
func (m *PoolMetrics) Run(ctx context.Context, interval time.Duration) {
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

// Ready reports whether at least one valid pool snapshot has been accepted.
func (m *PoolMetrics) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastGood != nil
}

func (m *PoolMetrics) recordRequest(category string) {
	m.mu.Lock()
	m.lastError = category
	m.requests[category]++
	m.mu.Unlock()
}

// Describe sends deterministic metric descriptors.
func (m *PoolMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- poolHashrateDesc
	ch <- poolActiveWorkersDesc
	ch <- poolSourceUpdateTimestampDesc
}

// Collect exposes only the cached last-good pool snapshot.
func (m *PoolMetrics) Collect(ch chan<- prometheus.Metric) {
	m.mu.RLock()
	snapshot := m.lastGood
	requests := copyCounters(m.requests)
	m.mu.RUnlock()
	for _, result := range sortedCounterKeys(requests) {
		ch <- prometheus.MustNewConstMetric(apiRequestsTotalDesc, prometheus.CounterValue, requests[result], poolEndpointStats, result)
	}
	if snapshot == nil {
		return
	}
	for _, window := range []string{"5m", "60m", "24h"} {
		value, ok := snapshot.hashrates[window]
		if !ok {
			continue
		}
		ch <- prometheus.MustNewConstMetric(poolHashrateDesc, prometheus.GaugeValue, value, window)
	}
	ch <- prometheus.MustNewConstMetric(poolActiveWorkersDesc, prometheus.GaugeValue, snapshot.activeWorkers)
	if snapshot.sourceUpdateTS != nil {
		ch <- prometheus.MustNewConstMetric(poolSourceUpdateTimestampDesc, prometheus.GaugeValue, *snapshot.sourceUpdateTS)
	}
	ch <- prometheus.MustNewConstMetric(apiLastSuccessDesc, prometheus.GaugeValue, float64(snapshot.collectedAt.Unix()), poolEndpointStats)
	ch <- prometheus.MustNewConstMetric(dataAgeDesc, prometheus.GaugeValue, m.clock.Now().Sub(snapshot.collectedAt).Seconds(), poolEndpointStats)
}

var (
	poolHashrateDesc = prometheus.NewDesc(
		"braiins_pool_hashrate_ghs",
		"Pool-wide hashrate reported by the authenticated Braiins Pool stats endpoint in Gh/s.",
		[]string{"window"},
		nil,
	)
	poolActiveWorkersDesc = prometheus.NewDesc(
		"braiins_pool_active_workers",
		"Pool-wide active worker count reported by the authenticated Braiins Pool stats endpoint.",
		nil,
		nil,
	)
	poolSourceUpdateTimestampDesc = prometheus.NewDesc(
		"braiins_pool_stats_update_timestamp_seconds",
		"Unix timestamp of the pool stats source update reported by the authenticated Braiins Pool stats endpoint.",
		nil,
		nil,
	)
)

func buildPoolSnapshot(envelope braiins.CoinEnvelope[braiins.PoolStats], coin string, collectedAt time.Time) (*poolSnapshot, error) {
	stats, ok := envelope[strings.ToLower(coin)]
	if !ok {
		return nil, errors.New("Braiins pool stats response missing configured coin")
	}
	if strings.TrimSpace(stats.HashRateUnit) != "Gh/s" {
		return nil, errors.New("Braiins pool stats response has unsupported hashrate unit")
	}
	if stats.PoolActiveWorkers == nil {
		return nil, errors.New("Braiins pool stats response missing active workers")
	}
	hashrates := make(map[string]float64, 3)
	for window, value := range map[string]braiins.Decimal{
		"5m":  stats.PoolHashRate5m,
		"60m": stats.PoolHashRate60m,
		"24h": stats.PoolHashRate24h,
	} {
		if value == "" {
			continue
		}
		parsed, err := decimalToFloat(value)
		if err != nil {
			return nil, errors.New("Braiins pool stats response has invalid hashrate value")
		}
		hashrates[window] = parsed
	}
	activeWorkers := float64(*stats.PoolActiveWorkers)
	if activeWorkers < 0 {
		return nil, errors.New("Braiins pool stats response has invalid active workers")
	}
	var sourceUpdateTS *float64
	if stats.UpdateTS > 0 {
		value := float64(stats.UpdateTS)
		sourceUpdateTS = &value
	}
	return &poolSnapshot{
		collectedAt:    collectedAt,
		hashrates:      hashrates,
		activeWorkers:  activeWorkers,
		sourceUpdateTS: sourceUpdateTS,
	}, nil
}
