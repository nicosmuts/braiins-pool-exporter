// Package collector owns Prometheus collectors exposed by the exporter.
package collector

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nicosmuts/braiins-pool-exporter/internal/version"
)

// SelfMetrics owns exporter build and readiness metrics.
type SelfMetrics struct {
	ready atomic.Bool
}

// NewRegistry creates an isolated registry with Go, process, and exporter
// self-metrics. Braiins account metrics are intentionally absent in Milestone
// 00.
func NewRegistry(build version.Info) (*prometheus.Registry, *SelfMetrics) {
	registry := prometheus.NewRegistry()
	self := &SelfMetrics{}

	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "braiins_pool",
		Subsystem: "exporter",
		Name:      "build_info",
		Help:      "A metric with a constant value of 1 labeled by build information.",
	}, []string{"version", "commit", "build_date", "go_version"})
	buildInfo.WithLabelValues(build.Version, build.Commit, build.BuildDate, build.GoVersion).Set(1)

	ready := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "braiins_pool",
		Subsystem: "exporter",
		Name:      "ready",
		Help:      "Whether the exporter is initialized and ready to serve.",
	}, func() float64 {
		if self.ready.Load() {
			return 1
		}
		return 0
	})

	registry.MustRegister(
		buildInfo,
		ready,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	return registry, self
}

// SetReady updates the exporter readiness state.
func (m *SelfMetrics) SetReady(ready bool) {
	m.ready.Store(ready)
}

// Ready reports the exporter readiness state.
func (m *SelfMetrics) Ready() bool {
	return m.ready.Load()
}
