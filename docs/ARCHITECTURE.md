# Architecture

## Goals

Braiins Pool Exporter translates verified, authoritative Braiins Pool API
fields into stable Prometheus metrics. It remains lightweight, supports
multiple workers, limits cardinality, and can run without Kubernetes.

## Foundation flow

```text
flags and environment
        |
        v
 immutable config ---> structured logger
        |
        v
 HTTP server ---> /metrics       (isolated Prometheus registry)
             ---> /-/healthy     (process liveness)
             ---> /-/ready       (initialization readiness)
             ---> /version       (sanitized build metadata)
```

The foundation makes no Braiins network request. `main` wires dependencies and
signals; behavior lives in internal packages.

## Planned polling flow

After API discovery, a bounded poll loop will use an injected `net/http` client
with timeouts and contexts. Verified wire responses will be normalized into an
immutable snapshot. Collectors will expose that snapshot while self-metrics
describe API errors, last success, and data age.

A last-known-good snapshot may remain available during transient failures, but
staleness will be explicit. Polling, cache replacement, and collection must be
race-safe. The API client will not perform work during a Prometheus scrape.

## Readiness

In Milestone 00, ready means local configuration is valid, the registry is
constructed, and the listener is open. Liveness never depends on Braiins Pool.
API discovery and hardening milestones must decide whether readiness requires
an initial successful poll and how stale cached data affects it.

## Dependency direction

`cmd` may import all internal wiring packages. `server` depends only on a
minimal readiness behavior and Prometheus registry. `collector` owns metrics,
not HTTP or API transport. `braiins` owns transport/schema details, not
Prometheus. `config` and `version` are leaf packages.

Interfaces are introduced at actual boundaries—primarily an API client and
test clock—not preemptively for every type.

## Historical data

Prometheus naturally records values after deployment. Rewards and payouts must
not be represented by attaching historical dates to current scrape samples.
Milestone 04 will choose among current summary metrics, bounded recent-event
exposure, or a separate timestamp-aware persistent ingestion path. Historical
backfill is out of scope until that decision is documented.

## Deployment boundary

The exporter binary and future container are public-project artifacts. The
Smuts Tech Helm chart, Secret, scrape configuration, and composite workshop
dashboard belong to later work in the separate Helm repository.
