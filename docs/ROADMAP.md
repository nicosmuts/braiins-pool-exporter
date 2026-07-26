# Roadmap

GitHub Issues and Milestones are the tracking source of truth. This document
explains the durable stage boundaries.

## Milestone 00 — Repository Foundation

Status: implemented, validated, committed, and published. GitHub milestone and
issue tracking remains to be created after the required approval and
authenticated GitHub access.

Bootstrap the repository, Go module, documentation, HTTP lifecycle, exporter
self-metrics, secure configuration skeleton, and local validation. No Braiins
API calls or API-derived metrics.

## Milestone 01 — Braiins API Discovery

Verify official endpoints, authentication, coin selection, schemas,
pagination, time ranges, rate limits, numeric encoding, optional fields,
worker states, timestamps, balances, and payout thresholds. Produce sanitized
fixtures, redaction tests, and an initial approved metrics contract.

## Milestone 02 — Account Collector

Implement verified account profile/statistics, hashrate windows, available
balance fields, failure behavior, caching, and freshness metrics.

## Milestone 03 — Worker Collector

Implement worker discovery, state, hashrates, shares, last-share time, and
bounded multi-worker behavior.

## Milestone 04 — Rewards and Payouts

Implement verified reward and payout summaries with correct BTC precision,
pagination/date behavior, and a deliberate history model.

## Milestone 05 — Exporter Hardening

Status: implemented, validated, committed, published, and closed.

Add bounded retry/backoff, last-known-good caching, staleness behavior,
rate-limit handling, performance validation, race testing, and security review.

## Milestone 06 — Default Grafana Dashboard

Status: implemented in `grafana/braiins-pool-exporter.json` with static
dashboard validation.

Build and verify the reusable `braiins-pool-exporter` dashboard using only
metrics emitted by this exporter.

## Milestone 07 — Containers and Release Engineering

Status: implemented with Dockerfile, Compose development stack, CI validation,
and tag-gated release publishing workflow.

Add a production multi-stage Dockerfile, multi-architecture GHCR publishing,
version injection, CI/CD, release procedure, and SBOM/provenance planning.

## Milestone 08 — First Public Release

Status: in pre-release review. No release tag has been created.

Complete documentation, license, dependency, history, fixture, and security
reviews; produce `v0.1.0`; explicitly review repository visibility before any
public change.

## Milestone 09 — Production Integration

This milestone has two deliberately separate delivery phases:

1. In an operator-managed deployment repository, deploy an immutable published
   image, mount a Kubernetes Secret, configure Prometheus scraping, and validate
   the production rollout. This phase requires separate deployment approval.
2. In site-specific configuration, combine exporter metrics with device
   telemetry, price data, profitability assumptions, and optional wallet data.
   Keep external data sources and deployment-specific mappings outside the
   public exporter.

## Delivery dependencies

```text
00 Foundation
  -> 01 API Discovery
     -> 02 Account Collector
     -> 03 Worker Collector
     -> 04 Rewards and Payouts
        -> 05 Hardening
           -> 06 Default Dashboard
           -> 07 Containers and Releases
              -> 08 First Public Release
                 -> 09 Production Integration
                    -> production deployment phase
                    -> operational dashboard phase
```

Some collector work may overlap after Milestone 01, but metric and security
decisions from discovery are prerequisites.
