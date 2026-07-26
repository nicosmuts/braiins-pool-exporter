package braiins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://pool.braiins.com"
	UserAgent      = "braiins-pool-exporter"

	maxResponseBytes = 1 << 20
)

// Secret prevents API tokens from being exposed by common formatting.
type Secret string

// String redacts the secret.
func (Secret) String() string { return "<redacted>" }

// GoString redacts the secret in Go-syntax formatting.
func (Secret) GoString() string { return "<redacted>" }

// RoundTripper executes HTTP requests. It matches http.RoundTripper and keeps
// tests independent from the network.
type RoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

// Client is a small context-aware API boundary for verified Braiins Pool
// endpoints.
type Client struct {
	baseURL *url.URL
	token   Secret
	client  *http.Client
}

// String returns a non-sensitive client summary.
func (c *Client) String() string {
	if c == nil {
		return "<nil>"
	}
	host := ""
	if c.baseURL != nil {
		host = c.baseURL.Host
	}
	return fmt.Sprintf("BraiinsClient{base_host:%s token:%s}", host, Secret(""))
}

// GoString returns a non-sensitive client summary for Go-syntax formatting.
func (c *Client) GoString() string { return c.String() }

// Config configures a Client.
type Config struct {
	BaseURL   string
	Token     Secret
	Timeout   time.Duration
	Transport RoundTripper
}

// NewClient constructs a Client without making network calls.
func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("Braiins API base URL must be absolute")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Braiins API base URL must not contain credentials, a query, or a fragment")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("Braiins API base URL must use HTTP or HTTPS")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := cfg.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{
		baseURL: parsed,
		token:   cfg.Token,
		client:  &http.Client{Timeout: timeout, Transport: transport},
	}, nil
}

// Endpoint identifies a verified Braiins Pool API path.
type Endpoint string

const (
	EndpointPoolStats     Endpoint = "/stats/json/{coin}"
	EndpointProfile       Endpoint = "/accounts/profile/json/{coin}"
	EndpointRewards       Endpoint = "/accounts/rewards/json/{coin}"
	EndpointDailyHashrate Endpoint = "/accounts/hash_rate_daily/json/{group}/{coin}"
	EndpointBlockRewards  Endpoint = "/accounts/block_rewards/json/{coin}"
	EndpointWorkers       Endpoint = "/accounts/workers/json/{coin}"
	EndpointPayouts       Endpoint = "/accounts/payouts/json/{coin}"
)

// Request describes a read-only API request.
type Request struct {
	Endpoint Endpoint
	Coin     string
	Group    string
	FromDate string
	ToDate   string
}

// NewRequest builds an authenticated GET request without executing it.
func (c *Client) NewRequest(ctx context.Context, req Request) (*http.Request, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	coin := normalizeCoin(req.Coin)
	if coin == "" {
		return nil, errors.New("Braiins API coin selector is required")
	}
	relative, err := endpointPath(req.Endpoint, coin, req.Group)
	if err != nil {
		return nil, err
	}
	u := *c.baseURL
	u.Path = joinURLPath(c.baseURL.Path, relative)
	q := u.Query()
	if req.FromDate != "" {
		q.Set("from", req.FromDate)
	}
	if req.ToDate != "" {
		q.Set("to", req.ToDate)
	}
	u.RawQuery = q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("build Braiins API request")
	}
	httpReq.Header.Set("User-Agent", UserAgent)
	if c.token != "" {
		httpReq.Header.Set("Pool-Auth-Token", string(c.token))
	}
	return httpReq, nil
}

// DoJSON executes a read-only request and decodes a JSON response.
func (c *Client) DoJSON(ctx context.Context, req Request, out any) error {
	httpReq, err := c.NewRequest(ctx, req)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Braiins API request failed: %w", safeTransportError(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read Braiins API response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return errors.New("Braiins API response exceeds size limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return StatusError{StatusCode: resp.StatusCode, ContentType: resp.Header.Get("Content-Type")}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode Braiins API JSON response: %w", err)
	}
	return nil
}

// StatusError describes a non-2xx response without exposing a response body.
type StatusError struct {
	StatusCode  int
	ContentType string
}

func (e StatusError) Error() string {
	if e.ContentType == "" {
		return fmt.Sprintf("Braiins API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("Braiins API returned HTTP %d (%s)", e.StatusCode, e.ContentType)
}

func endpointPath(endpoint Endpoint, coin, group string) (string, error) {
	switch endpoint {
	case EndpointPoolStats:
		return strings.ReplaceAll(string(endpoint), "{coin}", coin), nil
	case EndpointProfile, EndpointRewards, EndpointBlockRewards, EndpointWorkers, EndpointPayouts:
		return strings.ReplaceAll(string(endpoint), "{coin}", coin), nil
	case EndpointDailyHashrate:
		group = strings.TrimSpace(group)
		if group == "" {
			return "", errors.New("Braiins daily hashrate group selector is required")
		}
		relative := strings.ReplaceAll(string(endpoint), "{group}", path.Clean(group))
		return strings.ReplaceAll(relative, "{coin}", coin), nil
	default:
		return "", errors.New("unknown Braiins API endpoint")
	}
}

func normalizeCoin(coin string) string {
	return strings.ToLower(strings.TrimSpace(coin))
}

func joinURLPath(basePath, relative string) string {
	basePath = strings.TrimRight(basePath, "/")
	relative = strings.TrimLeft(relative, "/")
	if basePath == "" {
		return "/" + relative
	}
	return basePath + "/" + relative
}

func safeTransportError(err error) error {
	if err == nil {
		return nil
	}
	text := err.Error()
	text = redactHeaderValue(text)
	text = redactQuerySecrets(text)
	return errors.New(text)
}

func redactHeaderValue(text string) string {
	for _, name := range []string{"Pool-Auth-Token", "X-Pool-Auth-Token", "Authorization"} {
		searchFrom := 0
		for searchFrom < len(text) {
			idx := strings.Index(strings.ToLower(text[searchFrom:]), strings.ToLower(name))
			if idx < 0 {
				break
			}
			idx += searchFrom
			valueStart := idx + len(name)
			for valueStart < len(text) && strings.ContainsRune(":= \t", rune(text[valueStart])) {
				valueStart++
			}
			end := valueStart
			for end < len(text) && !strings.ContainsRune(" \t\r\n,;", rune(text[end])) {
				end++
			}
			if valueStart >= len(text) {
				text = text[:idx] + name + "=<redacted>"
				searchFrom = len(text)
				break
			}
			text = text[:idx] + name + "=<redacted>" + text[end:]
			searchFrom = idx + len(name) + len("=<redacted>")
		}
	}
	return text
}

func redactQuerySecrets(text string) string {
	for _, key := range []string{"token=", "auth=", "api_key=", "apikey="} {
		searchFrom := 0
		for searchFrom < len(text) {
			idx := strings.Index(strings.ToLower(text[searchFrom:]), key)
			if idx < 0 {
				break
			}
			idx += searchFrom
			end := idx + len(key)
			for end < len(text) && !strings.ContainsRune("& \t\r\n", rune(text[end])) {
				end++
			}
			text = text[:idx+len(key)] + "<redacted>" + text[end:]
			searchFrom = idx + len(key) + len("<redacted>")
		}
	}
	return text
}
