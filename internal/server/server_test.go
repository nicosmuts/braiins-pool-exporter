package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nicosmuts/braiins-pool-exporter/internal/collector"
	"github.com/nicosmuts/braiins-pool-exporter/internal/version"
)

func TestEndpoints(t *testing.T) {
	t.Parallel()

	build := version.Info{
		Version:   "test-version",
		Commit:    "test-commit",
		BuildDate: "2026-07-26T00:00:00Z",
		GoVersion: "go-test",
	}
	registry, self := collector.NewRegistry(build)
	self.SetReady(true)
	app := New(":0", "/metrics", registry, self, build)
	testServer := httptest.NewServer(app.httpServer.Handler)
	t.Cleanup(testServer.Close)

	tests := []struct {
		path        string
		status      int
		contentType string
		contains    string
	}{
		{path: "/-/healthy", status: http.StatusOK, contentType: "application/json", contains: `"status":"healthy"`},
		{path: "/-/ready", status: http.StatusOK, contentType: "application/json", contains: `"status":"ready"`},
		{path: "/version", status: http.StatusOK, contentType: "application/json", contains: `"version":"test-version"`},
		{path: "/metrics", status: http.StatusOK, contentType: "text/plain", contains: "braiins_pool_exporter_build_info"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			response, err := testServer.Client().Get(testServer.URL + tt.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tt.path, err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			if response.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", response.StatusCode, tt.status)
			}
			if !strings.Contains(response.Header.Get("Content-Type"), tt.contentType) {
				t.Errorf("Content-Type = %q, want to contain %q", response.Header.Get("Content-Type"), tt.contentType)
			}
			if !strings.Contains(string(body), tt.contains) {
				t.Errorf("body does not contain %q: %s", tt.contains, body)
			}
		})
	}
}

func TestNotReady(t *testing.T) {
	t.Parallel()

	build := version.Current()
	registry, self := collector.NewRegistry(build)
	app := New(":0", "/metrics", registry, self, build)
	request := httptest.NewRequest(http.MethodGet, "/-/ready", nil)
	recorder := httptest.NewRecorder()

	app.httpServer.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestGracefulStartupAndShutdown(t *testing.T) {
	t.Parallel()

	build := version.Current()
	registry, self := collector.NewRegistry(build)
	self.SetReady(true)
	app := New("127.0.0.1:0", "/metrics", registry, self, build)
	listener, err := app.Listen()
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	errs := make(chan error, 1)
	go func() {
		errs <- app.Serve(listener)
	}()

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + listener.Addr().String() + "/-/healthy")
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	response.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-errs; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
}
