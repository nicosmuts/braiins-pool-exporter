package collector

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

func BenchmarkAccountCollectorCollect(b *testing.B) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	metrics := newBenchmarkAccountMetrics(b, clock)
	registry := prometheus.NewRegistry()
	RegisterAccountMetrics(registry, metrics)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := registry.Gather(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWorkerCollectorCollect(b *testing.B) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	metrics := newBenchmarkWorkerMetrics(b, clock, 100)
	registry := prometheus.NewRegistry()
	RegisterWorkerMetrics(registry, metrics)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := registry.Gather(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHistoryCollectorCollect(b *testing.B) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	metrics := newBenchmarkHistoryMetrics(b, clock)
	registry := prometheus.NewRegistry()
	RegisterHistoryMetrics(registry, metrics)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := registry.Gather(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWorkerSnapshotNormalization(b *testing.B) {
	envelope := testWorkers(makeWorkerNames(100)...)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := buildWorkerSnapshot(envelope, "btc", time.Unix(100, 0), 100); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRewardsAndPayoutsAggregation(b *testing.B) {
	rewards := make([]braiins.DailyReward, 90)
	for i := range rewards {
		rewards[i] = testReward(100+int64(i), "0.00000001", "0.00000001")
	}
	payouts := make([]braiins.Payout, 100)
	for i := range payouts {
		payouts[i] = testPayout("confirmed", 1, 0, 100+int64(i))
	}
	rewardEnvelope := testRewards(rewards...)
	payoutResponse := testPayouts(payouts...)
	start := time.Unix(0, 0)
	end := time.Unix(10000, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := buildRewardSnapshot(rewardEnvelope, "btc", time.Unix(100, 0), start, end); err != nil {
			b.Fatal(err)
		}
		if _, err := buildPayoutSnapshot(payoutResponse, time.Unix(100, 0), start, end); err != nil {
			b.Fatal(err)
		}
	}
}

func newBenchmarkAccountMetrics(b *testing.B, clock Clock) *AccountMetrics {
	b.Helper()
	metrics, err := NewAccountMetrics(AccountOptions{Client: fakeAccountClient{profiles: []braiins.ProfileResponse{testProfile()}}, Coin: "btc", Clock: clock})
	if err != nil {
		b.Fatal(err)
	}
	if err := metrics.Poll(context.Background()); err != nil {
		b.Fatal(err)
	}
	return metrics
}

func newBenchmarkWorkerMetrics(b *testing.B, clock Clock, count int) *WorkerMetrics {
	b.Helper()
	metrics, err := NewWorkerMetrics(WorkerOptions{
		Client:     fakeWorkerClient{responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{testWorkers(makeWorkerNames(count)...)}},
		Coin:       "btc",
		Clock:      clock,
		MaxWorkers: count,
	})
	if err != nil {
		b.Fatal(err)
	}
	if err := metrics.Poll(context.Background()); err != nil {
		b.Fatal(err)
	}
	return metrics
}

func newBenchmarkHistoryMetrics(b *testing.B, clock Clock) *HistoryMetrics {
	b.Helper()
	metrics, err := NewHistoryMetrics(HistoryOptions{
		Client: fakeHistoryClient{
			reward: testRewards(testReward(100, "0.1", "0.1")),
			payout: testPayouts(testPayout("confirmed", 10, 1, 100)),
		},
		Coin:           "btc",
		Clock:          clock,
		HistoryDays:    7,
		RewardsEnabled: true,
		PayoutsEnabled: true,
	})
	if err != nil {
		b.Fatal(err)
	}
	if err := metrics.Poll(context.Background()); err != nil {
		b.Fatal(err)
	}
	return metrics
}

func makeWorkerNames(count int) []string {
	names := make([]string, count)
	for i := range names {
		names[i] = "worker-" + strconv.Itoa(i)
	}
	return names
}
