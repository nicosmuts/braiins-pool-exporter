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

## Account, worker, rewards, and payouts polling flow

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

Milestone 04 adds rewards and payouts polling to the same scheduled sequence.
Each enabled source is spaced by about five seconds to respect the documented
safe request rate. Rewards and payouts use independent cached snapshots and do
not affect account readiness. A malformed or failed rewards response preserves
the previous rewards snapshot without touching payouts; payouts failures behave
the same way in reverse.

Prometheus collection never makes Braiins API requests. Scrapes expose the
cached snapshots, API request counters, last successful poll timestamps, and
data age. A transient failure leaves the last-known-good account, worker,
rewards, or payouts metrics visible, with staleness increasing. Before the
first successful poll for a source, that source's data metrics are omitted. A
successful worker response is authoritative, so disappeared workers are
removed immediately from the worker snapshot. Successful empty rewards or
payouts responses replace prior non-empty summaries with zeroed bounded
aggregates.

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
decimal text at the wire boundary. Account and worker metrics convert to
`float64` at exposition. Milestone 04 reward aggregation uses exact rational
decimal arithmetic and converts only at Prometheus exposition. Payout satoshi
values are integer counts with overflow checks.

Timestamp policy: documented timestamps are Unix seconds. Date filters use
`YYYY-MM-DD`. Historical dates must not become Prometheus labels.

Schema-evolution policy: the discovery wire types model documented fields and
ignore unknown JSON fields. Optional or nullable behavior that is not yet
documented remains an explicit gap until live validation proves it.

Rate-limit policy: the documented safe rate is about one request per five
seconds. The exporter polls enabled profile, workers, rewards, and payouts
endpoints at the configured interval, defaulting to one minute, with about
five seconds between enabled requests. Milestone 05 owns retry and backoff
design.

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

Rewards and payouts do not add readiness dependencies. First-poll failures
omit only their data metrics and increment bounded request-result categories.

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
Milestone 04 chooses rolling bounded-window summaries over event labels or a
persistent backfill store. The exporter computes one inclusive UTC date window
from `BRAIINS_POOL_HISTORY_DAYS` and requests rewards and payouts with
`from=YYYY-MM-DD&to=YYYY-MM-DD`. Because no page-number, cursor, offset,
page-size parameter, or pagination metadata is documented or live-observed,
the collector performs one request per endpoint per window and deduplicates
exact repeated records within that response only. Historical backfill remains
out of scope unless a future timestamp-aware ingestion design is approved.

## Milestone 05 hardening decisions

Milestone 05 hardens the existing profile, workers, rewards, and payouts
polling path without adding new business metrics.

| Decision | Policy |
|---|---|
| Retry eligibility | Retry transient transport failures, request timeouts, HTTP 429, and HTTP 5xx. Do not retry invalid credentials, forbidden access, HTTP 4xx other than 429, decode errors, schema/validation errors, local configuration errors, or cancellation. |
| Retry count | A logical poll may make at most three attempts: one initial request and two retries. |
| Backoff formula | Non-rate-limit retries use deterministic exponential backoff: 1 second, then 2 seconds. |
| Maximum backoff | Any computed wait is capped at 5 seconds. |
| Jitter policy | No jitter in this milestone. The exporter already spaces enabled endpoints by about five seconds, and deterministic waits keep testing clear. |
| Cancellation behavior | Context cancellation interrupts in-flight requests and retry waits. A canceled wait returns `canceled` and no further attempts start. |
| Rate-limit behavior | HTTP 429 is classified as `rate_limited`. A valid `Retry-After` seconds or HTTP-date value is honored up to the 5-second cap. Missing, negative, malformed, or excessive values fall back to the capped 5-second conservative delay. |
| Timeout ownership | `BRAIINS_POOL_TIMEOUT` configures the HTTP client's per-attempt timeout, defaulting to 10 seconds. Poll lifecycle cancellation owns shutdown and retry-wait cancellation. No endpoint-specific timeout is added in this milestone. |
| Cache retention | Failed polls preserve each endpoint's last-known-good snapshot. Successful authoritative empty responses replace previous data with that endpoint's documented empty representation. |
| Staleness thresholds | Data older than five poll intervals is operationally stale. Stale-but-valid data remains exported; `braiins_pool_data_age_seconds` is the bounded operational signal. No new stale metric is added because data age already exposes the transition without expanding the metric contract. |
| Readiness impact | Readiness remains tied only to initial account profile success when account collection is enabled. Worker, rewards, and payouts staleness or failure do not block readiness. |
| Partial-endpoint failure | Profile, workers, rewards, and payouts maintain independent request counters, last-success timestamps, data age, and last-known-good caches. |
| Poll overlap prevention | Polling is serialized in one lifecycle-owned loop. The next cycle is scheduled after the previous cycle completes and then waits the configured poll interval, so a slow endpoint cannot overlap itself. |
| Concurrent scrape behavior | Prometheus scrapes use cached snapshots only. Snapshot replacement is lock-protected, and reward/payout snapshots are deep-copied before exposition. |
| Error classification | Public request results are bounded: `success`, `unauthorized`, `forbidden`, `rate_limited`, `timeout`, `transport`, `server`, `decode`, `invalid_data`, `limit_exceeded`, `canceled`, and `http_error` for unexpected non-retryable HTTP statuses. |
| Logging and redaction | Poll errors return bounded categories where private identifiers may exist. The API client never includes response bodies, authorization headers, full URLs, or raw header values in errors. |
| Resource limits | HTTP responses are capped at 1 MiB. Worker snapshots are capped by `BRAIINS_POOL_MAX_WORKERS`. Rewards and payouts accept at most 1,000 records each per bounded window. Payout/reward deduplication maps are bounded by those record caps. |
| Performance targets | Cached scrapes should be independent of API latency, allocation-conscious, and benchmarked with synthetic account, worker, rewards, and payouts snapshots. Benchmarks are engineering evidence, not release SLOs. |

## Milestone 06 dashboard decisions

The default Grafana dashboard lives under `grafana/` and is a reusable public
artifact, not production provisioning. It imports with UID
`braiins-pool-exporter` and title `Braiins Pool Exporter`.

The dashboard depends only on the documented exporter metric contract:
account, worker, reward, payout, API request, freshness, and exporter
readiness metrics. It intentionally does not query deferred metrics, standard
Go/process metrics, miner-device telemetry, Kubernetes metrics, Bitcoin price
feeds, wallet monitors, or profitability data.

Datasource, job, instance, and worker selection are Grafana variables.
`braiins_pool_exporter_ready` is used for portable job and instance discovery
because it is always present when the exporter is scraped. Worker filtering is
optional and stores no worker values in JSON; runtime worker labels may still
be private operator data.

No-data states remain visible instead of being converted to false zero. Absent
series can mean no token, disabled collectors, first poll not complete,
optional API fields absent, empty variable selections, or missing Prometheus
scrapes. Panel descriptions carry those caveats so the dashboard does not
expand the metric contract.

## Deployment boundary

The exporter binary and future container are public-project artifacts.
Operator-managed deployment configuration, Secrets, scrape configuration, and
composite production dashboards remain outside public exporter defaults and
belong to later integration work.
