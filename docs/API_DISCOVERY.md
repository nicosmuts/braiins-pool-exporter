# Braiins Pool API Discovery

Access date: 2026-07-26.

Primary source: official Braiins Academy Monitoring documentation:
https://academy.braiins.com/braiins-pool/monitoring

Supporting payout-threshold source: official Braiins Academy Rewards &
Payouts documentation:
https://academy.braiins.com/braiins-pool/rewards-and-payouts

Live validation status: a narrow read-only structural checkpoint was performed
on 2026-07-26 using a token extracted only from the ignored local
`SECRETS.md` Braiins Pool section. The token was copied to an OS-temporary
token file outside the repository, used via `BRAIINS_POOL_TOKEN_FILE`, and
deleted after validation. No raw live responses or private values were
committed.

## Evidence summary

| Category | Status |
|---|---|
| Authentication header | Documented and live-confirmed |
| Base URL | Documented and live-confirmed |
| Coin selector | BTC documented and live-confirmed |
| Profile schema | Documented and live-confirmed structurally |
| Worker schema and states | Documented and live-confirmed structurally |
| Rewards, block rewards, payouts | Documented and live-confirmed structurally |
| Daily hashrate group endpoint | Documented, but group selector remains unresolved |
| Error response body shape | Auth failures returned non-JSON text in the live checkpoint |
| Pagination | Unknown; date filters documented for reward and payout history |
| Rate limit | Documented as about one request per five seconds; no rate-limit headers observed |

## Authentication

The official documentation says an access profile token must be sent in either
`Pool-Auth-Token` or `X-Pool-Auth-Token`. This exporter uses
`Pool-Auth-Token` for the initial client boundary. No bearer prefix is
documented.

Access tokens are generated from Settings > Access Profiles after enabling web
API access for the profile. Token rotation cancels the previous access token
for that access profile.

Live checkpoint findings:

- missing and synthetic invalid tokens returned HTTP 403;
- authentication failure responses used `text/plain; charset=utf-8`;
- valid `Pool-Auth-Token` access returned HTTP 200 for profile requests.

Unknown after live validation:

- blank-token behavior;
- whether all error paths use text/plain;
- whether errors ever echo request data in other failure classes;
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

The live checkpoint spaced requests by about five seconds and observed no
`Retry-After`, `X-RateLimit-*`, or `RateLimit-*` headers.

Milestone 05 owns retry/backoff policy. Milestone 02 should keep polling no
faster than the documented safe rate unless stronger evidence is recorded.

## Endpoint matrix

| Endpoint | Method | Path | Purpose | Parameters | Response envelope | Fixture | Confidence |
|---|---|---|---|---|---|---|---|
| Pool stats | GET | `/stats/json/btc` | Pool performance and recent blocks | coin path segment | `{ "btc": { ... } }` | none | documented |
| User profile | GET | `/accounts/profile/json/btc/` | User performance and rewards | coin path segment | `{ "username": "...", "btc": { ... } }` | `profile_success.json` | documented |
| Daily rewards | GET | `/accounts/rewards/json/btc?from=YYYY-MM-DD&to=YYYY-MM-DD` | Daily rewards for a selected period; last 90 days by default | optional `from`, `to` date strings | `{ "btc": { "daily_rewards": [...] } }` | `rewards_success.json` | documented |
| Daily hashrate | GET | `/accounts/hash_rate_daily/json/[group]/btc` | Daily average hashrate for user or user group | group path segment, coin path segment | `{ "btc": [...] }` | none | documented; group selector unresolved |
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

Live checkpoint findings:

- lowercase `btc` worked for authenticated account endpoints and pool stats.

Unknown after live validation:

- whether non-BTC coins are supported;
- whether coin selectors are case-sensitive at the API boundary;
- default coin behavior when the selector is omitted;
- unsupported coin status code and error body.

## Profile schema

The profile response includes a top-level `username` and coin-specific profile
object. `username` is sensitive and must not become a metric label by default.
Milestone 02 uses only the coin-specific profile object and does not expose the
username.

Documented BTC fields:

- reward and balance strings: `all_time_reward`, `current_balance`,
  `today_reward`, `estimated_reward`;
- hashrate unit string: `hash_rate_unit`;
- hashrate values: `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h`,
  `hash_rate_yesterday`;
- worker counts: `low_workers`, `off_workers`, `ok_workers`, `dis_workers`;
- share counts: `shares_5m`, `shares_60m`, `shares_24h`, `shares_yesterday`.

The official field table lists `hash_rate_5m` as a string, while the sample
and live checkpoint show numeric JSON values for hashrate and share windows.
The Go wire type accepts both number and string via `Decimal`.

## Worker schema

Workers are keyed by worker name inside a `workers` object. Worker names are
private operator data and must not be copied into fixtures. A future worker
label may use a sanitized worker identity only after explicit cardinality and
privacy review.

Documented worker states are `ok`, `low`, `off`, and `dis`. The monitoring
article describes them as OK, Low, Offline, and Disabled. The pool compares
worker effective hashrate snapshots every five minutes against monitoring
limits.

Live-confirmed worker fields:

- `state`;
- `last_share` Unix timestamp;
- `hash_rate_unit`;
- `hash_rate_5m`, `hash_rate_60m`, `hash_rate_24h`;
- `shares_5m`, `shares_60m`, `shares_24h`.

The official example includes `hash_rate_scoring`, but the live checkpoint did
not observe that field in worker records. The wire schema treats it as optional
until broader evidence is available.

Unknown after live validation:

- empty worker-list shape;
- ordering guarantees for object keys;
- whether deleted workers remain visible;
- duplicate or renamed worker behavior;
- whether `last_share` can be null.

## Rewards and payouts

Daily rewards use date filters with `YYYY-MM-DD` strings and return the last
90 days by default. Reward amounts are shown as strings, while the field table
describes them as numbers; the wire type therefore preserves either encoding
without converting to float. The live checkpoint also observed `shares` as a
number and `share_prices` as an array in daily reward entries.

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
verified endpoints, and no pagination metadata was observed in the live
checkpoint. Date ranges are documented for daily rewards, block rewards, and
payouts. Daily rewards return the last 90 days by default.

Milestone 02 does not call the unresolved daily-hashrate group endpoint and
does not use historical reward, block, or payout endpoints.

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
the documented official examples and the live structural checkpoint. They
preserve field names, nesting, nullable payout fields, worker states, and
numeric encoding. They do not contain raw account responses, real worker names,
payout destinations, transaction IDs, invoices, preimages, balances, earnings,
or operational timestamps.
