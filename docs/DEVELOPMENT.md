# Development

## Toolchain

Milestone 00 was initialized with Go 1.26.4 on Windows/amd64. The repository
records `go 1.26.0` and `toolchain go1.26.4` in `go.mod`, plus `1.26.4` in
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
go build ./cmd/braiins-pool-exporter
go run ./cmd/braiins-pool-exporter
```

Or:

```sh
make help
make check
```

The race detector may require platform-specific C tooling on some systems.
Report the exact failure rather than claiming it passed.

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
