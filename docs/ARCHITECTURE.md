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

When no token is configured, the exporter makes no Braiins network request.
`main` wires dependencies and signals; behavior lives in internal packages.

## Account and worker polling flow

Milestone 02 adds a bounded account profile poll loop when exactly one token
source is configured. It uses the `internal/braiins` client with an explicit
HTTP timeout and context cancellation. Verified profile responses are
normalized into an immutable last-known-good snapshot owned by
`internal/collector`.

Milestone 03 adds worker polling using the same client and cache pattern. When
worker metrics are enabled, startup polling requests the profile first and then
the workers endpoint after a five-second spacing delay. Subsequent poll cycles
reuse the configured poll interval and the same conservative spacing. This is a
small milestone-specific scheduler, not a generic retry/backoff framework.

Prometheus collection never makes Braiins API requests. Scrapes expose the
cached snapshots, API request counters, last successful poll timestamps, and
data age. A transient failure leaves the last-known-good account or worker
metrics visible, with staleness increasing. Before the first successful poll
for a source, that source's data metrics are omitted. A successful worker
response is authoritative, so disappeared workers are removed immediately from
the worker snapshot.

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
seconds. Milestone 03 polls the profile and workers endpoints at the configured
interval, defaulting to one minute, with five seconds between those two
requests. Milestone 05 owns retry and backoff design.

Fixture policy: committed fixtures are synthetic or field-by-field sanitized,
with provenance documented under `testdata/braiins/`. Raw private API responses
must never be committed.

## Readiness

Without account polling, ready means local configuration is valid, the registry
is constructed, and the listener is open. With account polling enabled, ready
also requires at least one accepted account profile snapshot. Transient Braiins
Pool failures after a successful poll do not flip readiness by themselves; data
age makes stale last-known-good data observable. Liveness never depends on
Braiins Pool.

Worker polling does not add a readiness dependency. A worker first-poll failure
omits worker metrics and increments the bounded workers request result, while a
valid account snapshot can still make the exporter ready.

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
