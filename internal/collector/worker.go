package collector

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

const (
	workerEndpointWorkers = "workers"
	defaultMaxWorkers     = 100
	maxWorkerLabelLength  = 128
)

var (
	workerStateValues     = []string{"ok", "low", "off", "dis", "unknown"}
	workerHashrateWindows = []string{"scoring", "5m", "60m", "24h"}
	workerShareWindows    = []string{"5m", "60m", "24h"}
)

// WorkerClient fetches verified worker-level Braiins API data.
type WorkerClient interface {
	Workers(context.Context, string) (braiins.CoinEnvelope[braiins.WorkersResponse], error)
}

// WorkerOptions configures worker polling and metric conversion.
type WorkerOptions struct {
	Client     WorkerClient
	Coin       string
	MaxWorkers int
	Clock      Clock
}

// WorkerMetrics owns the cached worker snapshot and worker-related Prometheus
// metrics. API calls are made only by Poll or Run, never by Collect.
type WorkerMetrics struct {
	client     WorkerClient
	coin       string
	maxWorkers int
	clock      Clock

	mu          sync.RWMutex
	lastGood    *workerSnapshot
	lastSuccess time.Time
	lastError   string
	requests    map[string]float64
}

type workerSnapshot struct {
	collectedAt time.Time
	workers     []workerSample
}

type workerSample struct {
	label              string
	state              string
	hashrates          map[string]float64
	shares             map[string]float64
	lastShareTimestamp *float64
}

// NewWorkerMetrics constructs a cache-backed worker collector.
func NewWorkerMetrics(options WorkerOptions) (*WorkerMetrics, error) {
	if options.Client == nil {
		return nil, errors.New("worker client is required")
	}
	coin := strings.ToLower(strings.TrimSpace(options.Coin))
	if coin == "" {
		coin = "btc"
	}
	maxWorkers := options.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = defaultMaxWorkers
	}
	clock := options.Clock
	if clock == nil {
		clock = realClock{}
	}
	return &WorkerMetrics{
		client:     options.Client,
		coin:       coin,
		maxWorkers: maxWorkers,
		clock:      clock,
		requests:   make(map[string]float64),
	}, nil
}

// RegisterWorkerMetrics registers worker data and worker API self-metrics.
func RegisterWorkerMetrics(registry *prometheus.Registry, metrics *WorkerMetrics) {
	registry.MustRegister(metrics)
}

// Poll performs one bounded workers request and replaces the last-good cache
// only after a complete, valid worker snapshot is built.
func (m *WorkerMetrics) Poll(ctx context.Context) error {
	envelope, err := m.client.Workers(ctx, m.coin)
	if err != nil {
		category := categorizeError(err)
		m.recordRequest(category)
		return fmt.Errorf("poll Braiins workers failed: %s", category)
	}
	snapshot, err := buildWorkerSnapshot(envelope, m.coin, m.clock.Now(), m.maxWorkers)
	if err != nil {
		category := "malformed"
		if errors.Is(err, errWorkerLimitExceeded) {
			category = "limit_exceeded"
		}
		m.recordRequest(category)
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
func (m *WorkerMetrics) Run(ctx context.Context, interval time.Duration) {
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

// Ready reports whether at least one valid worker snapshot has been accepted.
func (m *WorkerMetrics) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastGood != nil
}

func (m *WorkerMetrics) recordRequest(category string) {
	m.mu.Lock()
	m.lastError = category
	m.requests[category]++
	m.mu.Unlock()
}

// Describe sends deterministic metric descriptors.
func (m *WorkerMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- workerStateDesc
	ch <- workerHashrateDesc
	ch <- workerSharesDesc
	ch <- workerLastShareTimestampDesc
	ch <- workerLastShareAgeDesc
}

// Collect exposes only the cached last-good worker snapshot.
func (m *WorkerMetrics) Collect(ch chan<- prometheus.Metric) {
	m.mu.RLock()
	snapshot := m.lastGood
	requests := copyCounters(m.requests)
	m.mu.RUnlock()
	for _, result := range sortedCounterKeys(requests) {
		ch <- prometheus.MustNewConstMetric(apiRequestsTotalDesc, prometheus.CounterValue, requests[result], workerEndpointWorkers, result)
	}
	if snapshot == nil {
		return
	}
	for _, worker := range snapshot.workers {
		for _, state := range workerStateValues {
			value := 0.0
			if worker.state == state {
				value = 1
			}
			ch <- prometheus.MustNewConstMetric(workerStateDesc, prometheus.GaugeValue, value, worker.label, state)
		}
		for _, window := range workerHashrateWindows {
			value, ok := worker.hashrates[window]
			if !ok {
				continue
			}
			ch <- prometheus.MustNewConstMetric(workerHashrateDesc, prometheus.GaugeValue, value, worker.label, window)
		}
		for _, window := range workerShareWindows {
			value, ok := worker.shares[window]
			if !ok {
				continue
			}
			ch <- prometheus.MustNewConstMetric(workerSharesDesc, prometheus.GaugeValue, value, worker.label, window)
		}
		if worker.lastShareTimestamp != nil {
			ch <- prometheus.MustNewConstMetric(workerLastShareTimestampDesc, prometheus.GaugeValue, *worker.lastShareTimestamp, worker.label)
			ch <- prometheus.MustNewConstMetric(workerLastShareAgeDesc, prometheus.GaugeValue, m.clock.Now().Sub(time.Unix(int64(*worker.lastShareTimestamp), 0)).Seconds(), worker.label)
		}
	}
	ch <- prometheus.MustNewConstMetric(apiLastSuccessDesc, prometheus.GaugeValue, float64(snapshot.collectedAt.Unix()), workerEndpointWorkers)
	ch <- prometheus.MustNewConstMetric(dataAgeDesc, prometheus.GaugeValue, m.clock.Now().Sub(snapshot.collectedAt).Seconds(), workerEndpointWorkers)
}

var (
	workerStateDesc = prometheus.NewDesc(
		"braiins_pool_worker_state",
		"Worker state reported by Braiins Pool workers endpoint as a bounded one-hot value.",
		[]string{"worker", "state"},
		nil,
	)
	workerHashrateDesc = prometheus.NewDesc(
		"braiins_pool_worker_hashrate_ghs",
		"Worker hashrate reported by Braiins Pool workers endpoint in Gh/s.",
		[]string{"worker", "window"},
		nil,
	)
	workerSharesDesc = prometheus.NewDesc(
		"braiins_pool_worker_shares",
		"Worker rolling-window shares reported by Braiins Pool workers endpoint.",
		[]string{"worker", "window"},
		nil,
	)
	workerLastShareTimestampDesc = prometheus.NewDesc(
		"braiins_pool_worker_last_share_timestamp_seconds",
		"Unix timestamp of the worker's last accepted share reported by Braiins Pool.",
		[]string{"worker"},
		nil,
	)
	workerLastShareAgeDesc = prometheus.NewDesc(
		"braiins_pool_worker_last_share_age_seconds",
		"Age in seconds since the worker's last accepted share reported by Braiins Pool.",
		[]string{"worker"},
		nil,
	)
)

var errWorkerLimitExceeded = errors.New("Braiins worker snapshot exceeds configured worker limit")

func buildWorkerSnapshot(envelope braiins.CoinEnvelope[braiins.WorkersResponse], coin string, collectedAt time.Time, maxWorkers int) (*workerSnapshot, error) {
	response, ok := envelope[strings.ToLower(coin)]
	if !ok {
		return nil, errors.New("Braiins workers response missing configured coin")
	}
	if response.Workers == nil {
		response.Workers = map[string]braiins.Worker{}
	}
	if len(response.Workers) > maxWorkers {
		return nil, errWorkerLimitExceeded
	}
	labels := make(map[string]struct{}, len(response.Workers))
	samples := make([]workerSample, 0, len(response.Workers))
	names := make([]string, 0, len(response.Workers))
	for name := range response.Workers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		label, err := normalizeWorkerLabel(name)
		if err != nil {
			return nil, err
		}
		if _, exists := labels[label]; exists {
			return nil, errors.New("Braiins workers response contains duplicate normalized worker labels")
		}
		labels[label] = struct{}{}
		sample, err := buildWorkerSample(label, response.Workers[name])
		if err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	return &workerSnapshot{collectedAt: collectedAt, workers: samples}, nil
}

func buildWorkerSample(label string, worker braiins.Worker) (workerSample, error) {
	if unit := strings.TrimSpace(worker.HashRateUnit); unit != "" && unit != "Gh/s" {
		return workerSample{}, errors.New("Braiins worker has unsupported hashrate unit")
	}
	hashrates := map[string]float64{}
	for window, value := range map[string]braiins.Decimal{
		"5m":  worker.HashRate5m,
		"60m": worker.HashRate60m,
		"24h": worker.HashRate24h,
	} {
		if value == "" {
			continue
		}
		parsed, err := decimalToFloat(value)
		if err != nil {
			return workerSample{}, errors.New("Braiins worker has invalid hashrate value")
		}
		hashrates[window] = parsed
	}
	if worker.HashRateScoring != nil {
		parsed, err := decimalToFloat(*worker.HashRateScoring)
		if err != nil {
			return workerSample{}, errors.New("Braiins worker has invalid scoring hashrate value")
		}
		hashrates["scoring"] = parsed
	}
	shares := map[string]float64{}
	for window, value := range map[string]braiins.Decimal{
		"5m":  worker.Shares5m,
		"60m": worker.Shares60m,
		"24h": worker.Shares24h,
	} {
		if value == "" {
			continue
		}
		parsed, err := decimalToFloat(value)
		if err != nil {
			return workerSample{}, errors.New("Braiins worker has invalid shares value")
		}
		shares[window] = parsed
	}
	var lastShare *float64
	if worker.LastShare != nil {
		value := float64(*worker.LastShare)
		lastShare = &value
	}
	return workerSample{
		label:              label,
		state:              normalizeWorkerState(worker.State),
		hashrates:          hashrates,
		shares:             shares,
		lastShareTimestamp: lastShare,
	}, nil
}

func normalizeWorkerLabel(name string) (string, error) {
	label := strings.TrimSpace(name)
	if label == "" {
		return "", errors.New("Braiins worker label is blank")
	}
	if !utf8.ValidString(label) {
		return "", errors.New("Braiins worker label is not valid UTF-8")
	}
	if len(label) > maxWorkerLabelLength {
		return "", errors.New("Braiins worker label exceeds maximum length")
	}
	return label, nil
}

func normalizeWorkerState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "ok":
		return "ok"
	case "low":
		return "low"
	case "off":
		return "off"
	case "dis":
		return "dis"
	default:
		return "unknown"
	}
}
