# Grafana dashboard

Milestone 06 adds the reusable default dashboard for Braiins Pool Exporter.

- File: [`braiins-pool-exporter.json`](braiins-pool-exporter.json)
- UID: `braiins-pool-exporter`
- Title: `Braiins Pool Exporter`
- Minimum Grafana version: Grafana 10.4 or newer
- Dashboard schema version: `39`

The dashboard uses only metrics emitted by this exporter. It does not create a
datasource, provisioning ConfigMap, alert rule, container image, deployment
manifest, or production-specific dashboard.

## Import

1. In Grafana, import `grafana/braiins-pool-exporter.json`.
2. Select the Prometheus datasource for `DS_PROMETHEUS`.
3. Choose the `job`, `instance`, and optional `worker` filters after import.

Panels reference the datasource variable with `${DS_PROMETHEUS}` and do not
embed a datasource UID, job name, instance name, IP address, hostname, worker
name, or deployment URL.

## Design decisions

| Decision | Value |
|---|---|
| Dashboard UID | `braiins-pool-exporter` |
| Dashboard title | `Braiins Pool Exporter` |
| Minimum Grafana version | 10.4 or newer |
| Schema version | 39 |
| Datasource variable | `DS_PROMETHEUS`, type `datasource`, query `prometheus` |
| Job variable | `job`, multi-select include-all, `label_values(braiins_pool_exporter_ready, job)` |
| Instance variable | `instance`, multi-select include-all, filtered by selected job |
| Worker variable | `worker`, optional multi-select include-all, filtered by selected job and instance |
| Refresh interval | `1m` |
| Default time range | Last 6 hours |
| Panel grouping | Overview first, then collapsed pool statistics, account, worker, rewards, payouts, and API/freshness evidence |
| No-data behavior | Missing series remain no data unless the metric contract defines a real zero |
| Missing optional metric behavior | Optional fields and disabled collectors are documented in descriptions, not filled with false zeros |
| Worker-name privacy | Worker labels may expose operator naming conventions; no worker value is stored in JSON |
| Cardinality | Runtime worker labels are bounded by the exporter; dashboard variables do not add labels |
| Unit formatting | Gh/s, BTC, satoshis, seconds, request/s, percentage, and counts |
| Thresholds | Only conservative operational hints for readiness, data age, failure ratio, and last-share age |
| Rate versus counter | `braiins_pool_api_requests_total` is shown with `rate()`; rolling shares are gauges |
| Freshness interpretation | `braiins_pool_data_age_seconds` is the primary staleness signal |
| Portability | No hard-coded datasource, scrape target, worker, URL, IP, account, wallet, or environment value |

The dashboard uses `braiins_pool_exporter_ready` for job and instance variable
discovery because it is an always-present exporter self-metric and receives
Prometheus scrape labels naturally.

## Metric inventory

| Metric | Type | Labels | Unit | Collector | Presence | Dashboard use | No-data meaning | Privacy |
|---|---|---|---|---|---|---|---|---|
| `braiins_pool_exporter_build_info` | gauge | `version`, `commit`, `build_date`, `go_version` | none | self | always present | inventory only | exporter not scraped if absent | safe build metadata |
| `braiins_pool_exporter_ready` | gauge | none | none | self | always present | readiness, job and instance variables | exporter not scraped or selection empty | no private labels |
| `braiins_pool_hashrate_ghs` | gauge | `window` | Gh/s | pool | token-dependent after first pool stats success | pool hashrate time series | no token, no first success, missing documented window, or empty selection | pool-wide aggregate only |
| `braiins_pool_active_workers` | gauge | none | count | pool | token-dependent after first pool stats success | pool active workers stat | no token, no first success, or empty selection | pool-wide aggregate only |
| `braiins_pool_stats_update_timestamp_seconds` | gauge | none | Unix seconds | pool | token-dependent and field-dependent | source update time stat | no token, no first success, omitted source timestamp, or empty selection | pool-wide source timestamp only |
| `braiins_pool_account_hashrate_ghs` | gauge | `window` | Gh/s | account | token-dependent after first profile success | account hashrate time series | no token, no first success, missing profile window, or empty selection | no account label |
| `braiins_pool_account_balance_btc` | gauge | none | BTC | account | token-dependent and field-dependent | account balance stat | no token, no first success, omitted balance field, or empty selection | financial value, no account label |
| `braiins_pool_account_workers` | gauge | `state` | count | account | token-dependent after first profile success | aggregate worker states | no token, no first success, or empty selection | aggregate only |
| `braiins_pool_worker_state` | gauge | `worker`, `state` | one-hot | worker | token- and worker-endpoint-dependent | state counts, status table, worker variable | worker metrics disabled, no first worker success, no workers, or empty selection | worker label may be private |
| `braiins_pool_worker_hashrate_ghs` | gauge | `worker`, `window` | Gh/s | worker | token- and optional-field-dependent | worker hashrate time series | worker metrics disabled, no first success, omitted window, or empty selection | worker label may be private |
| `braiins_pool_worker_shares` | gauge | `worker`, `window` | shares | worker | token- and optional-field-dependent | worker shares time series | worker metrics disabled, no first success, omitted window, or empty selection | worker label may be private |
| `braiins_pool_worker_last_share_timestamp_seconds` | gauge | `worker` | Unix seconds | worker | optional-field-dependent | inventory only | last-share absent or no first worker success | worker label may be private |
| `braiins_pool_worker_last_share_age_seconds` | gauge | `worker` | seconds | worker | optional-field-dependent | last-share age bar gauge | last-share absent or no first worker success | worker label may be private |
| `braiins_pool_reward_daily_btc` | gauge | `component` | BTC | rewards | token- and rewards-endpoint-dependent | reward component summary | rewards disabled, no first success, or empty selection | bounded financial summary |
| `braiins_pool_payout_amount_sats` | gauge | `rail`, `status` | satoshis | payouts | token- and payouts-endpoint-dependent | payout amount summary | payouts disabled, no first success, or empty selection | bounded financial summary; no destinations |
| `braiins_pool_payout_fee_sats` | gauge | `rail`, `status` | satoshis | payouts | token- and payouts-endpoint-dependent | payout fee summary | payouts disabled, no first success, or empty selection | bounded financial summary; no destinations |
| `braiins_pool_api_requests_total` | counter | `endpoint`, `result` | logical polls | pool, account, worker, rewards, payouts | emitted after poll attempts | request rate and failed-poll ratio | endpoint has not attempted a poll or selection empty | bounded endpoint/result labels |
| `braiins_pool_api_last_success_timestamp_seconds` | gauge | `endpoint` | Unix seconds | pool, account, worker, rewards, payouts | endpoint-dependent after first success | last successful poll table | endpoint has not succeeded, disabled collector, or empty selection | bounded endpoint labels |
| `braiins_pool_data_age_seconds` | gauge | `endpoint` | seconds | pool, account, worker, rewards, payouts | endpoint-dependent after first success | endpoint freshness bar gauge | endpoint has not succeeded, disabled collector, or empty selection | bounded endpoint labels |

The standard Go runtime and process collectors are available from Prometheus
but are intentionally not used by this dashboard; the default view focuses on
the exporter contract documented in `docs/METRICS.md`.

## Rows and panels

| Section | Panels | Metrics |
|---|---|---|
| Exporter overview | Exporter readiness, selected instances, endpoint data age, API poll result rate | `braiins_pool_exporter_ready`, `braiins_pool_data_age_seconds`, `braiins_pool_api_requests_total` |
| Pool Statistics | Pool hashrate, pool active workers, pool stats freshness, pool source update time | pool metrics and pool stats freshness |
| Account | Account hashrate, account balance, account workers by state | account metrics |
| Workers | Worker states, worker status table, worker hashrate, worker last-share age, worker shares | worker metrics |
| Rewards | Rewards by component | `braiins_pool_reward_daily_btc` |
| Payouts | Payout amount, payout fees | payout metrics |
| API and freshness | Last successful poll, failed poll ratio | API request and freshness metrics |

No empty row is included for metrics that do not exist. Active users and the
website-only 30-minute pool hashrate average are not queried because they are
not exposed by the documented authenticated API. Deferred account reward and
account share metrics are not queried.

## Query and no-data policy

Every query uses portable filters such as:

```promql
{job=~"$job", instance=~"$instance"}
```

Worker panels also use:

```promql
{worker=~"$worker"}
```

The dashboard avoids `or vector(0)` because absent account, worker, reward,
payout, or freshness series can mean several different things: no token,
collector disabled, first poll not complete, optional API field absent,
successful empty history before a first scrape, or an empty variable
selection. Panel descriptions explain these states instead of converting them
to false zeros.

Freshness uses `braiins_pool_data_age_seconds`. Data older than five configured
poll intervals is operationally stale according to the exporter design.
Because the poll interval is configurable and not exported as a metric, the
dashboard's data-age and last-share thresholds are conservative visual hints,
not universal fault rules.

## Worker privacy

The exporter emits Braiins API worker names as the `worker` label. The
dashboard never stores worker names, aliases, or device mappings, but runtime
Grafana users can see worker labels in the optional worker variable and worker
panels. Operators should keep the exporter and dashboard behind appropriate
access controls if worker naming conventions are sensitive.

## Validation

Static validation is implemented in `dashboard_test.go` and runs with:

```sh
go test -count=1 ./grafana
go test -count=1 ./...
```

The tests parse the dashboard JSON and check UID, title, datasource variable,
job/instance/worker variables, metric references, approved units, portable
filters, counter handling, alert absence, and forbidden public values.

Rendered review for Milestone 06 used a temporary local Grafana container and
synthetic/no-live-data import only. It did not connect to production Grafana
or use private Braiins data.

## Optional Avalon dashboard

The standalone Avalon dashboard is stored at
[`dashboards/avalon-dashboard.json`](dashboards/avalon-dashboard.json).

- UID: `avalon-miner-exporter`
- Title: `Avalon Miner Exporter`
- Datasource variable: `DS_PROMETHEUS`
- Runtime filters: `job` and `instance`

This dashboard uses only metrics emitted by
`avalonhome-prometheus-exporter`. It is provisioned by the existing Grafana file
provider when the `grafana/` directory is mounted. It does not modify the
default Braiins Pool Exporter dashboard and is not the combined production
operations dashboard tracked by Issue #11.

Some panels intentionally show `No data` with upstream `v0.3.2`, including
Fan2, current temperature, board metrics, and individual chip telemetry when
`EXPORT_CHIP_METRICS=false`. These panels validate metric usefulness as
upstream support evolves without filling absent optional series with false
zeros.
