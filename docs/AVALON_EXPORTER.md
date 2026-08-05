# Optional Avalon miner exporter

This repository can optionally run
[`brav0charlie/avalonhome-prometheus-exporter`](https://github.com/brav0charlie/avalonhome-prometheus-exporter)
alongside Braiins Pool Exporter for local device-side metrics. The integration
is intentionally optional because miner telemetry can reveal operational
details and is not required for the default pool-side exporter stack.

## Scope

The default stack remains:

```sh
docker compose up -d
```

It starts only:

- `braiins-pool-exporter`
- `prometheus`
- `grafana`

Enable local Avalon miner metrics with:

```sh
docker compose -f compose.yaml -f compose.miner.yaml --profile miner up -d
```

The optional profile starts `avalonhome-prometheus-exporter` and switches
Prometheus to `prometheus/prometheus.miner.yml`, which adds:

```yaml
job_name: avalonhome-prometheus-exporter
```

Prometheus scrapes the exporter by Compose service name. Miner IPs are passed
only to the Avalon exporter.

## Local defaults

The local profile defaults are:

```text
AVALON_IPS=avalon1047-01.local,avalon1047-02.local,avalon1047-03.local,avalon1047-04.local,avalon1047-05.local,avalon1047-06.local
AVALON_PORT=4028
UPDATE_INTERVAL=15
EXPORTER_PORT=9100
EXPORT_CHIP_METRICS=false
MINER_TIMEOUT=5.0
ENABLE_DEBUG_ENDPOINT=false
AVALON_LOG_LEVEL=INFO
```

These are local development defaults only. Operators must override them outside
this repository for other environments.

## Selected upstream image

The optional profile uses the upstream released image:

```text
ghcr.io/brav0charlie/avalonhome-prometheus-exporter:v0.4.0@sha256:a6e0e4a2bf9070bafb2e084e18b347ef9bfbdbe5ce86c2c6a2850363f29adea7
```

Review findings:

- Repository: `brav0charlie/avalonhome-prometheus-exporter`
- Latest release reviewed: `v0.4.0`
- License: MIT
- Runtime: Python 3.12 Alpine, non-root `app` user
- Health endpoint: `/health`
- Metrics endpoint: `/metrics`
- Version endpoint: `/version`
- Debug endpoint: `/debug`, disabled by default
- Miner commands: read-only CGMiner commands combined as
  `version+summary+stats+config+devs+devdetails+pools`
- Image platforms verified: `linux/amd64` and `linux/arm64`
- Attestations: BuildKit attestation manifests are present

## AvalonMiner 1047 compatibility decision

Outcome A is now selected: the upstream exporter includes AvalonMiner 1047
support and remains suitable for the optional local Compose integration. The
Compose profile configures six placeholder miner targets through `AVALON_IPS`.
Prometheus still scrapes one target, `avalonhome-prometheus-exporter`, and the
exporter emits one set of miner metrics per configured target with an `ip`
label.

Live read-only validation against two AvalonMiner 1047 devices confirmed:

- Product/model response identifies `AvalonMiner 1047`
- CGMiner version is `4.11.1`
- CGMiner API version is `3.7`
- Both miners were reachable from an ordinary Docker bridge container
- Upstream `v0.3.2` reported both miners up during the earlier baseline review
- Summary shares, hardware errors, fan duty, Fan1, max temperature, average
  temperature, moving hashrate, average hashrate, and work utility were
  exported
- No raw miner payload, DNA, MAC address, pool URL, account/worker name, or
  private response body was committed

Previously reviewed gaps in upstream `v0.3.2` that `v0.4.0` is expected to
address for AvalonMiner 1047:

| 1047 field | `v0.4.0` expectation | Notes |
|---|---|---|
| Model detection | Addressed upstream | `AvalonMiner 1047` is now listed as supported by upstream documentation. |
| `MHS 30s` | Addressed upstream | Current hashrate is expected through `avalon_hashrate_ghs`. |
| `MHS 1m`, `MHS 5m`, `MHS 15m` | Addressed upstream | Summary-window metrics are expected through `avalon_hashrate_1m_ghs`, `avalon_hashrate_5m_ghs`, and `avalon_hashrate_15m_ghs`. |
| `Fan2` | Addressed upstream | Second fan RPM is expected through `avalon_fan2_rpm` when firmware reports it. |
| `Temp` | Addressed upstream | Current temperature is expected through `avalon_temp_current_celsius`. |
| `SYSTEMSTATU` | Addressed upstream | Working state is expected through `avalon_system_working`. |
| Hash-board count | Addressed upstream | Board count is expected through `avalon_hash_boards`. |
| `PVT_T1`, `PVT_V1`, `MW1` | Addressed upstream | Board-indexed chip arrays are expected to be included in aggregate chip metrics. |
| Per-board `MGHS`, `MTmax`, `MTavg` | Addressed upstream | Board metrics are expected through `avalon_hashboard_*` metrics. |
| Power fields | Still firmware-defined | `MPO` and `PS` fields remain firmware-defined and should be labelled as experimental in dashboards. |
| Pool labels | Privacy-sensitive | Upstream labels per-pool metrics with pool URL; keep miner exporter endpoints private. |

## Six-miner configuration

Configure six Avalon 1047 miners with a comma-separated `AVALON_IPS` value in
an uncommitted `.env` or operator-managed configuration:

```text
AVALON_IPS=avalon1047-01.local,avalon1047-02.local,avalon1047-03.local,avalon1047-04.local,avalon1047-05.local,avalon1047-06.local
```

Use real local hostnames or IP addresses outside the public repository. Stable
hostnames are preferred because the upstream exporter exposes the configured
target through the Prometheus `ip` label, and Grafana uses that label as the
miner filter. Prometheus must continue scraping only
`avalonhome-prometheus-exporter:9100`; it should not scrape miner CGMiner ports
directly.

Offline miners are expected to leave the exporter, Prometheus, Grafana, and
Compose lifecycle healthy. When the upstream exporter can emit the down state,
the miner appears as `avalon_up{ip="..."}` with value `0`. Otherwise, panels
should show no data for metrics that are absent while the miner is offline.

## Earlier direct-versus-exported metric summary

The earlier live comparison used read-only CGMiner commands and compared
selected fields to `/metrics` from upstream `v0.3.2`. Exact live readings are
not published because they are operational telemetry. Re-validate these fields
against `v0.4.0` when all six miners are available.

| Source field | Exporter metric | Result |
|---|---|---|
| `GHSmm` | `avalon_hashrate_moving_ghs` | Matched within scrape-time variance. |
| `GHSavg` | `avalon_hashrate_avg_ghs` | Matched within scrape-time variance. |
| `MHS 30s` | `avalon_hashrate_ghs` | Not exported when `GHSspd` is absent. |
| `Accepted` | `avalon_shares_accepted_total` | Matched. |
| `Rejected` | `avalon_shares_rejected_total` | Matched. |
| `HW` | `avalon_hw_errors_total` | Matched. |
| `TMax` | `avalon_temp_max_celsius` | Matched. |
| `TAvg` | `avalon_temp_avg_celsius` | Matched within scrape-time variance. |
| `Fan1` | `avalon_fan1_rpm` | Matched within scrape-time variance. |
| `Fan2` | none | Missing. |
| `FanR` | `avalon_fan_duty_percent` | Matched. |
| `MPO` | `avalon_mpo_target` | Present, unit remains firmware-defined. |
| `PVT_T0` and `PVT_T1` | `avalon_chip_count` | Only first-board arrays are counted. |

## Security notes

- The miner exporter uses no privileged mode and no additional Linux
  capabilities in this Compose integration.
- Host networking is not used; ordinary Compose bridge networking was
  validated.
- `EXPORT_CHIP_METRICS=false` by default to avoid high-cardinality per-chip
  series.
- `ENABLE_DEBUG_ENDPOINT=false` by default because `/debug` exposes internal
  state such as miner targets and recent error messages.
- Device-side metrics can reveal operational details and should not be exposed
  publicly without network controls.
- The selected upstream exporter labels per-pool metrics with pool URLs. Keep
  access to the Avalon exporter and Prometheus restricted.

## Issue #11 handoff

The standalone Avalon dashboard lives at
`grafana/dashboards/avalon-dashboard.json` with UID
`avalon-miner-exporter` and title `Avalon Miner Exporter`. It visualizes only
`avalonhome-prometheus-exporter` metrics and is intended to validate device-side
metric usefulness before Issue #11 combines pool-side and miner-side views.

Issue #11 should later build the production operations dashboard from already
available metrics. It should combine:

- Braiins Pool account and worker hashrate
- Direct miner hashrate
- Miner availability
- Pool-side worker state
- Miner temperatures
- Fan metrics
- Shares and rejections
- Stale-data state
- Partial-fleet state

This task does not implement that dashboard.
