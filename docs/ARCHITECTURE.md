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

## Milestone 01 API boundary

`internal/braiins` owns the official API transport and wire-schema boundary.
It constructs read-only GET requests for documented endpoints, injects the
documented `Pool-Auth-Token` header, applies a bounded response-size limit,
uses `context.Context`, and decodes JSON with precision-preserving decimal
text. It does not own polling, retry/backoff, caching, or Prometheus metrics.

Endpoint paths are versioned by explicit constants rather than string literals
inside collectors. Unknown endpoint behavior is recorded in
`docs/API_DISCOVERY.md` instead of being guessed in code.

Numeric policy: BTC amounts and high-precision hashrates are preserved as
decimal text at the wire boundary. Collector milestones may convert them at
the exposition boundary only after metric units and precision tradeoffs are
accepted. Payout satoshi values are integer counts.

Timestamp policy: documented timestamps are Unix seconds. Date filters use
`YYYY-MM-DD`. Historical dates must not become Prometheus labels.

Schema-evolution policy: the discovery wire types model documented fields and
ignore unknown JSON fields. Optional or nullable behavior that is not yet
documented remains an explicit gap until live validation proves it.

Rate-limit policy: the documented safe rate is about one request per five
seconds. Milestone 02 should poll conservatively; Milestone 05 owns retry and
backoff design.

Fixture policy: committed fixtures are synthetic or field-by-field sanitized,
with provenance documented under `testdata/braiins/`. Raw private API responses
must never be committed.

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

The exporter binary and future container are public-project artifacts.
Operator-managed deployment configuration, Secrets, scrape configuration, and
composite production dashboards remain outside public exporter defaults and
belong to later integration work.
