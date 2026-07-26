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

## Implemented in Milestone 00

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `braiins_pool_exporter_build_info` | gauge | `version`, `commit`, `build_date`, `go_version` | Constant build identity |
| `braiins_pool_exporter_ready` | gauge | none | Local initialization readiness |

The standard Go runtime and process collectors are also registered.

## Proposed API-derived contract from Milestone 01

These metrics are proposed from official documentation, not implemented yet.
Live validation remains blocked until `BRAIINS_POOL_TOKEN` or
`BRAIINS_POOL_TOKEN_FILE` is explicitly provided to the process. Metrics marked
approved are approved for the next implementation milestone from the documented
contract, with live validation still required before a public release.

| Metric | Type | Labels | Source | Unit/conversion | Behavior | Milestone | Status |
|---|---|---|---|---|---|---|---|
| `braiins_pool_account_hashrate_ghs` | gauge | `window` | profile: `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h`, `hash_rate_yesterday` | source `Gh/s`, export as `Gh/s` | omit missing windows; stale snapshot remains explicit through freshness metric | 02 | approved |
| `braiins_pool_account_balance_btc` | gauge | none | profile: `current_balance` | decimal BTC string | omit if absent; never label by account | 02 | approved |
| `braiins_pool_account_reward_btc` | gauge | `period` | profile: `today_reward`, `estimated_reward`, `all_time_reward` | decimal BTC string | `estimated` must be clearly named as estimated in help text | 02 | deferred pending naming review |
| `braiins_pool_account_workers` | gauge | `state` | profile: `ok_workers`, `low_workers`, `off_workers`, `dis_workers` | worker count | states map to `ok`, `low`, `off`, `dis` | 02 | approved |
| `braiins_pool_account_shares` | gauge | `window` | profile: `shares_5m`, `shares_60m`, `shares_24h`, `shares_yesterday` | rolling-window shares | not a counter because documentation describes active shares in windows | 02 | deferred pending usefulness review |
| `braiins_pool_worker_state` | gauge | `worker`, `state` | workers: `state` | one-hot state value | worker label requires explicit privacy/cardinality approval | 03 | deferred |
| `braiins_pool_worker_hashrate_ghs` | gauge | `worker`, `window` | workers: `hash_rate_scoring`, `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h` | source `Gh/s`, export as `Gh/s` | omit missing windows; stale snapshot explicit through freshness metric | 03 | deferred |
| `braiins_pool_worker_last_share_timestamp_seconds` | gauge | `worker` | workers: `last_share` | Unix seconds | do not add date labels; worker label requires approval | 03 | deferred |
| `braiins_pool_worker_shares` | gauge | `worker`, `window` | workers: `shares_5m`, `shares_60m`, `shares_24h` | rolling-window shares | not a counter because values are rolling-window counts | 03 | deferred |
| `braiins_pool_reward_daily_btc` | gauge | `component` | daily rewards: reward amount fields | decimal BTC string | expose bounded summaries only; no date labels | 04 | deferred |
| `braiins_pool_payout_amount_sats` | gauge | `rail`, `status` | payouts: `amount_sats` | satoshis | aggregate by rail/status; never label by destination, tx ID, invoice, preimage, or account name | 04 | deferred |
| `braiins_pool_payout_fee_sats` | gauge | `rail`, `status` | payouts: `fee_sats` | satoshis | aggregate by rail/status | 04 | deferred |
| `braiins_pool_api_requests_total` | counter | `endpoint`, `result` | exporter HTTP client | request count | endpoint is a bounded enum; result is a bounded success/error category | 02 | approved |
| `braiins_pool_api_last_success_timestamp_seconds` | gauge | `endpoint` | exporter polling state | Unix seconds | set only after a successful poll | 02 | approved |
| `braiins_pool_data_age_seconds` | gauge | `endpoint` | exporter polling state | seconds | age of latest accepted snapshot | 02 | approved |

Rejected label dimensions:

- username or account identifier;
- wallet or payout destination;
- transaction ID, Lightning invoice, or preimage;
- reward ID or block height as a label on ordinary scrape samples;
- arbitrary error strings, full URLs, or timestamps;
- unreviewed raw worker names.

## Historical rewards and payouts

Prometheus should not receive a sample at scrape time with a historical reward
date encoded as a label. The initial design should prefer current summaries and
let Prometheus accumulate history after deployment. If users require backfill,
Milestone 04 must evaluate a bounded persistent store or a timestamp-aware
ingestion mechanism outside ordinary collector semantics.
