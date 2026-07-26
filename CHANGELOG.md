# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project intends to follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) once releases begin.

## [Unreleased]

### Added

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
