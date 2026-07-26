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
