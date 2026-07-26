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
or secrets.

## Adding API behavior

Do not infer the contract from memory. Use official documentation in Milestone
01, capture only sanitized fixtures, document optional fields and numeric
encoding, and finalize metric semantics before adding collectors.

## Build metadata

The variables in `internal/version` are safe linker injection points. A later
release workflow may use:

```text
-X github.com/nicosmuts/braiins-pool-exporter/internal/version.Version=...
-X github.com/nicosmuts/braiins-pool-exporter/internal/version.Commit=...
-X github.com/nicosmuts/braiins-pool-exporter/internal/version.BuildDate=...
```

Do not inject configuration or secrets into build metadata.
