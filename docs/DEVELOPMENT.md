# Development

## Toolchain

Milestone 00 was initialized with Go 1.26.4 on Windows/amd64. The repository
records `go 1.26.0` and `toolchain go1.26.5` in `go.mod`, plus `1.26.5` in
`.go-version`.

GNU Make is convenient but not required. `golangci-lint` is optional locally
until CI and release engineering are defined; install it deliberately rather
than silently changing a workstation.

## Commands

```sh
go mod tidy
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go test -count=1 ./grafana
go test -run '^$' -bench . -benchmem ./...
go build ./cmd/braiins-pool-exporter
go run ./cmd/braiins-pool-exporter
```

Or:

```sh
make help
make check
```

The race detector may require platform-specific C tooling on some systems.
Report the exact failure rather than claiming it passed. On Windows hosts
without `gcc`, use an already-authorized Linux, WSL, CI, or official Go
container environment rather than installing broad tooling silently.

## Docker development stack

The Compose stack starts only the exporter, Prometheus, and Grafana:

```sh
cp .env.example .env
docker compose up --build -d
docker compose ps
```

Open:

- exporter: <http://localhost:9108>
- Prometheus: <http://localhost:9090>
- Grafana: <http://localhost:3000>

Prometheus is provisioned from `prometheus/prometheus.yml`. Grafana is
provisioned from `grafana/provisioning/` and imports the dashboard from
`grafana/braiins-pool-exporter.json`.

Token-free validation is supported. With both `BRAIINS_POOL_TOKEN` and
`BRAIINS_POOL_TOKEN_FILE` blank, the exporter starts without external Braiins
Pool API calls; Prometheus still scrapes self-metrics and Grafana still imports
the dashboard.

For local token-file validation, create `secrets/braiins_pool_token` and set
`BRAIINS_POOL_TOKEN_FILE=/run/secrets/braiins_pool_token` in `.env`. Files in
`secrets/` are ignored except for `README.md` and `.gitkeep`.

Stop and remove local volumes with:

```sh
docker compose down -v
```

For optional local Avalon miner validation:

```sh
docker compose -f compose.yaml -f compose.miner.yaml --profile miner config
docker compose -f compose.yaml -f compose.miner.yaml --profile miner up -d
docker compose -f compose.yaml -f compose.miner.yaml --profile miner ps
docker compose -f compose.yaml -f compose.miner.yaml --profile miner down -v
```

The miner profile must not be required for default exporter development. It is
only for local device-side metrics and Issue #11 preparation. The default
Braiins Pool dashboard is not modified by the miner profile.

## Container image

The production image is built from `Dockerfile`.

```sh
docker build -t ghcr.io/nicosmuts/braiins-pool-exporter:dev .
docker run --rm -p 9108:9108 ghcr.io/nicosmuts/braiins-pool-exporter:dev
```

The runtime container uses a distroless non-root image and supports read-only
filesystem operation with a writable `/tmp` tmpfs in Compose. It does not
include a shell or package manager. Do not copy `.env`, `SECRETS.md`, local
`secrets/`, or raw API responses into images.

## GitHub Actions

`.github/workflows/ci.yml` runs on pull requests and pushes to `main`. It
checks formatting, `go vet`, unit tests, race tests, `git diff --check`,
Docker build, and `docker compose config`.

`.github/workflows/release.yml` runs only for tags matching `v*.*.*`. It
validates the repository, builds linux/amd64 and linux/arm64 images, pushes to
`ghcr.io/nicosmuts/braiins-pool-exporter`, attaches OCI metadata, requests
SBOM/provenance from BuildKit, and creates a GitHub Release. The release
procedure and pre-release checklist are documented in `docs/RELEASE.md`. Do not
create release tags until release contents are reviewed.

## Manual smoke test

Start the exporter without a token:

```sh
go run ./cmd/braiins-pool-exporter --web.listen-address=:9108
```

Request `/metrics`, `/-/healthy`, `/-/ready`, and `/version`, then interrupt
the process. Confirm a clean exit and that logs contain no environment values
or secrets. Account metrics should be absent in this mode.

For local account validation, prefer a temporary token file and set
`BRAIINS_POOL_TOKEN_FILE` only in the shell running the exporter. Do not pass a
token as a command argument. With a valid token, `/metrics` should expose the
Milestone 02 account metrics after the first successful profile poll; `/-/ready`
returns not ready until that first snapshot is accepted.

Milestone 03 worker metrics are enabled by default with account collection.
Set `BRAIINS_POOL_WORKER_METRICS_ENABLED=false` to run account-only polling.
Use `BRAIINS_POOL_MAX_WORKERS` to lower the accepted worker-cardinality limit
for local tests. Worker names are emitted as runtime metric labels, so do not
copy live worker output into public files.

Milestone 04 rewards and payouts metrics are also enabled by default with
account collection. Set `BRAIINS_POOL_REWARDS_ENABLED=false` or
`BRAIINS_POOL_PAYOUTS_ENABLED=false` to disable either endpoint. Use
`BRAIINS_POOL_HISTORY_DAYS` to change the inclusive UTC date window; accepted
values are 1 through 90 days. Do not copy live reward dates, payout
destinations, transaction IDs, Lightning invoices, preimages, account names,
or financial history into public files.

Milestone 05 retry behavior is automatic and bounded. Each logical poll makes
at most three HTTP attempts. HTTP 429 honors `Retry-After` when valid and falls
back to a capped five-second delay otherwise. Scrapes never retry and never
call Braiins Pool directly. For hardening smoke tests, use synthetic local
responses for failure, retry, and rate-limit scenarios; do not intentionally
trigger live rate limits.

## Dashboard validation

The reusable default Grafana dashboard is stored at
`grafana/braiins-pool-exporter.json`. Keep it portable:

- use UID `braiins-pool-exporter` and title `Braiins Pool Exporter`;
- use the `DS_PROMETHEUS` datasource variable rather than a hard-coded
  datasource UID;
- use portable `job`, `instance`, and optional `worker` variables;
- query only metrics documented in `docs/METRICS.md` and emitted by the
  exporter;
- do not embed worker names, IP addresses, datasource names, tokens, account
  identifiers, payout destinations, transaction IDs, or live query results.

Run the static dashboard checks with:

```sh
go test -count=1 ./grafana
```

These checks parse the dashboard JSON, validate metric references, enforce
portable variables and datasource usage, reject alert definitions, verify
approved units, and scan for forbidden public values.

## Adding API behavior

Do not infer the contract from memory. Use official documentation in Milestone
01, capture only sanitized fixtures, document optional fields and numeric
encoding, and finalize metric semantics before adding collectors.

Milestone 01 records the current API discovery matrix in
`docs/API_DISCOVERY.md`. Milestones 02, 03, and 04 implement profile-backed
account metrics, worker metrics, and bounded rewards/payouts summaries. To
perform live read-only validation in a future session, export exactly one of
`BRAIINS_POOL_TOKEN` or
`BRAIINS_POOL_TOKEN_FILE` in the shell running the validation. Do not pass a
token as a command argument and do not copy raw responses into the repository.

## Build metadata

The variables in `internal/version` are safe linker injection points. A later
release workflow may use:

```text
-X github.com/nicosmuts/braiins-pool-exporter/internal/version.Version=...
-X github.com/nicosmuts/braiins-pool-exporter/internal/version.Commit=...
-X github.com/nicosmuts/braiins-pool-exporter/internal/version.BuildDate=...
```

Do not inject configuration or secrets into build metadata.
