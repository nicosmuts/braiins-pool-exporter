# Braiins Pool Exporter

<p align="center">
  <img src="assets/banner-image.png" width="100%" alt="Braiins Pool Exporter architecture from miner to Braiins Pool, Prometheus, and Grafana">
</p>

<p align="center">
  <a href="https://github.com/nicosmuts/braiins-pool-exporter/blob/main/go.mod"><img alt="Go version" src="https://img.shields.io/badge/go-1.26-blue"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/nicosmuts/braiins-pool-exporter"></a>
  <a href="https://github.com/nicosmuts/braiins-pool-exporter/commits/main"><img alt="Last commit" src="https://img.shields.io/github/last-commit/nicosmuts/braiins-pool-exporter"></a>
  <a href="https://github.com/nicosmuts/braiins-pool-exporter/issues"><img alt="Open issues" src="https://img.shields.io/github/issues/nicosmuts/braiins-pool-exporter"></a>
</p>

<p align="center">
  <img alt="Prometheus exporter" src="https://img.shields.io/badge/Prometheus-exporter-orange">
  <img alt="Grafana dashboard" src="https://img.shields.io/badge/Grafana-dashboard-f46800">
  <img alt="Docker planned" src="https://img.shields.io/badge/Docker-planned-2496ed">
  <img alt="Docker Compose planned" src="https://img.shields.io/badge/Docker%20Compose-planned-2496ed">
  <img alt="Braiins Pool API" src="https://img.shields.io/badge/Braiins%20Pool-API-black">
  <img alt="Go" src="https://img.shields.io/badge/Go-00add8">
</p>

Braiins Pool Exporter is an independent Prometheus exporter for users of the
official Braiins Pool API. It polls verified Braiins Pool account, worker,
reward, and payout endpoints, normalizes the responses into stable Prometheus
metrics, and serves them from a small Go HTTP process. Prometheus scrapes the
exporter rather than the pool API directly, while the included Grafana
dashboard provides a reusable default view of the exported metrics.

Data flows from Avalon miners to Braiins Pool, then through Braiins Pool
Exporter into Prometheus and Grafana. The exporter keeps deployment-specific
scrape configuration, worker mappings, credentials, and production dashboards
outside the public repository.

## Project status

- Active development.
- Official API contract verified for the currently implemented collectors.
- Default Grafana dashboard available.
- Stable releases are not yet published.
- Docker image, Docker Compose development stack, and release automation are
  planned.

## Features

### Account metrics

- Account hashrate windows.
- Current account balance.
- Account worker counts by normalized state.
- Account freshness and last-success timestamps.

### Worker metrics

- Worker state as bounded one-hot gauges.
- Worker hashrate windows.
- Worker share windows.
- Worker last-share timestamp and age.
- Bounded worker label cardinality.

### Rewards and payouts

- Bounded reward aggregation by safe reward component.
- Bounded payout amount and fee aggregation by rail and status.
- Precision-safe BTC reward accounting.
- Integer satoshi payout accounting.
- No reward dates, payout destinations, or transaction identifiers as labels.

### Exporter

- Prometheus, Go runtime, process, build, and readiness metrics.
- `/metrics`, `/-/healthy`, `/-/ready`, and `/version` endpoints.
- Environment- or file-based token loading with redaction safeguards.
- Bounded polling, retries, backoff, rate-limit handling, and caching.
- Stale-but-visible last-known-good data.
- Graceful shutdown and offline unit tests.

### Grafana

- Importable default dashboard at
  [`grafana/braiins-pool-exporter.json`](grafana/braiins-pool-exporter.json).
- Stable dashboard UID `braiins-pool-exporter`.
- Prometheus datasource variable.
- Portable job, instance, and optional worker filters.
- Account, worker, rewards, payouts, API health, and freshness panels.

## Quick start

### Requirements

- Go 1.26.4.
- Prometheus for scraping the exporter.
- Grafana 10.4 or newer for the included dashboard.
- GNU Make is optional.

### Clone

```sh
git clone https://github.com/nicosmuts/braiins-pool-exporter.git
cd braiins-pool-exporter
```

### Build

```sh
go mod download
go build -o bin/braiins-pool-exporter ./cmd/braiins-pool-exporter
```

### Run

Without a Braiins token, the exporter starts without making external network
requests and exposes only self-metrics.

```sh
go run ./cmd/braiins-pool-exporter
```

The default listen address is `:9108`.

```sh
curl http://localhost:9108/-/healthy
curl http://localhost:9108/-/ready
curl http://localhost:9108/version
curl http://localhost:9108/metrics
```

### Configure Prometheus

```yaml
scrape_configs:
  - job_name: braiins-pool-exporter
    static_configs:
      - targets: ["localhost:9108"]
```

### Import the Grafana dashboard

Import [`grafana/braiins-pool-exporter.json`](grafana/braiins-pool-exporter.json)
and select your Prometheus datasource for `DS_PROMETHEUS`.

## Configuration

The exporter requires exactly one Braiins token source when API-derived metrics
are enabled.

### Required for Braiins API polling

| Setting | Default | Description |
|---|---:|---|
| `BRAIINS_POOL_TOKEN` | unset | Braiins API token value. |
| `BRAIINS_POOL_TOKEN_FILE` | unset | Path to a file containing the Braiins API token. |

Set only one of `BRAIINS_POOL_TOKEN` or `BRAIINS_POOL_TOKEN_FILE`. Token command
line flags are intentionally unsupported.

### Optional environment variables

| Setting | Default | Description |
|---|---:|---|
| `BRAIINS_POOL_COIN` | `btc` | Coin selector. Only `btc` is currently verified and accepted. |
| `BRAIINS_POOL_API_BASE_URL` | official API origin | Override for tests or compatible endpoints. |
| `BRAIINS_POOL_POLL_INTERVAL` | `1m` | Poll interval. |
| `BRAIINS_POOL_TIMEOUT` | `10s` | Per-request HTTP timeout. |
| `BRAIINS_POOL_WORKER_METRICS_ENABLED` | `true` | Enable worker metrics when a token is configured. |
| `BRAIINS_POOL_MAX_WORKERS` | `100` | Maximum accepted workers per snapshot. |
| `BRAIINS_POOL_REWARDS_ENABLED` | `true` | Enable bounded rewards metrics. |
| `BRAIINS_POOL_PAYOUTS_ENABLED` | `true` | Enable bounded payout metrics. |
| `BRAIINS_POOL_HISTORY_DAYS` | `7` | Inclusive rewards and payouts history window, capped at 90 days. |

### Command-line flags

| Flag | Default | Description |
|---|---:|---|
| `--web.listen-address` | `:9108` | HTTP listen address. |
| `--web.telemetry-path` | `/metrics` | Metrics path. |
| `--log.level` | `info` | Log level: `debug`, `info`, `warn`, or `error`. |
| `--log.format` | `text` | Log format: `text` or `json`. |
| `--config.file` | empty | Reserved until a safe configuration format is defined. |

Worker labels use Braiins API worker names. Treat the exporter HTTP interface
as private operational telemetry if worker names reveal internal conventions.

## Metrics

Metric names, labels, units, and behavior are documented in
[`docs/METRICS.md`](docs/METRICS.md). The dashboard and tests use that metric
contract as the source of truth.

## Documentation

- [API discovery](docs/API_DISCOVERY.md)
- [Metrics](docs/METRICS.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Security design](docs/SECURITY.md)
- [Development](docs/DEVELOPMENT.md)
- [Grafana dashboard](grafana/README.md)
- [Contributing](CONTRIBUTING.md)

## Roadmap

Completed:

- API verification.
- Account metrics.
- Worker metrics.
- Rewards and payouts.
- Exporter hardening.
- Default Grafana dashboard.

Planned:

- Docker image.
- Docker Compose development stack.
- Release automation.
- First stable release.

## Contributing

Contributions are welcome. Please read [`CONTRIBUTING.md`](CONTRIBUTING.md)
before opening issues or pull requests.

## Security

Never provide a wallet private key or seed phrase to this exporter. Report
vulnerabilities according to [`SECURITY.md`](SECURITY.md).

## License

Licensed under the [Apache License 2.0](LICENSE).

Braiins is a trademark of its respective owner. This independent project is
not affiliated with or endorsed by Braiins.
