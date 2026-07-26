# Braiins Pool Exporter

Braiins Pool Exporter is a lightweight Prometheus exporter for authoritative
data from the official Braiins Pool API.

> [!IMPORTANT]
> This project is under active development. No stable release exists, and the
> Braiins API contract is documented for Milestone 01, but API-derived
> collectors have not started. The running exporter exposes only self-metrics.

This independent exporter is designed for Braiins Pool users and keeps
environment-specific addresses, worker mappings, credentials, and dashboards
outside the public project.

## Current scope

Milestone 00 provides:

- a small Go service using `net/http` and `log/slog`;
- Prometheus, Go runtime, process, build, and readiness metrics;
- health, readiness, and sanitized version endpoints;
- environment- or file-based secret loading with redaction safeguards;
- graceful shutdown and unit tests.

Milestone 01 records the documented official API contract in
[docs/API_DISCOVERY.md](docs/API_DISCOVERY.md). Future milestones will use
that contract before implementing account, worker, reward, or payout metrics.
Proposed metrics are documented in [docs/METRICS.md](docs/METRICS.md), but
they are not implemented yet.

## Quick start

Prerequisites:

- Go 1.26.4 (recorded in `.go-version` and `go.mod`);
- GNU Make for Makefile targets (optional on Windows).

```sh
go mod download
go run ./cmd/braiins-pool-exporter
```

The exporter listens on `:9108` by default. No Braiins token or external
network access is needed in Milestone 00.

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

The future Braiins client configuration contract is:

| Environment variable | Purpose |
|---|---|
| `BRAIINS_POOL_TOKEN` | Token value |
| `BRAIINS_POOL_TOKEN_FILE` | Path to a mounted token file |
| `BRAIINS_POOL_COIN` | Coin selector, pending API discovery |
| `BRAIINS_POOL_API_BASE_URL` | Override for tests or compatible endpoints |
| `BRAIINS_POOL_POLL_INTERVAL` | Poll interval, default `1m` |
| `BRAIINS_POOL_TIMEOUT` | HTTP timeout, default `10s` |

Set only one token source. Command-line token flags are intentionally
unsupported because process listings can expose their values. For containers
and Kubernetes, prefer a read-only mounted Secret file. Tokens are never metric
labels and must never be logged.

See [examples/README.md](examples/README.md) and
[docs/SECURITY.md](docs/SECURITY.md) for safe usage.

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
