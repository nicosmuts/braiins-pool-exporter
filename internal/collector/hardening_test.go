package collector

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
)

func TestAllCollectorsConcurrentPollAndGather(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(100, 0)}
	pool := newTestPoolMetrics(t, fakePoolClient{stats: []braiins.CoinEnvelope[braiins.PoolStats]{testPoolStats()}}, clock)
	account := newTestAccountMetrics(t, fakeAccountClient{profiles: []braiins.ProfileResponse{testProfile()}}, clock)
	worker := newTestWorkerMetrics(t, fakeWorkerClient{responses: []braiins.CoinEnvelope[braiins.WorkersResponse]{testWorkers("worker-a", "worker-b")}}, clock, 100)
	history := newTestHistoryMetrics(t, fakeHistoryClient{
		reward: testRewards(testReward(100, "0.1", "0.1")),
		payout: testPayouts(testPayout("confirmed", 10, 1, 100)),
	}, clock, 7, true, true)

	registry := prometheus.NewRegistry()
	RegisterPoolMetrics(registry, pool)
	RegisterAccountMetrics(registry, account)
	RegisterWorkerMetrics(registry, worker)
	RegisterHistoryMetrics(registry, history)

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 160)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = pool.Poll(ctx)
				_ = account.Poll(ctx)
				_ = worker.Poll(ctx)
				_ = history.Poll(ctx)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if _, err := registry.Gather(); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(fmt.Errorf("concurrent gather: %w", err))
		}
	}
	families := gatherFamilies(t, registry)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "pool_stats", "result": "success"}, 160)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "profile", "result": "success"}, 160)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "workers", "result": "success"}, 160)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "rewards", "result": "success"}, 160)
	assertMetric(t, families, "braiins_pool_api_requests_total", map[string]string{"endpoint": "payouts", "result": "success"}, 160)
}
