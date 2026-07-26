# Metrics

## Compatibility rules

- Braiins Pool data uses the `braiins_pool_` prefix.
- Exporter implementation state uses `braiins_pool_exporter_`.
- Use base units where Prometheus conventions require them; BTC-denominated
  values retain `_btc` because converting them to satoshis changes user-facing
  semantics and precision requirements that still need verification.
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

## Design candidates—not implemented

The following names are discussion inputs only. API discovery must verify
source fields, time windows, numeric encoding, units, missing values, and
labels before any are implemented:

- `braiins_pool_account_hashrate_ghs`
- `braiins_pool_account_balance_btc`
- `braiins_pool_worker_hashrate_ghs`
- `braiins_pool_worker_state`
- `braiins_pool_worker_last_share_timestamp_seconds`
- `braiins_pool_worker_shares_total`
- `braiins_pool_reward_btc`
- `braiins_pool_payout_btc`
- `braiins_pool_api_requests_total`
- `braiins_pool_api_errors_total`
- `braiins_pool_api_last_success_timestamp_seconds`
- `braiins_pool_data_age_seconds`

No candidate is promised until the contract is recorded here.

## Historical rewards and payouts

Prometheus should not receive a sample at scrape time with a historical reward
date encoded as a label. The initial design should prefer current summaries and
let Prometheus accumulate history after deployment. If users require backfill,
Milestone 04 must evaluate a bounded persistent store or a timestamp-aware
ingestion mechanism outside ordinary collector semantics.
