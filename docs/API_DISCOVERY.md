# Braiins Pool API Discovery

Access date: 2026-07-26.

Primary source: official Braiins Academy Monitoring documentation:
https://academy.braiins.com/braiins-pool/monitoring

Supporting payout-threshold source: official Braiins Academy Rewards &
Payouts documentation:
https://academy.braiins.com/braiins-pool/rewards-and-payouts

Live validation status: blocked. This process did not have
`BRAIINS_POOL_TOKEN` or `BRAIINS_POOL_TOKEN_FILE` set. `SECRETS.md` exists and
is ignored, but it was not used as an implicit credential source. No live
Braiins API request was made.

## Evidence summary

| Category | Status |
|---|---|
| Authentication header | Documented, not live-confirmed |
| Base URL | Documented, not live-confirmed |
| Coin selector | BTC documented, not live-confirmed |
| Profile schema | Documented, not live-confirmed |
| Worker schema and states | Documented, not live-confirmed |
| Rewards, daily hashrate, block rewards, payouts | Documented, not live-confirmed |
| Error response body shape | Unknown; fixtures are synthetic |
| Pagination | Unknown; date filters documented for reward and payout history |
| Rate limit | Documented as about one request per five seconds |

## Authentication

The official documentation says an access profile token must be sent in either
`Pool-Auth-Token` or `X-Pool-Auth-Token`. This exporter uses
`Pool-Auth-Token` for the initial client boundary. No bearer prefix is
documented.

Access tokens are generated from Settings > Access Profiles after enabling web
API access for the profile. Token rotation cancels the previous access token
for that access profile.

Unknown until live validation:

- exact status code and content type for missing, blank, malformed, and
  incorrect tokens;
- whether errors ever echo request data;
- token expiry semantics beyond regeneration;
- token scope granularity beyond access-profile web API access.

## Rate limiting and retry signals

The official documentation gives a safe request rate of approximately one
request per five seconds and warns that transient excess requests may be
ignored, while large or prolonged excess may result in an IP ban.

Unknown until live validation:

- whether HTTP 429 is returned;
- whether `Retry-After` or rate-limit headers are emitted;
- whether ignored requests use empty responses, errors, or stale data.

Milestone 05 owns retry/backoff policy. Milestone 02 should keep polling no
faster than the documented safe rate unless stronger evidence is recorded.

## Endpoint matrix

| Endpoint | Method | Path | Purpose | Parameters | Response envelope | Fixture | Confidence |
|---|---|---|---|---|---|---|---|
| Pool stats | GET | `/stats/json/btc` | Pool performance and recent blocks | coin path segment | `{ "btc": { ... } }` | none | documented |
| User profile | GET | `/accounts/profile/json/btc/` | User performance and rewards | coin path segment | `{ "username": "...", "btc": { ... } }` | `profile_success.json` | documented |
| Daily rewards | GET | `/accounts/rewards/json/btc?from=YYYY-MM-DD&to=YYYY-MM-DD` | Daily rewards for a selected period; last 90 days by default | optional `from`, `to` date strings | `{ "btc": { "daily_rewards": [...] } }` | `rewards_success.json` | documented |
| Daily hashrate | GET | `/accounts/hash_rate_daily/json/[group]/btc` | Daily average hashrate for user or user group | group path segment, coin path segment | `{ "btc": [...] }` | none | documented |
| Block rewards | GET | `/accounts/block_rewards/json/btc?from=YYYY-MM-DD&to=YYYY-MM-DD` | Block-level reward history | optional `from`, `to` date strings | `{ "btc": { "block_rewards": [...] } }` | none | documented |
| Workers | GET | `/accounts/workers/json/btc` | Per-worker performance | coin path segment | `{ "btc": { "workers": { "<worker>": { ... } } } }` | `workers_success.json` | documented |
| Payouts | GET | `/accounts/payouts/json/btc?from=YYYY-MM-DD&to=YYYY-MM-DD` | On-chain and Lightning payout records | optional `from`, `to` date strings | `{ "onchain": [...], "lightning": [...] }` | `payouts_success.json` | documented |

The documentation overview mentions four endpoints: stats, profile, workers,
and payouts. The same page also documents daily rewards, daily hashrate, and
block rewards. This project treats those additional documented sections as API
endpoints, but keeps implementation sequencing tied to milestones.

## Coin behavior

Only BTC is explicitly documented on the official monitoring page. URLs use a
lowercase `btc` path segment, while parameter notes show `BTC`. The client
normalizes configured coin values to lowercase for URL construction.

Unknown until live validation:

- whether non-BTC coins are supported;
- whether coin selectors are case-sensitive at the API boundary;
- default coin behavior when the selector is omitted;
- unsupported coin status code and error body.

## Profile schema

The profile response includes a top-level `username` and coin-specific profile
object. `username` is sensitive and must not become a metric label by default.

Documented BTC fields:

- reward and balance strings: `all_time_reward`, `current_balance`,
  `today_reward`, `estimated_reward`;
- hashrate unit string: `hash_rate_unit`;
- hashrate values: `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h`,
  `hash_rate_yesterday`;
- worker counts: `low_workers`, `off_workers`, `ok_workers`, `dis_workers`;
- share counts: `shares_5m`, `shares_60m`, `shares_24h`, `shares_yesterday`.

The official field table lists `hash_rate_5m` as a string, while the sample
shows a number. The Go wire type accepts both number and string via `Decimal`
until live behavior is verified.

## Worker schema

Workers are keyed by worker name inside a `workers` object. Worker names are
private operator data and must not be copied into fixtures. A future worker
label may use a sanitized worker identity only after explicit cardinality and
privacy review.

Documented worker states are `ok`, `low`, `off`, and `dis`. The monitoring
article describes them as OK, Low, Offline, and Disabled. The pool compares
worker effective hashrate snapshots every five minutes against monitoring
limits.

Documented worker fields:

- `state`;
- `last_share` Unix timestamp;
- `hash_rate_unit`;
- `hash_rate_scoring`, `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h`;
- `shares_5m`, `shares_60m`, `shares_24h`.

Unknown until live validation:

- empty worker-list shape;
- ordering guarantees for object keys;
- whether deleted workers remain visible;
- duplicate or renamed worker behavior;
- whether `last_share` can be null.

## Rewards and payouts

Daily rewards use date filters with `YYYY-MM-DD` strings and return the last
90 days by default. Reward amounts are shown as strings, while the field table
describes them as numbers; the wire type therefore preserves either encoding
without converting to float.

Payouts are separated into `onchain` and `lightning` arrays. Sensitive payout
fields include `destination`, `tx_id`, `invoice`, and `preimage`; these must
never be labels or public fixture values. Payout status values documented in
the sample and table are `queued`, `confirmed`, and `failed`. Trigger values
documented are `triggered` and `manual`.

The Rewards & Payouts page documents payout processing and limits. On-chain
automatic/manual minimum payout is 0.0002 BTC, minimum free payout is 0.005
BTC, and maximum payout is 5 BTC. Lightning minimum is 1 satoshi and maximum
is 0.005 BTC. Payout thresholds can be customized for financial accounts.

## Pagination, time ranges, and history

No page-number, cursor, offset, or page-size parameter is documented for the
verified endpoints. Date ranges are documented for daily rewards, block
rewards, and payouts. Daily rewards return the last 90 days by default.

Historical rewards and payouts remain unsafe as unbounded Prometheus event
labels. Milestone 04 must prefer current bounded summaries and natural
Prometheus history unless a separate timestamp-aware ingestion design is
approved.

## Numeric and timestamp policy

BTC amounts and high-precision hashrates are decoded as `Decimal`, which
retains the original JSON token text. Collector milestones may convert values
only after unit and precision decisions are finalized. Satoshi fields in
payouts are integer counts.

Documented timestamps are Unix seconds. Documentation mentions UTC for block
reward timestamps and uses Unix seconds elsewhere. Date filter parameters use
ISO `YYYY-MM-DD` strings.

## Sanitized fixtures

Fixtures under `testdata/braiins/` are synthetic, sanitized samples shaped from
the documented official examples. They preserve field names, nesting, nullable
payout fields, worker states, and numeric encoding. They do not contain raw
account responses, real worker names, payout destinations, transaction IDs,
invoices, preimages, balances, or earnings.
