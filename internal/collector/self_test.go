package collector

import (
	"testing"

	"github.com/nicosmuts/braiins-pool-exporter/internal/version"
)

func TestNewRegistryIncludesSelfMetrics(t *testing.T) {
	t.Parallel()

	registry, self := NewRegistry(version.Info{
		Version:   "test",
		Commit:    "abc123",
		BuildDate: "2026-07-26T00:00:00Z",
		GoVersion: "go1.test",
	})
	self.SetReady(true)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	found := map[string]bool{}
	for _, family := range families {
		found[family.GetName()] = true
	}
	for _, name := range []string{
		"braiins_pool_exporter_build_info",
		"braiins_pool_exporter_ready",
		"go_goroutines",
		"process_cpu_seconds_total",
	} {
		if !found[name] {
			t.Errorf("metric family %q not found", name)
		}
	}
}
