package braiins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewRequestUsesDocumentedHeaderAndRedactsToken(t *testing.T) {
	t.Parallel()

	const token = "distinctive-secret-token"
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Token:   Secret(token),
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	req, err := client.NewRequest(context.Background(), Request{
		Endpoint: EndpointProfile,
		Coin:     "BTC",
	})
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if got := req.URL.String(); got != "https://pool.braiins.com/accounts/profile/json/btc" {
		t.Fatalf("URL = %q", got)
	}
	if got := req.Header.Get("Pool-Auth-Token"); got != token {
		t.Fatal("Pool-Auth-Token header was not set")
	}
	for _, formatted := range []string{fmt.Sprint(client.token), fmt.Sprintf("%#v", client.token), fmt.Sprintf("%+v", client)} {
		if strings.Contains(formatted, token) {
			t.Fatalf("formatted client leaked token: %q", formatted)
		}
	}
}

func TestRewardsAndPayoutsUseBoundedDateQueries(t *testing.T) {
	t.Parallel()

	var urls []string
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com/api",
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			urls = append(urls, req.URL.String())
			body := `{"btc":{"daily_rewards":[]}}`
			if strings.Contains(req.URL.Path, "/payouts/") {
				body = `{"onchain":[],"lightning":[]}`
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := client.Rewards(context.Background(), "BTC", "2026-07-20", "2026-07-26"); err != nil {
		t.Fatalf("Rewards() error = %v", err)
	}
	if _, err := client.Payouts(context.Background(), "BTC", "2026-07-20", "2026-07-26"); err != nil {
		t.Fatalf("Payouts() error = %v", err)
	}
	if urls[0] != "https://pool.braiins.com/api/accounts/rewards/json/btc?from=2026-07-20&to=2026-07-26" {
		t.Fatalf("Rewards URL = %q", urls[0])
	}
	if urls[1] != "https://pool.braiins.com/api/accounts/payouts/json/btc?from=2026-07-20&to=2026-07-26" {
		t.Fatalf("Payouts URL = %q", urls[1])
	}
}

func TestClientRejectsUnsafeBaseURLWithoutLeakingToken(t *testing.T) {
	t.Parallel()

	const token = "do-not-leak-client-token"
	_, err := NewClient(Config{
		BaseURL: "https://user:pass@example.test/api?token=" + token,
		Token:   Secret(token),
	})
	if err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked token: %v", err)
	}
}

func TestDoJSONRedactsTransportErrors(t *testing.T) {
	t.Parallel()

	const token = "transport-secret-token"
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Token:   Secret(token),
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("Pool-Auth-Token: " + token + " https://pool.braiins.com/?token=" + token)
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.DoJSON(context.Background(), Request{Endpoint: EndpointProfile, Coin: "btc"}, &ProfileResponse{})
	if err == nil {
		t.Fatal("DoJSON() error = nil, want error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("transport error leaked token: %v", err)
	}
	if !strings.Contains(err.Error(), "<redacted>") {
		t.Fatalf("transport error did not include redaction marker: %v", err)
	}
}

func TestDoJSONStatusErrorDoesNotExposeResponseBody(t *testing.T) {
	t.Parallel()

	const token = "body-secret-token"
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Token:   Secret(token),
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"detail":"` + token + `"}`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.DoJSON(context.Background(), Request{Endpoint: EndpointProfile, Coin: "btc"}, &ProfileResponse{})
	if err == nil {
		t.Fatal("DoJSON() error = nil, want error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("status error leaked response body: %v", err)
	}
}

func TestDoJSONDecodeErrorDoesNotExposeBodyValues(t *testing.T) {
	t.Parallel()

	const sensitive = "private-worker-name"
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"btc":{"workers":{"` + sensitive + `":`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.DoJSON(context.Background(), Request{Endpoint: EndpointWorkers, Coin: "btc"}, &CoinEnvelope[WorkersResponse]{})
	if err == nil {
		t.Fatal("DoJSON() error = nil, want error")
	}
	if strings.Contains(err.Error(), sensitive) {
		t.Fatalf("decode error leaked body value: %v", err)
	}
}

func TestDoJSONRetriesTransientServerError(t *testing.T) {
	t.Parallel()

	sleeper := &recordingSleeper{}
	calls := 0
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: time.Second,
			MaxBackoff:     5 * time.Second,
			Sleeper:        sleeper,
		},
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			if calls < 3 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`private body ignored`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"username":"example","btc":{"hash_rate_unit":"Gh/s"}}`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := client.DoJSON(context.Background(), Request{Endpoint: EndpointProfile, Coin: "btc"}, &ProfileResponse{}); err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if got := sleeper.delaysText(); got != "1s,2s" {
		t.Fatalf("retry delays = %q, want 1s,2s", got)
	}
}

func TestDoJSONRetryExhaustionReturnsBoundedError(t *testing.T) {
	t.Parallel()

	const secret = "secret-response-body"
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: time.Second,
			MaxBackoff:     5 * time.Second,
			Sleeper:        &recordingSleeper{},
		},
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader(secret)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.DoJSON(context.Background(), Request{Endpoint: EndpointProfile, Coin: "btc"}, &ProfileResponse{})
	if err == nil {
		t.Fatal("DoJSON() error = nil, want error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("retry exhaustion leaked body: %v", err)
	}
	var status StatusError
	if !errors.As(err, &status) || status.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error = %T %[1]v, want StatusError 503", err)
	}
}

func TestDoJSONCancellationDuringBackoffStopsRetries(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	sleeper := cancelingSleeper{cancel: cancel}
	calls := 0
	client, err := NewClient(Config{
		BaseURL: "https://pool.braiins.com",
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: time.Second,
			MaxBackoff:     5 * time.Second,
			Sleeper:        sleeper,
		},
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`ignored`)),
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.DoJSON(ctx, Request{Endpoint: EndpointProfile, Coin: "btc"}, &ProfileResponse{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DoJSON() error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDoJSONRateLimitRetryAfterHandling(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		header string
		want   string
	}{
		"seconds":   {header: "2", want: "2s"},
		"capped":    {header: "60", want: "5s"},
		"malformed": {header: "soon", want: "5s"},
		"missing":   {header: "", want: "5s"},
		"date":      {header: "Sun, 26 Jul 2026 12:00:03 GMT", want: "3s"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sleeper := &recordingSleeper{}
			calls := 0
			client, err := NewClient(Config{
				BaseURL: "https://pool.braiins.com",
				Retry: RetryPolicy{
					MaxAttempts:      2,
					MaxBackoff:       5 * time.Second,
					RateLimitBackoff: 5 * time.Second,
					Sleeper:          sleeper,
					Now:              func() time.Time { return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC) },
				},
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						header := http.Header{}
						if tt.header != "" {
							header.Set("Retry-After", tt.header)
						}
						return &http.Response{
							StatusCode: http.StatusTooManyRequests,
							Header:     header,
							Body:       io.NopCloser(strings.NewReader(`ignored`)),
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"username":"example","btc":{"hash_rate_unit":"Gh/s"}}`)),
					}, nil
				}),
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			if err := client.DoJSON(context.Background(), Request{Endpoint: EndpointProfile, Coin: "btc"}, &ProfileResponse{}); err != nil {
				t.Fatalf("DoJSON() error = %v", err)
			}
			if got := sleeper.delaysText(); got != tt.want {
				t.Fatalf("retry delay = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecimalPreservesJSONNumberAndStringPrecision(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		`"0.00000001"`:           "0.00000001",
		`5727000000.74660415488`: "5727000000.74660415488",
	}
	for input, want := range tests {
		var got Decimal
		if err := got.UnmarshalJSON([]byte(input)); err != nil {
			t.Fatalf("UnmarshalJSON(%s) error = %v", input, err)
		}
		if got.String() != want {
			t.Fatalf("Decimal = %q, want %q", got, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type recordingSleeper struct {
	delays []time.Duration
}

func (s *recordingSleeper) Sleep(_ context.Context, delay time.Duration) error {
	s.delays = append(s.delays, delay)
	return nil
}

func (s *recordingSleeper) delaysText() string {
	values := make([]string, 0, len(s.delays))
	for _, delay := range s.delays {
		values = append(values, delay.String())
	}
	return strings.Join(values, ",")
}

type cancelingSleeper struct {
	cancel context.CancelFunc
}

func (s cancelingSleeper) Sleep(ctx context.Context, _ time.Duration) error {
	s.cancel()
	return ctx.Err()
}
