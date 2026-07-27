// Command braiins-pool-exporter exposes Prometheus metrics for Braiins Pool.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicosmuts/braiins-pool-exporter/internal/braiins"
	"github.com/nicosmuts/braiins-pool-exporter/internal/collector"
	"github.com/nicosmuts/braiins-pool-exporter/internal/config"
	"github.com/nicosmuts/braiins-pool-exporter/internal/server"
	"github.com/nicosmuts/braiins-pool-exporter/internal/version"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "braiins-pool-exporter:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load(args, config.Environment{})
	if err != nil {
		return err
	}
	logger := newLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	build := version.Current()
	registry, selfMetrics := collector.NewRegistry(build)
	var poolMetrics *collector.PoolMetrics
	var accountMetrics *collector.AccountMetrics
	var workerMetrics *collector.WorkerMetrics
	var historyMetrics *collector.HistoryMetrics
	if cfg.Token != "" {
		client, err := braiins.NewClient(braiins.Config{
			BaseURL: cfg.APIBaseURL,
			Token:   braiins.Secret(cfg.Token),
			Timeout: cfg.Timeout,
		})
		if err != nil {
			return err
		}
		poolMetrics, err = collector.NewPoolMetrics(collector.PoolOptions{
			Client: client,
			Coin:   cfg.Coin,
		})
		if err != nil {
			return err
		}
		collector.RegisterPoolMetrics(registry, poolMetrics)
		accountMetrics, err = collector.NewAccountMetrics(collector.AccountOptions{
			Client:       client,
			Coin:         cfg.Coin,
			PollInterval: cfg.PollInterval,
		})
		if err != nil {
			return err
		}
		collector.RegisterAccountMetrics(registry, accountMetrics)
		selfMetrics.RequireAccountReady(accountMetrics.Ready)
		if cfg.WorkerMetricsEnabled {
			workerMetrics, err = collector.NewWorkerMetrics(collector.WorkerOptions{
				Client:     client,
				Coin:       cfg.Coin,
				MaxWorkers: cfg.MaxWorkers,
			})
			if err != nil {
				return err
			}
			collector.RegisterWorkerMetrics(registry, workerMetrics)
		}
		if cfg.RewardsEnabled || cfg.PayoutsEnabled {
			historyMetrics, err = collector.NewHistoryMetrics(collector.HistoryOptions{
				Client:         client,
				Coin:           cfg.Coin,
				HistoryDays:    cfg.HistoryDays,
				RewardsEnabled: cfg.RewardsEnabled,
				PayoutsEnabled: cfg.PayoutsEnabled,
			})
			if err != nil {
				return err
			}
			collector.RegisterHistoryMetrics(registry, historyMetrics)
		}
	}
	app := server.New(cfg.ListenAddress, cfg.TelemetryPath, registry, selfMetrics, build)
	listener, err := app.Listen()
	if err != nil {
		return fmt.Errorf("listen on configured address: %w", err)
	}

	selfMetrics.SetReady(true)
	logger.Info("exporter initialized",
		"address", listener.Addr().String(),
		"telemetry_path", cfg.TelemetryPath,
		"version", build.Version,
		"config", cfg.Summary(),
	)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- app.Serve(listener)
	}()
	pollCtx, stopPolling := context.WithCancel(context.Background())
	defer stopPolling()
	pollDone := make(chan struct{})
	if accountMetrics != nil {
		steps := []pollStep{accountMetrics.Poll}
		if poolMetrics != nil {
			steps = append(steps, poolMetrics.Poll)
		}
		if workerMetrics != nil {
			steps = append(steps, workerMetrics.Poll)
		}
		if historyMetrics != nil && historyMetrics.RewardsEnabled() {
			steps = append(steps, historyMetrics.PollRewards)
		}
		if historyMetrics != nil && historyMetrics.PayoutsEnabled() {
			steps = append(steps, historyMetrics.PollPayouts)
		}
		go func() {
			defer close(pollDone)
			runPollSteps(pollCtx, cfg.PollInterval, steps)
		}()
	} else {
		close(pollDone)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case sig := <-signals:
		logger.Info("shutdown requested", "signal", sig.String())
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}

	selfMetrics.SetReady(false)
	stopPolling()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	select {
	case <-pollDone:
	case <-ctx.Done():
	}
	if err := app.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serveErr; err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	logger.Info("exporter stopped")
	return nil
}

type pollStep func(context.Context) error

func runPollSteps(ctx context.Context, interval time.Duration, steps []pollStep) {
	if interval <= 0 {
		interval = time.Minute
	}
	pollAll := func() {
		for index, step := range steps {
			_ = step(ctx)
			if index == len(steps)-1 {
				continue
			}
			timer := time.NewTimer(5 * time.Second)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}
		}
	}
	for {
		pollAll()
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func newLogger(level, format string) *slog.Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	options := &slog.HandlerOptions{Level: slogLevel}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, options))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, options))
}
