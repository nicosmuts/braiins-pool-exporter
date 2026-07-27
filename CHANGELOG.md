# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project intends to follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) once releases begin.

## [Unreleased]

### Added

- Optional Docker Compose `miner` profile for the upstream
  `avalonhome-prometheus-exporter` image, with local AvalonMiner defaults,
  separate miner-mode Prometheus scrape configuration, and documentation of
  known AvalonMiner 1047 metric gaps before Issue #11 dashboard work.

### Changed

- Default Braiins polling cadence is now 10 seconds, and the Grafana dashboard
  refreshes at 10 seconds for local operational validation.
- Braiins dashboard hashrate panels now keep exporter metrics in Gh/s while
  displaying pool hashrate as EH/s and account/worker hashrate as TH/s.
- Braiins dashboard timestamp panels now convert Unix seconds for Grafana date
  rendering, and the worker status table now joins current worker state,
  last-share, and hashrate fields from instant Prometheus queries.

## [0.0.1] - 2026-07-27

Initial public development release. This release is suitable for public review,
local validation, and early integration testing, but it is not a stable
production declaration. Interfaces, configuration behavior, image metadata, and
the Prometheus metric contract may still change before `v0.1.0`.

### Added

- Account metrics for account hashrate windows, balance, worker counts,
  account API freshness, and readiness-aware first snapshot behavior.
- Worker metrics with bounded worker labels, normalized worker states,
  hashrate/share windows, last-share timestamps, and independent worker API
  freshness.
- Reward and payout summaries with bounded history windows, exact BTC reward
  aggregation, integer satoshi payout aggregation, safe labels, and no payout
  addresses, transaction identifiers, or event timestamps as labels.
- Hardened polling with serialized poll cycles, bounded retries/backoff,
  `Retry-After` handling for HTTP 429, redirect refusal, cancellation-aware
  waits, bounded public error categories, capped reward/payout records, and
  stale-but-visible cache behavior.
- Default Grafana dashboard for exporter readiness, API health/freshness,
  account, worker, reward, and payout metrics.
- Production-oriented Docker image and Docker Compose stack for exporter,
  Prometheus, and Grafana local validation.
- CI, Docker build validation, Linux race testing, Dependabot dependency
  tracking, tag-gated release automation, and container SBOM/provenance
  publishing.
- Pre-release repository hardening with Dependabot for Go modules, GitHub
  Actions, Dockerfile, and Docker Compose dependencies; improved GitHub issue
  and pull request templates; workflow concurrency; and a documented first
  release process.
- Go toolchain and container build base updated to 1.26.5, with the runtime
  image hardened to distroless non-root after vulnerability review.
- Milestone 07 container and release engineering with a production-oriented
  multi-stage Dockerfile, Docker Compose development stack, Prometheus and
  Grafana provisioning, `.env.example`, local secret-file handling, CI
  validation workflow, and tag-gated GHCR/GitHub Release workflow.
- Milestone 06 default Grafana dashboard with stable UID
  `braiins-pool-exporter`, title `Braiins Pool Exporter`, a Prometheus
  datasource variable, portable job/instance/worker filters, account, worker,
  reward, payout, API health, and freshness panels, plus static dashboard
  validation.
- Milestone 05 exporter hardening with bounded retry/backoff, 429
  `Retry-After` handling, redirect refusal, deterministic poll serialization,
  bounded error categories, reward/payout record caps, concurrent scrape
  tests, synthetic benchmarks, race validation in Docker, security review
  notes, and documented staleness behavior.
- Milestone 04 rewards and payouts collector with bounded date-window
  summaries, exact decimal BTC reward aggregation, integer satoshi payout
  aggregation, internal deduplication, independent freshness/failure behavior,
  and documented history/pagination decisions.
- Milestone 03 worker collector with bounded direct worker labels, worker
  state/hashrate/shares/last-share metrics, worker API freshness, and
  documented disappearance and stale-snapshot behavior.
- Milestone 02 account collector with profile polling, cached account
  hashrate/balance/worker metrics, API request counters, freshness metrics,
  and account-aware readiness.
- Milestone 01 Braiins Pool API discovery documentation, sanitized fixtures,
  minimal API client boundary, precision-preserving wire types, and redaction
  tests.
- Live Braiins Pool API structural validation checkpoint and reconciled
  optional worker/reward fields.
- Milestone 00 repository and Go service foundation.
- Health, readiness, version, and metrics endpoints.
- Exporter build information and readiness metrics.
- Secure configuration skeleton and project documentation.

[Unreleased]: https://github.com/nicosmuts/braiins-pool-exporter/commits/main
[0.0.1]: https://github.com/nicosmuts/braiins-pool-exporter/releases/tag/v0.0.1
