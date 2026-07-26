# Braiins Pool Exporter Agent Guide

## Objective

Build a small, reusable Prometheus exporter for authoritative data from the
official Braiins Pool API. This is an independent open-source project and must
remain safe to publish.

## Architecture and ownership

- `cmd/braiins-pool-exporter`: process wiring, signals, and logging only.
- `internal/config`: immutable startup configuration and secret loading.
- `internal/braiins`: official API client and verified wire types.
- `internal/collector`: deterministic Prometheus collectors and cache state.
- `internal/server`: HTTP endpoints and graceful lifecycle.
- `internal/version`: build metadata.
- `docs`: architecture, security, metrics, roadmap, and handoff records.
- `grafana`: only the reusable dashboard produced from verified metrics.
- `testdata`: sanitized, synthetic, or explicitly reviewed fixtures.

Do not add `pkg/` without a demonstrated external consumer. Avoid global
mutable state, framework-heavy designs, deep package trees, and premature
interfaces.

## Coding standards

- Use the standard library where practical, including `net/http`, `log/slog`,
  and `context`.
- Give every network client an explicit timeout and propagate cancellation.
- Inject API clients and clocks where testing requires it.
- Keep runtime configuration immutable after startup.
- Wrap errors with safe context; never include response bodies by default.
- Use race-safe state, bounded caches, and deterministic metric descriptors.
- Write table-driven unit tests and GoDoc for exported identifiers.
- Format with `gofmt`; run `go vet`, unit tests, race tests, and builds.

## Metrics

- Public Braiins metrics use the `braiins_pool_` prefix.
- Exporter self-metrics use `braiins_pool_exporter_`.
- Metric names, help text, type, unit, and label sets form a compatibility
  contract after release.
- Labels must be bounded. Never label by token, wallet or payout address,
  transaction/reward ID, arbitrary error text, full URL, or timestamp.
- Do not implement an API-derived metric until official fields and semantics
  have been documented with sanitized fixtures.
- Do not encode historical dates as labels on samples emitted at scrape time.

## Security constraints

- Never log, format, return, panic with, or attach a Braiins token to a URL.
- Never accept a token through a command-line flag.
- Never require wallet private keys or seed phrases.
- Never commit live API responses until identifiers and private financial
  details have been removed and the fixture has been reviewed.
- Sanitize URL logging and API errors. Keep tokens out of metric labels.
- Do not scrape the Braiins website or automate a browser.

## Scope constraints

- The public exporter does not query or configure mining devices.
- Do not add environment-specific IP addresses, worker mappings, Kubernetes
  assumptions, device telemetry, price feeds, wallet monitoring, or
  profitability logic to public defaults.
- Operator-managed deployment configuration and composite production
  dashboards live outside public exporter defaults and in later milestones.

## Tests and validation

Run:

```sh
go mod tidy
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/braiins-pool-exporter
golangci-lint run # when installed
git diff --check
```

Tests must not need a live Braiins token or external network. Any API fixture
must be sanitized and documented.

## Commits and releases

- Follow Conventional Commits, keep changes focused, and update the changelog.
- Do not stage, commit, push, tag, release, change visibility, or deploy without
  the explicit approval required by the parent workspace policy.
- Use immutable version and image identifiers for releases.
- Review dependency licenses, security documentation, and public content
  before changing repository visibility.
