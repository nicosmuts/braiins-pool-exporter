# Braiins Pool Exporter

Braiins Pool Exporter is a lightweight Prometheus exporter for authoritative
data from the official Braiins Pool API.

> [!IMPORTANT]
> This project is under active development. No stable release exists, and the
> Milestone 04 implements bounded rewards and payouts. Dashboard, container,
> release, and deployment work remain future milestones.

This independent exporter is designed for Braiins Pool users and keeps
environment-specific addresses, worker mappings, credentials, and dashboards
outside the public project.

## Current scope

The exporter currently provides:

- a small Go service using `net/http` and `log/slog`;
- Prometheus, Go runtime, process, build, and readiness metrics;
- health, readiness, and sanitized version endpoints;
- environment- or file-based secret loading with redaction safeguards;
- optional Braiins account profile polling when a token is configured;
- verified account hashrate, balance, worker-count, request, and freshness
  metrics;
- optional Braiins worker polling with bounded per-worker state, hashrate,
  shares, last-share, request, and freshness metrics;
- optional bounded rewards and payouts polling with precision-safe BTC reward
  aggregation, satoshi payout aggregation, request, and freshness metrics;
- graceful shutdown and unit tests.

Milestone 01 records the documented official API contract in
[docs/API_DISCOVERY.md](docs/API_DISCOVERY.md). Implemented and deferred
metrics are documented in [docs/METRICS.md](docs/METRICS.md).

## Quick start

Prerequisites:

- Go 1.26.4 (recorded in `.go-version` and `go.mod`);
- GNU Make for Makefile targets (optional on Windows).

```sh
go mod download
go run ./cmd/braiins-pool-exporter
```

The exporter listens on `:9108` by default. Without a Braiins token it exposes
only self-metrics and makes no external network requests.

```sh
curl http://localhost:9108/-/healthy
curl http://localhost:9108/-/ready
curl http://localhost:9108/version
curl http://localhost:9108/metrics
```

## HTTP endpoints

| Endpoint | Purpose |
|---|---|
| `/metrics` | Prometheus exposition |
| `/-/healthy` | Process liveness; independent of Braiins Pool |
| `/-/ready` | Initialization readiness |
| `/version` | Sanitized version, commit, build date, and Go version |

Foundation readiness means local configuration is valid and the HTTP service
has initialized. Its semantics must be revisited when polling is added.

## Configuration

Command-line flags:

| Flag | Default | Purpose |
|---|---|---|
| `--web.listen-address` | `:9108` | HTTP listen address |
| `--web.telemetry-path` | `/metrics` | Metrics path |
| `--log.level` | `info` | `debug`, `info`, `warn`, or `error` |
| `--log.format` | `text` | `text` or `json` |
| `--config.file` | empty | Reserved until a safe format is defined |

Account collection is enabled by setting exactly one Braiins token source:

| Environment variable | Purpose |
|---|---|
| `BRAIINS_POOL_TOKEN` | Token value |
| `BRAIINS_POOL_TOKEN_FILE` | Path to a mounted token file |
| `BRAIINS_POOL_COIN` | Coin selector; only `btc` is currently verified and accepted |
| `BRAIINS_POOL_API_BASE_URL` | Override for tests or compatible endpoints |
| `BRAIINS_POOL_POLL_INTERVAL` | Poll interval, default `1m` |
| `BRAIINS_POOL_TIMEOUT` | HTTP timeout, default `10s` |
| `BRAIINS_POOL_WORKER_METRICS_ENABLED` | Enable worker metrics when a token is configured, default `true` |
| `BRAIINS_POOL_MAX_WORKERS` | Maximum accepted workers per snapshot, default `100` |
| `BRAIINS_POOL_REWARDS_ENABLED` | Enable bounded rewards metrics when a token is configured, default `true` |
| `BRAIINS_POOL_PAYOUTS_ENABLED` | Enable bounded payout metrics when a token is configured, default `true` |
| `BRAIINS_POOL_HISTORY_DAYS` | Inclusive rewards/payouts date window, default `7`, maximum `90` |

Set only one token source. Command-line token flags are intentionally
unsupported because process listings can expose their values. For containers
and Kubernetes, prefer a read-only mounted Secret file. Tokens are never metric
labels and must never be logged.

See [examples/README.md](examples/README.md) and
[docs/SECURITY.md](docs/SECURITY.md) for safe usage.

When account collection is enabled, readiness requires the first successful
profile poll. Later transient failures keep the last-known-good account
snapshot visible while `braiins_pool_data_age_seconds` increases.

Worker metrics use the Braiins API worker name as the `worker` label. Worker
names may be private operational identifiers; keep the exporter HTTP interface
private and use `BRAIINS_POOL_WORKER_METRICS_ENABLED=false` if direct worker
labels are not acceptable for an environment. Worker freshness is independent
from account freshness and does not block readiness.

Rewards and payouts use one bounded date-window request per endpoint. BTC
reward values are aggregated as exact decimals and converted to `float64` only
for Prometheus exposition. Payout amounts and fees remain integer satoshis.
No reward dates, payout destinations, transaction IDs, Lightning invoices,
preimages, account names, or event identifiers are exported as labels.
Rewards and payouts have independent freshness and do not block readiness.

## Development

```sh
make help
make check
```

Equivalent commands are documented in
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). The Makefile `lint` target requires
`golangci-lint` to be installed separately.

## Roadmap and dashboard

- [docs/ROADMAP.md](docs/ROADMAP.md) defines Milestones 00 through 09.
- [docs/API_DISCOVERY.md](docs/API_DISCOVERY.md) records the Milestone 01 API
  evidence and unknowns.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) records component boundaries.
- [grafana/README.md](grafana/README.md) reserves the reusable dashboard
  identity without querying speculative metrics.

Production deployment, device telemetry, Bitcoin prices, wallet monitoring,
and profitability calculations are separate integration concerns.

## Security

Never provide a wallet private key or seed phrase to this exporter. Report
vulnerabilities according to [SECURITY.md](SECURITY.md).

## License

Licensed under the [Apache License 2.0](LICENSE). Apache-2.0 was selected for a
Prometheus-ecosystem project because it is permissive and includes an explicit
patent grant. This project deliberately adopts Apache-2.0 for public use and
contribution.

Braiins is a trademark of its respective owner. This independent project is
not affiliated with or endorsed by Braiins.
