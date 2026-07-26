package braiins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizedFixturesDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		out  any
	}{
		{name: "profile", file: "profile_success.json", out: &ProfileResponse{}},
		{name: "workers", file: "workers_success.json", out: &CoinEnvelope[WorkersResponse]{}},
		{name: "rewards", file: "rewards_success.json", out: &CoinEnvelope[RewardsResponse]{}},
		{name: "payouts", file: "payouts_success.json", out: &PayoutsResponse{}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "braiins", tt.file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			if err := json.Unmarshal(data, tt.out); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
		})
	}
}

func TestWorkerStatesFromFixture(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "braiins", "workers_success.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var envelope CoinEnvelope[WorkersResponse]
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("decode workers: %v", err)
	}
	workers := envelope["btc"].Workers
	for _, name := range []string{"worker-a", "worker-b", "worker-offline"} {
		if _, ok := workers[name]; !ok {
			t.Fatalf("fixture missing worker %q", name)
		}
	}
	if workers["worker-a"].State != "ok" || workers["worker-b"].State != "low" || workers["worker-offline"].State != "off" {
		t.Fatalf("fixture states = %#v", workers)
	}
}
