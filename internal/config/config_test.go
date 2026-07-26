package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if cfg.ListenAddress != ":9108" {
		t.Fatalf("ListenAddress = %q, want :9108", cfg.ListenAddress)
	}
	if cfg.TelemetryPath != "/metrics" {
		t.Fatalf("TelemetryPath = %q, want /metrics", cfg.TelemetryPath)
	}
	if cfg.PollInterval != time.Minute {
		t.Fatalf("PollInterval = %s, want 1m", cfg.PollInterval)
	}
	if cfg.Coin != "btc" {
		t.Fatalf("Coin = %q, want btc", cfg.Coin)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{name: "listen address", args: []string{"--web.listen-address", "not-an-address"}},
		{name: "telemetry path", args: []string{"--web.telemetry-path", "metrics"}},
		{name: "reserved telemetry path", args: []string{"--web.telemetry-path", "/version"}},
		{name: "log level", args: []string{"--log.level", "verbose"}},
		{name: "log format", args: []string{"--log.format", "xml"}},
		{name: "conflicting token sources", env: map[string]string{
			"BRAIINS_POOL_TOKEN":      "secret-one",
			"BRAIINS_POOL_TOKEN_FILE": "token.txt",
		}},
		{name: "unsafe API URL", env: map[string]string{
			"BRAIINS_POOL_API_BASE_URL": "https://token@example.test/api?token=secret",
		}},
		{name: "invalid poll interval", env: map[string]string{
			"BRAIINS_POOL_POLL_INTERVAL": "soon",
		}},
		{name: "unverified coin", env: map[string]string{
			"BRAIINS_POOL_COIN": "bch",
		}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(tt.args, testEnvironment(tt.env, nil))
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if strings.Contains(err.Error(), "secret-one") {
				t.Fatal("Load() error contains token")
			}
		})
	}
}

func TestLoadTokenSourcesAndRedaction(t *testing.T) {
	t.Parallel()

	const token = "super-secret-token"
	tests := []struct {
		name       string
		env        map[string]string
		fileData   []byte
		wantSource string
	}{
		{
			name:       "environment",
			env:        map[string]string{"BRAIINS_POOL_TOKEN": token},
			wantSource: "environment",
		},
		{
			name:       "file",
			env:        map[string]string{"BRAIINS_POOL_TOKEN_FILE": "token.txt"},
			fileData:   []byte(token + "\n"),
			wantSource: "file",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(nil, testEnvironment(tt.env, tt.fileData))
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if string(cfg.Token) != token {
				t.Fatal("Load() did not retain token")
			}
			representations := []string{
				fmt.Sprint(cfg.Token),
				fmt.Sprintf("%#v", cfg.Token),
				fmt.Sprintf("%+v", cfg),
				fmt.Sprintf("%#v", cfg),
				fmt.Sprintf("%+v", cfg.Summary()),
			}
			for _, got := range representations {
				if strings.Contains(got, token) {
					t.Fatalf("formatted configuration contains token: %q", got)
				}
			}
			if cfg.Summary().TokenSource != tt.wantSource {
				t.Fatalf("TokenSource = %q, want %q", cfg.Summary().TokenSource, tt.wantSource)
			}
		})
	}
}

func TestLoadTokenFileErrorDoesNotExposeToken(t *testing.T) {
	t.Parallel()

	const token = "do-not-print-me"
	env := testEnvironment(map[string]string{"BRAIINS_POOL_TOKEN_FILE": "token.txt"}, nil)
	env.ReadFile = func(string) ([]byte, error) {
		return nil, errors.New("permission denied")
	}
	_, err := Load(nil, env)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatal("Load() error contains token")
	}
}

func testEnvironment(values map[string]string, fileData []byte) Environment {
	return Environment{
		LookupEnv: func(name string) (string, bool) {
			value, ok := values[name]
			return value, ok
		},
		ReadFile: func(string) ([]byte, error) {
			return fileData, nil
		},
	}
}
