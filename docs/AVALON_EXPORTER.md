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
AVALON_IPS=10.0.0.101,10.0.0.102
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
ghcr.io/brav0charlie/avalonhome-prometheus-exporter:v0.3.2@sha256:2768adbc0132c2b97e0ea78a00194b0bcf60f8e79e303e1268ecd3c66f75d0d7
```

Review findings:

- Repository: `brav0charlie/avalonhome-prometheus-exporter`
- Latest release reviewed: `v0.3.2`
- Release date: 2026-05-06
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

No upstream issue or pull request for Avalon 10-series support was present at
the time of review.

## AvalonMiner 1047 compatibility decision

Outcome B was selected: the upstream exporter is suitable as a baseline local
Compose integration, but a small upstream enhancement is required for complete
AvalonMiner 1047 support.

Live read-only validation against two AvalonMiner 1047 devices confirmed:

- Product/model response identifies `AvalonMiner 1047`
- CGMiner version is `4.11.1`
- CGMiner API version is `3.7`
- Both miners were reachable from an ordinary Docker bridge container
- Upstream `v0.3.2` reported both miners up
- Summary shares, hardware errors, fan duty, Fan1, max temperature, average
  temperature, moving hashrate, average hashrate, and work utility were
  exported
- No raw miner payload, DNA, MAC address, pool URL, account/worker name, or
  private response body was committed

Known gaps in upstream `v0.3.2` for AvalonMiner 1047:

| 1047 field | Upstream `v0.3.2` result | Notes |
|---|---|---|
| Model detection | Partial | `avalon_info` carries `model="1047"` and `prod="AvalonMiner 1047"`, but the project docs do not explicitly list 1047 support. |
| `MHS 30s` | Missing as current hashrate | `avalon_hashrate_ghs` is absent when `GHSspd` is absent. |
| `MHS 1m`, `MHS 5m`, `MHS 15m` | Missing | These summary windows are not exported. |
| `Fan2` | Missing | 1047 reports a second fan but upstream exports only `Fan1`. |
| `Temp` | Missing | Upstream exports `TMax` and `TAvg`, but not the 1047 current `Temp` field. |
| `SYSTEMSTATU` | Missing | The work-state string includes hash-board information and is not mapped. |
| Hash-board count | Missing | The 1047 devices reported two hash boards via `SYSTEMSTATU`. |
| `PVT_T1`, `PVT_V1`, `MW1` | Missing | Upstream aggregate and optional chip metrics use only board-0 arrays. |
| Per-board `MGHS`, `MTmax`, `MTavg` | Missing | Board-level telemetry is not exported. |
| Power fields | Partially uncertain | `MPO` is exported as firmware-defined target power; `PS` slot semantics were not treated as verified 1047 watts. |
| Pool labels | Privacy-sensitive | Upstream labels per-pool metrics with pool URL; keep miner exporter endpoints private. |

## Direct-versus-exported metric summary

The live comparison used read-only CGMiner commands and compared selected
fields to `/metrics` from the upstream exporter. Exact live readings are not
published because they are operational telemetry.

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
