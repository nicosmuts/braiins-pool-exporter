# Metrics

## Compatibility rules

- Braiins Pool data uses the `braiins_pool_` prefix.
- Exporter implementation state uses `braiins_pool_exporter_`.
- Use base units where Prometheus conventions require them; BTC-denominated
  values retain `_btc`.
- Counters end in `_total`; timestamps end in `_timestamp_seconds`; durations
  and ages end in `_seconds`.
- Help text, type, unit, and label sets are part of the public contract.
- Labels are bounded and stable across scrapes.

Never use tokens, wallet or payout addresses, transaction IDs, reward IDs,
arbitrary error text, full URLs, or dates/timestamps as labels. Worker labels
must be reviewed for bounded fleet behavior and privacy before release.

Worker-label decision: Milestone 03 emits the Braiins API worker name directly
as the `worker` label for per-worker metrics. Worker names are user-controlled
operator identifiers and may encode people, devices, sites, or locations, so
operators must treat the exporter HTTP surface as private operational data.
This public exporter does not contain alias mappings because those mappings are
deployment-specific. Cardinality is bounded by
`BRAIINS_POOL_MAX_WORKERS` (default `100`), blank labels are rejected, labels
longer than 128 bytes are rejected, and over-limit successful worker responses
are rejected as a whole snapshot rather than silently truncated.

Rewards/payouts history decision: Milestone 04 exposes rolling bounded-window
summaries, not historical event series. The exporter makes one date-filtered
request per enabled endpoint over `BRAIINS_POOL_HISTORY_DAYS` inclusive UTC
calendar days, defaulting to seven days and capped at 90 days. Prometheus then
records those summary gauges naturally over time. The exporter does not attach
reward dates, payout timestamps, transaction IDs, invoices, preimages,
destinations, account names, or reward/event identifiers to labels.

## Implemented in Milestone 00

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `braiins_pool_exporter_build_info` | gauge | `version`, `commit`, `build_date`, `go_version` | Constant build identity |
| `braiins_pool_exporter_ready` | gauge | none | Local initialization readiness |

The standard Go runtime and process collectors are also registered.

## Implemented in Pool Statistics telemetry

Pool-wide telemetry is enabled when a Braiins token is configured through
`BRAIINS_POOL_TOKEN` or `BRAIINS_POOL_TOKEN_FILE`. It uses the documented
authenticated Pool Stats endpoint and is cached independently from account,
worker, reward, and payout data. Before the first successful pool-stats poll,
pool data metrics are omitted.

The documented API exposes 5m, 60m, and 24h pool hashrate windows and active
pool workers. It does not expose total active users or the website-only
30-minute pool hashrate average, so those values are not implemented and must
not be estimated or scraped from the website.

| Metric | Type | Labels | Source | Unit/conversion | Behavior |
|---|---|---|---|---|---|
| `braiins_pool_hashrate_ghs` | gauge | `window` | pool stats: `pool_5m_hash_rate`, `pool_60m_hash_rate`, `pool_24h_hash_rate` | source `Gh/s`, export as `Gh/s` | omit missing windows; windows are `5m`, `60m`, and `24h` only |
| `braiins_pool_active_workers` | gauge | none | pool stats: `pool_active_workers` | worker count | emitted only after a valid pool stats snapshot; never label by user or account |
| `braiins_pool_stats_update_timestamp_seconds` | gauge | none | pool stats: `update_ts` | Unix seconds | omitted when the source timestamp is absent or zero |
| `braiins_pool_api_requests_total` | counter | `endpoint`, `result` | exporter HTTP client | logical polls | endpoint includes `pool_stats`; result is one of `success`, `unauthorized`, `forbidden`, `rate_limited`, `timeout`, `transport`, `server`, `decode`, `invalid_data`, `canceled`, `http_error`, or `error` |
| `braiins_pool_api_last_success_timestamp_seconds` | gauge | `endpoint` | exporter polling state | Unix seconds | emitted for endpoint `pool_stats` only after a successful pool stats poll |
| `braiins_pool_data_age_seconds` | gauge | `endpoint` | exporter polling state | seconds | age of latest accepted pool stats snapshot |

## Implemented in Milestone 02

Account collection is enabled only when a Braiins token is configured through
`BRAIINS_POOL_TOKEN` or `BRAIINS_POOL_TOKEN_FILE`. Polling runs outside
Prometheus scrapes and scrapes read the latest accepted profile snapshot.
After a transient failure, the last-known-good account snapshot remains exposed
and staleness is visible through `braiins_pool_data_age_seconds`. Before the
first successful account poll, account data metrics are omitted.

| Metric | Type | Labels | Source | Unit/conversion | Behavior |
|---|---|---|---|---|---|
| `braiins_pool_account_hashrate_ghs` | gauge | `window` | profile: `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h`, `hash_rate_yesterday` | source `Gh/s`, export as `Gh/s` | omit missing windows; stale snapshot remains explicit through freshness metric |
| `braiins_pool_account_balance_btc` | gauge | none | profile: `current_balance` | decimal BTC string parsed at exposition boundary | omit if absent; never label by account |
| `braiins_pool_account_workers` | gauge | `state` | profile: `ok_workers`, `low_workers`, `off_workers`, `dis_workers` | worker count | states map to `ok`, `low`, `off`, `dis` |
| `braiins_pool_api_requests_total` | counter | `endpoint`, `result` | exporter HTTP client | logical poll count | endpoint is `profile`; result is one of `success`, `unauthorized`, `forbidden`, `rate_limited`, `timeout`, `transport`, `server`, `decode`, `invalid_data`, `canceled`, `http_error`, or `error` |
| `braiins_pool_api_last_success_timestamp_seconds` | gauge | `endpoint` | exporter polling state | Unix seconds | emitted only after a successful profile poll |
| `braiins_pool_data_age_seconds` | gauge | `endpoint` | exporter polling state | seconds | age of latest accepted profile snapshot |

## Implemented in Milestone 03

Worker collection is enabled by default when account collection is enabled and
can be disabled with `BRAIINS_POOL_WORKER_METRICS_ENABLED=false`. Worker
polling runs outside Prometheus scrapes. Worker freshness is independent from
account freshness. Worker poll failures preserve the last-known-good worker
snapshot and increment bounded request-result categories. Before the first
successful worker poll, worker data metrics are omitted.

Known worker states are normalized to `ok`, `low`, `off`, and `dis`; blank or
future unseen states are exposed only as the bounded state value `unknown`.
Successful worker responses are authoritative: a worker absent from a later
successful response disappears immediately from per-worker metrics. Failed
polls preserve the previous snapshot.

| Metric | Type | Labels | Source | Unit/conversion | Behavior |
|---|---|---|---|---|---|
| `braiins_pool_worker_state` | gauge | `worker`, `state` | workers: `state` | one-hot state value | emits fixed states `ok`, `low`, `off`, `dis`, `unknown`; unknown raw states are not labels |
| `braiins_pool_worker_hashrate_ghs` | gauge | `worker`, `window` | workers: `hash_rate_scoring`, `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h` | source `Gh/s`, export as `Gh/s` | omit missing windows; `hash_rate_scoring` is optional |
| `braiins_pool_worker_shares` | gauge | `worker`, `window` | workers: `shares_5m`, `shares_60m`, `shares_24h` | rolling-window shares | omit missing windows; not a counter |
| `braiins_pool_worker_last_share_timestamp_seconds` | gauge | `worker` | workers: `last_share` | Unix seconds | omit if absent or null; no timestamp labels |
| `braiins_pool_worker_last_share_age_seconds` | gauge | `worker` | workers: `last_share` | seconds | omitted when last-share timestamp is absent |
| `braiins_pool_api_requests_total` | counter | `endpoint`, `result` | exporter HTTP client | logical poll count | endpoint includes `workers`; result is one of `success`, `unauthorized`, `forbidden`, `rate_limited`, `timeout`, `transport`, `server`, `decode`, `invalid_data`, `limit_exceeded`, `canceled`, `http_error`, or `error` |
| `braiins_pool_api_last_success_timestamp_seconds` | gauge | `endpoint` | exporter polling state | Unix seconds | emitted for endpoint `workers` only after a successful worker poll |
| `braiins_pool_data_age_seconds` | gauge | `endpoint` | exporter polling state | seconds | age of latest accepted worker snapshot |

## Implemented in Milestone 04

Rewards and payouts are enabled by default when account collection is enabled
and can be disabled independently with `BRAIINS_POOL_REWARDS_ENABLED=false` or
`BRAIINS_POOL_PAYOUTS_ENABLED=false`. Both endpoints use the same bounded
history window configured by `BRAIINS_POOL_HISTORY_DAYS`.

Pagination policy: no page-number, cursor, offset, page-size parameter, or
pagination metadata is documented or live-observed for the rewards and payouts
endpoints. Milestone 04 therefore does not invent pagination. It performs one
bounded date-window request per endpoint and deduplicates exact repeated
records inside that response.

Precision policy: BTC rewards are decoded as decimal text and aggregated with
exact rational arithmetic inside the collector. Conversion to `float64` occurs
only at the Prometheus exposition boundary. Payout amounts and fees are summed
as integer satoshis with overflow checks.

Freshness/failure policy: rewards and payouts maintain independent
last-known-good snapshots, request counters, last-success timestamps, and data
age. A rewards failure does not erase payouts, and a payouts failure does not
erase rewards. First failures omit only that endpoint's data metrics. Later
failures preserve the prior snapshot while data age increases. Neither
endpoint blocks readiness.

| Metric | Type | Labels | Source | Unit/conversion | Behavior |
|---|---|---|---|---|---|
| `braiins_pool_reward_daily_btc` | gauge | `component` | daily rewards: `total_reward`, `mining_reward`, `bos_plus_reward`, `referral_bonus`, `referral_reward` | decimal BTC aggregated exactly, exposed as BTC | bounded-window aggregate by `total`, `mining`, `bos_plus`, `referral_bonus`, and `referral_reward`; no date labels |
| `braiins_pool_payout_amount_sats` | gauge | `rail`, `status` | payouts: `amount_sats` | integer satoshis | bounded-window aggregate by rail `onchain`/`lightning` and status `queued`/`confirmed`/`failed`/`unknown`; no destination or event labels |
| `braiins_pool_payout_fee_sats` | gauge | `rail`, `status` | payouts: `fee_sats` | integer satoshis | same bounded labels and failure behavior as payout amount |
| `braiins_pool_api_requests_total` | counter | `endpoint`, `result` | exporter HTTP client | logical poll count | endpoint includes `rewards` and `payouts`; result is one of `success`, `unauthorized`, `forbidden`, `rate_limited`, `timeout`, `transport`, `server`, `decode`, `invalid_data`, `canceled`, `http_error`, or `error` |
| `braiins_pool_api_last_success_timestamp_seconds` | gauge | `endpoint` | exporter polling state | Unix seconds | emitted per endpoint only after that endpoint's successful poll |
| `braiins_pool_data_age_seconds` | gauge | `endpoint` | exporter polling state | seconds | age of latest accepted rewards or payouts snapshot |

## Deferred API-derived contract

These metrics are proposed from official documentation and a narrow read-only
live structural checkpoint, but are not implemented yet unless noted above.

| Metric | Type | Labels | Source | Unit/conversion | Behavior | Milestone | Status |
|---|---|---|---|---|---|---|---|
| `braiins_pool_account_reward_btc` | gauge | `period` | profile: `today_reward`, `estimated_reward`, `all_time_reward` | decimal BTC string | `estimated` must be clearly named as estimated in help text | 02 | deferred pending naming review |
| `braiins_pool_account_shares` | gauge | `window` | profile: `shares_5m`, `shares_60m`, `shares_24h`, `shares_yesterday` | rolling-window shares | not a counter because documentation describes active shares in windows | 02 | deferred pending usefulness review |

Rejected label dimensions:

- username or account identifier;
- wallet or payout destination;
- transaction ID, Lightning invoice, or preimage;
- reward ID or block height as a label on ordinary scrape samples;
- arbitrary error strings, full URLs, or timestamps;
- unreviewed raw worker names.

## Historical rewards and payouts

Prometheus does not receive scrape-time samples with historical reward dates
or payout timestamps encoded as labels. Milestone 04 chooses rolling
bounded-window summary gauges and lets Prometheus accumulate history after
deployment. If users require backfill later, that must be a separate
timestamp-aware ingestion design outside ordinary collector semantics.

## Hardening and staleness behavior

Milestone 05 retries happen inside a logical poll before the poll result is
recorded. `braiins_pool_api_requests_total` therefore counts logical polls by
final bounded result, not individual HTTP attempts. Operators can infer stale
state from `braiins_pool_data_age_seconds`; data older than five configured
poll intervals is considered operationally stale. Stale-but-valid
last-known-good data remains exported until a later successful poll replaces
it.
