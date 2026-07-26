// Package config parses and validates immutable process configuration.
package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenAddress = ":9108"
	defaultTelemetryPath = "/metrics"
	defaultCoin          = "btc"
	defaultPollInterval  = time.Minute
	defaultTimeout       = 10 * time.Second
	defaultMaxWorkers    = 100
	defaultHistoryDays   = 7
	maxHistoryDays       = 90
)

// Secret prevents an API token from being exposed by common formatting.
type Secret string

// String redacts the secret.
func (Secret) String() string { return "<redacted>" }

// GoString redacts the secret in Go-syntax formatting.
func (Secret) GoString() string { return "<redacted>" }

// Config is immutable after startup and safe to pass by value.
type Config struct {
	ListenAddress string
	TelemetryPath string
	LogLevel      string
	LogFormat     string
	ConfigFile    string

	Token        Secret
	TokenFile    string
	Coin         string
	APIBaseURL   string
	PollInterval time.Duration
	Timeout      time.Duration

	WorkerMetricsEnabled bool
	MaxWorkers           int
	RewardsEnabled       bool
	PayoutsEnabled       bool
	HistoryDays          int
}

// SafeSummary is a deliberately non-sensitive view suitable for logs.
type SafeSummary struct {
	ListenAddress string
	TelemetryPath string
	LogLevel      string
	LogFormat     string
	TokenSource   string
	Coin          string
	PollInterval  time.Duration
	Timeout       time.Duration
	WorkerMetrics bool
	MaxWorkers    int
	Rewards       bool
	Payouts       bool
	HistoryDays   int
}

// Environment provides dependencies used when loading environment-backed
// configuration.
type Environment struct {
	LookupEnv func(string) (string, bool)
	ReadFile  func(string) ([]byte, error)
}

// Default returns the foundation defaults without reading the environment.
func Default() Config {
	return Config{
		ListenAddress:        defaultListenAddress,
		TelemetryPath:        defaultTelemetryPath,
		LogLevel:             "info",
		LogFormat:            "text",
		Coin:                 defaultCoin,
		PollInterval:         defaultPollInterval,
		Timeout:              defaultTimeout,
		WorkerMetricsEnabled: true,
		MaxWorkers:           defaultMaxWorkers,
		RewardsEnabled:       true,
		PayoutsEnabled:       true,
		HistoryDays:          defaultHistoryDays,
	}
}

// Load parses flags, reads supported environment variables, and validates the
// resulting configuration. A real Braiins token is optional in Milestone 00.
func Load(args []string, env Environment) (Config, error) {
	if env.LookupEnv == nil {
		env.LookupEnv = os.LookupEnv
	}
	if env.ReadFile == nil {
		env.ReadFile = os.ReadFile
	}

	cfg := Default()
	flags := flag.NewFlagSet("braiins-pool-exporter", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&cfg.ListenAddress, "web.listen-address", cfg.ListenAddress, "Address on which to expose metrics and web interface.")
	flags.StringVar(&cfg.TelemetryPath, "web.telemetry-path", cfg.TelemetryPath, "Path under which to expose metrics.")
	flags.StringVar(&cfg.LogLevel, "log.level", cfg.LogLevel, "Log level: debug, info, warn, or error.")
	flags.StringVar(&cfg.LogFormat, "log.format", cfg.LogFormat, "Log format: text or json.")
	flags.StringVar(&cfg.ConfigFile, "config.file", "", "Reserved path for a future structured configuration file.")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments")
	}
	if cfg.ConfigFile != "" {
		return Config{}, errors.New("--config.file is reserved until its format is defined")
	}

	token, tokenSet := env.LookupEnv("BRAIINS_POOL_TOKEN")
	tokenFile, tokenFileSet := env.LookupEnv("BRAIINS_POOL_TOKEN_FILE")
	if tokenSet && strings.TrimSpace(token) != "" && tokenFileSet && strings.TrimSpace(tokenFile) != "" {
		return Config{}, errors.New("conflicting Braiins token sources: set only BRAIINS_POOL_TOKEN or BRAIINS_POOL_TOKEN_FILE")
	}
	if tokenSet && strings.TrimSpace(token) != "" {
		cfg.Token = Secret(strings.TrimSpace(token))
	}
	if tokenFileSet && strings.TrimSpace(tokenFile) != "" {
		cfg.TokenFile = strings.TrimSpace(tokenFile)
		data, err := env.ReadFile(cfg.TokenFile)
		if err != nil {
			return Config{}, fmt.Errorf("read Braiins token file: %w", err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return Config{}, errors.New("Braiins token file is empty")
		}
		cfg.Token = Secret(value)
	}

	if coin := envValue(env.LookupEnv, "BRAIINS_POOL_COIN"); coin != "" {
		cfg.Coin = strings.ToLower(coin)
	}
	cfg.APIBaseURL = envValue(env.LookupEnv, "BRAIINS_POOL_API_BASE_URL")

	var err error
	cfg.PollInterval, err = durationEnv(env.LookupEnv, "BRAIINS_POOL_POLL_INTERVAL", cfg.PollInterval)
	if err != nil {
		return Config{}, err
	}
	cfg.Timeout, err = durationEnv(env.LookupEnv, "BRAIINS_POOL_TIMEOUT", cfg.Timeout)
	if err != nil {
		return Config{}, err
	}
	cfg.WorkerMetricsEnabled, err = boolEnv(env.LookupEnv, "BRAIINS_POOL_WORKER_METRICS_ENABLED", cfg.WorkerMetricsEnabled)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxWorkers, err = intEnv(env.LookupEnv, "BRAIINS_POOL_MAX_WORKERS", cfg.MaxWorkers)
	if err != nil {
		return Config{}, err
	}
	cfg.RewardsEnabled, err = boolEnv(env.LookupEnv, "BRAIINS_POOL_REWARDS_ENABLED", cfg.RewardsEnabled)
	if err != nil {
		return Config{}, err
	}
	cfg.PayoutsEnabled, err = boolEnv(env.LookupEnv, "BRAIINS_POOL_PAYOUTS_ENABLED", cfg.PayoutsEnabled)
	if err != nil {
		return Config{}, err
	}
	cfg.HistoryDays, err = intEnv(env.LookupEnv, "BRAIINS_POOL_HISTORY_DAYS", cfg.HistoryDays)
	if err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks local configuration without making network calls.
func (c Config) Validate() error {
	if err := validateListenAddress(c.ListenAddress); err != nil {
		return err
	}
	if !strings.HasPrefix(c.TelemetryPath, "/") || strings.ContainsAny(c.TelemetryPath, "?#") {
		return errors.New("web telemetry path must be an absolute URL path without a query or fragment")
	}
	switch c.TelemetryPath {
	case "/-/healthy", "/-/ready", "/version":
		return errors.New("web telemetry path conflicts with a reserved endpoint")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("log level must be debug, info, warn, or error")
	}
	switch c.LogFormat {
	case "text", "json":
	default:
		return errors.New("log format must be text or json")
	}
	if c.PollInterval <= 0 {
		return errors.New("Braiins poll interval must be positive")
	}
	if c.Timeout <= 0 {
		return errors.New("Braiins request timeout must be positive")
	}
	if c.MaxWorkers <= 0 {
		return errors.New("Braiins max workers must be positive")
	}
	if c.HistoryDays <= 0 || c.HistoryDays > maxHistoryDays {
		return fmt.Errorf("Braiins history window must be between 1 and %d days", maxHistoryDays)
	}
	if strings.ToLower(strings.TrimSpace(c.Coin)) != "btc" {
		return errors.New("Braiins coin selector must be btc")
	}
	if c.APIBaseURL != "" {
		parsed, err := url.Parse(c.APIBaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errors.New("Braiins API base URL must be an absolute URL")
		}
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return errors.New("Braiins API base URL must use HTTP or HTTPS")
		}
		if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("Braiins API base URL must not contain credentials, a query, or a fragment")
		}
	}
	return nil
}

// Summary returns a view that never contains the token value or token path.
func (c Config) Summary() SafeSummary {
	source := "none"
	if c.TokenFile != "" {
		source = "file"
	} else if c.Token != "" {
		source = "environment"
	}
	return SafeSummary{
		ListenAddress: c.ListenAddress,
		TelemetryPath: c.TelemetryPath,
		LogLevel:      c.LogLevel,
		LogFormat:     c.LogFormat,
		TokenSource:   source,
		Coin:          c.Coin,
		PollInterval:  c.PollInterval,
		Timeout:       c.Timeout,
		WorkerMetrics: c.WorkerMetricsEnabled,
		MaxWorkers:    c.MaxWorkers,
		Rewards:       c.RewardsEnabled,
		Payouts:       c.PayoutsEnabled,
		HistoryDays:   c.HistoryDays,
	}
}

func envValue(lookup func(string) (string, bool), name string) string {
	value, _ := lookup(name)
	return strings.TrimSpace(value)
}

func durationEnv(lookup func(string) (string, bool), name string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration", name)
	}
	return parsed, nil
}

func boolEnv(lookup func(string) (string, bool), name string, fallback bool) (bool, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return parsed, nil
}

func intEnv(lookup func(string) (string, bool), name string, fallback int) (int, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return parsed, nil
}

func validateListenAddress(address string) error {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("web listen address must be in host:port form: %w", err)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 0 || number > 65535 {
		return errors.New("web listen address contains an invalid port")
	}
	return nil
}
