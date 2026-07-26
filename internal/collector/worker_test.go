package collector

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

func TestWorkerMetricsExposeApprovedMetrics(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1785081000, 0)}
	metrics := newTestWorkerMetrics(t, fakeWorkerClient{
		responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{testWorkers("worker-a", "worker-b")},
	}, clock, 100)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}

	registry := prometheus.NewRegistry()
	RegisterWorkerMetrics(registry, metrics)
	families := gatherFamilies(t, registry)

	assertMetric(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "ok"}, 1)
	assertMetric(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "low"}, 0)
	assertMetric(t, families, "braiins_pool_worker_hashrate_ghs", map[string]string{"worker": "worker-a", "window": "5m"}, 14977)
	assertMetric(t, families, "braiins_pool_worker_hashrate_ghs", map[string]string{"worker": "worker-a", "window": "60m"}, 15302)
	assertMetric(t, families, "braiins_pool_worker_hashrate_ghs", map[string]string{"worker": "worker-a", "window": "24h"}, 15351)
	assertMetric(t, families, "braiins_pool_worker_hashrate_ghs", map[string]string{"worker": "worker-b", "window": "scoring"}, 12000)
	assertMetric(t, families, "braiins_pool_worker_shares", map[string]string{"worker": "worker-a", "window": "5m"}, 150)
	assertMetric(t, families, "braiins_pool_worker_last_share_timestamp_seconds", map[string]string{"worker": "worker-a"}, 1542103204)
	assertMetric(t, families, "braiins_pool_worker_last_share_age_seconds", map[string]string{"worker": "worker-a"}, float64(clock.now.Unix()-1542103204))
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_last_success_timestamp_seconds", map[string]string{"endpoint": "workers"}, float64(clock.now.Unix()))
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "workers"}, 0)
}

func TestWorkerMetricsOrderingIndependent(t *testing.T) {
	t.Parallel()

	left := gatherWorkerMetricText(t, testWorkers("worker-a", "worker-b", "worker-c"))
	right := gatherWorkerMetricText(t, testWorkers("worker-c", "worker-b", "worker-a"))
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("metric output differs by API order\nleft=%v\nright=%v", left, right)
	}
}

func TestWorkerMetricsNoPollDuringGather(t *testing.T) {
	t.Parallel()

	client := &countingWorkerClient{response: testWorkers("worker-a")}
	metrics := newTestWorkerMetrics(t, client, &fakeClock{now: time.Unix(1, 0)}, 100)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	registry := prometheus.NewRegistry()
	RegisterWorkerMetrics(registry, metrics)
	_ = gatherFamilies(t, registry)
	_ = gatherFamilies(t, registry)
	if got := client.calls; got != 1 {
		t.Fatalf("Workers calls after gathers = %d, want 1", got)
	}
}

func TestWorkerMetricsUnknownStateUsesBoundedLabel(t *testing.T) {
	t.Parallel()

	response := testWorkers("worker-a")
	worker := response["btc"].Workers["worker-a"]
	worker.State = "surprising"
	response["btc"].Workers["worker-a"] = worker
	metrics := pollWorkerMetrics(t, response, &fakeClock{now: time.Unix(1, 0)}, 100)
	families := gatherRegisteredWorkerFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "unknown"}, 1)
	if hasLabelValue(families["braiins_pool_worker_state"], "state", "surprising") {
		t.Fatal("raw unknown state was emitted as a label")
	}
}

func TestWorkerMetricsOptionalFieldsAreOmitted(t *testing.T) {
	t.Parallel()

	response := testWorkers("worker-a")
	worker := response["btc"].Workers["worker-a"]
	worker.HashRateScoring = nil
	worker.HashRate24h = ""
	worker.Shares24h = ""
	worker.LastShare = nil
	response["btc"].Workers["worker-a"] = worker
	metrics := pollWorkerMetrics(t, response, &fakeClock{now: time.Unix(1, 0)}, 100)
	families := gatherRegisteredWorkerFamilies(t, metrics)
	assertMetricAbsent(t, families, "braiins_pool_worker_hashrate_ghs", map[string]string{"worker": "worker-a", "window": "scoring"})
	assertMetricAbsent(t, families, "braiins_pool_worker_hashrate_ghs", map[string]string{"worker": "worker-a", "window": "24h"})
	assertMetricAbsent(t, families, "braiins_pool_worker_shares", map[string]string{"worker": "worker-a", "window": "24h"})
	assertMetricAbsent(t, families, "braiins_pool_worker_last_share_timestamp_seconds", map[string]string{"worker": "worker-a"})
}

func TestWorkerMetricsRejectMalformedAndDuplicateLabels(t *testing.T) {
	t.Parallel()

	tests := map[string]braiins.CoinEnvelope[braiins.WorkersResponse]{
		"blank": func() braiins.CoinEnvelope[braiins.WorkersResponse] {
			return testWorkers(" ")
		}(),
		"duplicate trimmed": func() braiins.CoinEnvelope[braiins.WorkersResponse] {
			response := testWorkers("worker-a")
			response["btc"].Workers[" worker-a "] = testWorker("low")
			return response
		}(),
		"invalid decimal": func() braiins.CoinEnvelope[braiins.WorkersResponse] {
			response := testWorkers("worker-a")
			worker := response["btc"].Workers["worker-a"]
			worker.HashRate5m = "bad"
			response["btc"].Workers["worker-a"] = worker
			return response
		}(),
	}
	for name, response := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			metrics := newTestWorkerMetrics(t, fakeWorkerClient{responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{response}}, &fakeClock{now: time.Unix(1, 0)}, 100)
			if err := metrics.Poll(context.Background()); err == nil {
				t.Fatal("Poll() error = nil, want error")
			}
			if metrics.Ready() {
				t.Fatal("Ready() = true, want false after malformed first response")
			}
		})
	}
}

func TestWorkerMetricsCardinalityLimitPreservesLastGood(t *testing.T) {
	t.Parallel()

	client := &sequenceWorkerClient{
		responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{
			testWorkers("worker-a"),
			testWorkers("worker-a", "worker-b"),
		},
	}
	metrics := newTestWorkerMetrics(t, client, &fakeClock{now: time.Unix(1, 0)}, 1)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("first Poll() error = %v", err)
	}
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("second Poll() error = nil, want limit error")
	}
	families := gatherRegisteredWorkerFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "ok"}, 1)
	assertMetricAbsent(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-b", "state": "low"})
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "limit_exceeded"}, 1)
}

func TestWorkerMetricsFirstFailureAndTransientFailure(t *testing.T) {
	t.Parallel()

	firstFail := newTestWorkerMetrics(t, fakeWorkerClient{err: context.DeadlineExceeded}, &fakeClock{now: time.Unix(1, 0)}, 100)
	if err := firstFail.Poll(context.Background()); err == nil {
		t.Fatal("first failure Poll() error = nil, want error")
	}
	if firstFail.Ready() {
		t.Fatal("Ready() = true before first worker snapshot")
	}
	firstFamilies := gatherRegisteredWorkerFamilies(t, firstFail)
	assertMetricAbsent(t, firstFamilies, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "ok"})
	assertMetric(t, firstFamilies, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "timeout"}, 1)

	clock := &fakeClock{now: time.Unix(100, 0)}
	client := &sequenceWorkerClient{
		responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{testWorkers("worker-a")},
		errors:    []error{context.Canceled},
	}
	metrics := newTestWorkerMetrics(t, client, clock, 100)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("success Poll() error = %v", err)
	}
	clock.now = time.Unix(130, 0)
	if err := metrics.Poll(context.Background()); err == nil {
		t.Fatal("transient failure Poll() error = nil, want error")
	}
	families := gatherRegisteredWorkerFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "ok"}, 1)
	assertMetric(t, families, "braiins_pool_data_age_seconds", map[string]string{"endpoint": "workers"}, 30)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "canceled"}, 1)
}

func TestWorkerMetricsDisappearanceAndEmptySuccessReplaceSnapshot(t *testing.T) {
	t.Parallel()

	client := &sequenceWorkerClient{
		responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{
			testWorkers("worker-a", "worker-b"),
			testWorkers("worker-a"),
			{"btc": {Workers: map[string]braiins.Worker{}}},
		},
	}
	metrics := newTestWorkerMetrics(t, client, &fakeClock{now: time.Unix(1, 0)}, 100)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("first Poll() error = %v", err)
	}
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("second Poll() error = %v", err)
	}
	families := gatherRegisteredWorkerFamilies(t, metrics)
	assertMetric(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "ok"}, 1)
	assertMetricAbsent(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-b", "state": "low"})
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("empty Poll() error = %v", err)
	}
	families = gatherRegisteredWorkerFamilies(t, metrics)
	assertMetricAbsent(t, families, "braiins_pool_worker_state", map[string]string{"worker": "worker-a", "state": "ok"})
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "success"}, 3)
}

func TestWorkerMetricsDescriptorConsistencyAndConcurrentGather(t *testing.T) {
	t.Parallel()

	metrics := pollWorkerMetrics(t, testWorkers("worker-a", "worker-b"), &fakeClock{now: time.Unix(1, 0)}, 100)
	registry := prometheus.NewRegistry()
	RegisterWorkerMetrics(registry, metrics)
	first := gatherFamilies(t, registry)
	second := gatherFamilies(t, registry)
	if !reflect.DeepEqual(metricFamilyNames(first), metricFamilyNames(second)) {
		t.Fatalf("metric family names changed between gathers")
	}
	done := make(chan struct{}, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_ = gatherFamilies(t, registry)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

func TestAccountAndWorkerMetricsRegisterTogether(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(100, 0)}
	account := newTestAccountMetrics(t, fakeAccountClient{profiles: []braiins.ProfileResponse{testProfile()}}, clock)
	worker := newTestWorkerMetrics(t, fakeWorkerClient{responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{testWorkers("worker-a")}}, clock, 100)
	if err := account.Poll(context.Background()); err != nil {
		t.Fatalf("account Poll() error = %v", err)
	}
	if err := worker.Poll(context.Background()); err != nil {
		t.Fatalf("worker Poll() error = %v", err)
	}
	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, account)
	RegisterWorkerMetrics(registry, worker)
	families := gatherFamilies(t, registry)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "success"}, 1)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "success"}, 1)
}

func TestWorkerPollErrorDoesNotLeakTokenOrPrivateWorkerName(t *testing.T) {
	t.Parallel()

	const privateWorker = "private-worker-name"
	metrics := newTestWorkerMetrics(t, fakeWorkerClient{
		err: errors.New("transport failed for " + privateWorker),
	}, &fakeClock{now: time.Unix(1, 0)}, 100)
	err := metrics.Poll(context.Background())
	if err == nil {
		t.Fatal("Poll() error = nil, want error")
	}
	if strings.Contains(err.Error(), privateWorker) {
		t.Fatalf("Poll() leaked private worker name: %v", err)
	}
}

func gatherRegisteredWorkerFamilies(t *testing.T, metrics *WorkerMetrics) map[string]*dto.MetricFamily {
	t.Helper()
	registry := prometheus.NewRegistry()
	RegisterWorkerMetrics(registry, metrics)
	return gatherFamilies(t, registry)
}

func assertMetricAbsent(t *testing.T, families map[string]*dto.MetricFamily, name string, labels map[string]string) {
	t.Helper()
	family, ok := families[name]
	if !ok {
		return
	}
	for _, metric := range family.Metric {
		if labelsMatch(metric, labels) {
			t.Fatalf("metric %q with labels %v is present, want absent", name, labels)
		}
	}
}

func hasLabelValue(family *dto.MetricFamily, labelName, labelValue string) bool {
	if family == nil {
		return false
	}
	for _, metric := range family.Metric {
		for _, label := range metric.Label {
			if label.GetName() == labelName && label.GetValue() == labelValue {
				return true
			}
		}
	}
	return false
}

func metricFamilyNames(families map[string]*dto.MetricFamily) []string {
	names := make([]string, 0, len(families))
	for name := range families {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func fmtFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func pollWorkerMetrics(t *testing.T, response braiins.CoinEnvelope[braiins.WorkersResponse], clock Clock, maxWorkers int) *WorkerMetrics {
	t.Helper()
	metrics := newTestWorkerMetrics(t, fakeWorkerClient{responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{response}}, clock, maxWorkers)
	if err := metrics.Poll(context.Background()); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	return metrics
}

func newTestWorkerMetrics(t *testing.T, client WorkerClient, clock Clock, maxWorkers int) *WorkerMetrics {
	t.Helper()
	metrics, err := NewWorkerMetrics(WorkerOptions{Client: client, Coin: "btc", Clock: clock, MaxWorkers: maxWorkers})
	if err != nil {
		t.Fatalf("NewWorkerMetrics() error = %v", err)
	}
	return metrics
}

func gatherWorkerMetricText(t *testing.T, response braiins.CoinEnvelope[braiins.WorkersResponse]) []string {
	t.Helper()
	metrics := pollWorkerMetrics(t, response, &fakeClock{now: time.Unix(1, 0)}, 100)
	families := gatherRegisteredWorkerFamilies(t, metrics)
	lines := []string{}
	for _, name := range metricFamilyNames(families) {
		if !strings.HasPrefix(name, "braiins_pool_worker_") {
			continue
		}
		family := families[name]
		for _, metric := range family.Metric {
			labels := make([]string, 0, len(metric.Label))
			for _, label := range metric.Label {
				labels = append(labels, label.GetName()+"="+label.GetValue())
			}
			lines = append(lines, name+" "+strings.Join(labels, ",")+" "+fmtFloat(metricValue(metric)))
		}
	}
	sort.Strings(lines)
	return lines
}

func testWorkers(names ...string) braiins.CoinEnvelope[braiins.WorkersResponse] {
	workers := make(map[string]braiins.Worker, len(names))
	for i, name := range names {
		state := "ok"
		if i%2 == 1 {
			state = "low"
		}
		workers[name] = testWorker(state)
	}
	return braiins.CoinEnvelope[braiins.WorkersResponse]{"btc": {Workers: workers}}
}

func testWorker(state string) braiins.Worker {
	lastShare := int64(1542103204)
	scoring := braiins.Decimal("12000")
	return braiins.Worker{
		State:           state,
		LastShare:       &lastShare,
		HashRateUnit:    "Gh/s",
		HashRateScoring: &scoring,
		HashRate5m:      "14977",
		HashRate60m:     "15302",
		HashRate24h:     "15351",
		Shares5m:        "150",
		Shares60m:       "1750",
		Shares24h:       "42000",
	}
}

type fakeWorkerClient struct {
	responses []braiins.CoinEnvelope[braiins.WorkersResponse]
	err       error
}

func (c fakeWorkerClient) Workers(context.Context, string) (braiins.CoinEnvelope[braiins.WorkersResponse], error) {
	if c.err != nil {
		return nil, c.err
	}
	if len(c.responses) == 0 {
		return testWorkers("worker-a"), nil
	}
	return c.responses[0], nil
}

type sequenceWorkerClient struct {
	responses []braiins.CoinEnvelope[braiins.WorkersResponse]
	errors    []error
	calls     int
}

func (c *sequenceWorkerClient) Workers(context.Context, string) (braiins.CoinEnvelope[braiins.WorkersResponse], error) {
	call := c.calls
	c.calls++
	if call < len(c.responses) {
		return c.responses[call], nil
	}
	errIndex := call - len(c.responses)
	if errIndex < len(c.errors) {
		return nil, c.errors[errIndex]
	}
	return nil, errors.New("unexpected worker call")
}

type countingWorkerClient struct {
	response braiins.CoinEnvelope[braiins.WorkersResponse]
	calls    int
}

func (c *countingWorkerClient) Workers(context.Context, string) (braiins.CoinEnvelope[braiins.WorkersResponse], error) {
	c.calls++
	return c.response, nil
}
