// Package server provides the exporter's HTTP interface and lifecycle.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nicosmuts/braiins-pool-exporter/internal/version"
)

const (
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 60 * time.Second
)

// Readiness reports whether initialization has completed.
type Readiness interface {
	Ready() bool
}

// App owns the HTTP server.
type App struct {
	httpServer *http.Server
}

// New constructs an HTTP server without opening a listener.
func New(address, telemetryPath string, registry *prometheus.Registry, readiness Readiness, build version.Info) *App {
	mux := http.NewServeMux()
	mux.Handle(telemetryPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("/-/ready", func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusOK
		body := map[string]string{"status": "ready"}
		if !readiness.Ready() {
			status = http.StatusServiceUnavailable
			body["status"] = "not_ready"
		}
		writeJSON(w, status, body)
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, build)
	})

	return &App{
		httpServer: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

// Listen opens the configured TCP listener.
func (a *App) Listen() (net.Listener, error) {
	listener, err := net.Listen("tcp", a.httpServer.Addr)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// Serve handles requests until Shutdown is called.
func (a *App) Serve(listener net.Listener) error {
	err := a.httpServer.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops accepting requests and waits for active requests.
func (a *App) Shutdown(ctx context.Context) error {
	return a.httpServer.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
