# Codex Handoff

## Objective

Continue building a public-quality Prometheus exporter for the official
Braiins Pool API without leaking secrets or adding workshop-specific behavior.

## Repository state

- Local path: `C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter`
- Canonical repository: `https://github.com/nicosmuts/braiins-pool-exporter`
- GitHub owner: `nicosmuts`
- Public author and project owner: Nico Smuts
- Visibility: public
- Branch: `main`
- Module: `github.com/nicosmuts/braiins-pool-exporter`
- Toolchain: Go 1.26.4
- License: Apache-2.0

The working tree was migrated intact from the former Smuts Tech workspace
location before publication. Check GitHub authentication, remote state,
milestones, issues, and Git status rather than assuming publication completed.
Repository policy requires a human review checkpoint before the initial commit
and a separate approval before push.

## Completed milestone

Milestone 00 is implemented locally:

- standalone Go module and public-project governance;
- `net/http` service with `/metrics`, `/-/healthy`, `/-/ready`, and `/version`;
- isolated Prometheus registry with Go/process collectors,
  `braiins_pool_exporter_build_info`, and
  `braiins_pool_exporter_ready`;
- `log/slog`, signal handling, and graceful shutdown;
- configuration skeleton for environment/file tokens with conflict detection,
  redaction, URL validation, and explicit timeouts;
- endpoint, metric, configuration, redaction, and lifecycle tests;
- architecture, security, metrics, roadmap, Grafana, and development docs.

No official Braiins API endpoint or response field has been assumed, and no
account, worker, reward, or payout collector exists.

## Foundation validation

On 2026-07-26:

- the working tree was moved intact from
  `C:\Users\Nico\dev\smuts-tech\braiins-pool-exporter` to the canonical local
  path, and the old path no longer exists;
- the module and all internal imports were migrated from
  `github.com/smuts-tech/braiins-pool-exporter` to
  `github.com/nicosmuts/braiins-pool-exporter`;
- `origin` was configured as
  `https://github.com/nicosmuts/braiins-pool-exporter.git`;
- the canonical GitHub repository was confirmed publicly readable and empty
  through an unauthenticated `git ls-remote`;
- `go mod tidy`, `gofmt`, `go vet ./...`, `go test -count=1 ./...`, and
  `git diff --check` passed;
- the binary built with `-buildvcs=false`, which is necessary until the
  repository has its first commit;
- a live local smoke test returned HTTP 200 for all four endpoints, exposed
  both exporter self-metrics, and did not log the placeholder token;
- the server lifecycle test exercised graceful startup and shutdown;
- `go test -race ./...` could not run because CGO is disabled, and enabling it
  failed because `gcc` is not installed;
- `golangci-lint` and GNU Make were not installed, so their targets were not
  executed.

GitHub CLI authentication still needs to be valid before GitHub issue and
milestone mutations. After the approved initial commit, rerun the exact build
command without `-buildvcs=false` and run race/lint checks in CI or an
environment with the required tools.

## Architecture decisions

- Poll outside Prometheus scrapes and expose an immutable cached snapshot.
- Keep API transport/schema in `internal/braiins` and metrics in
  `internal/collector`.
- Liveness never depends on the remote API.
- Revisit readiness after first-poll and staleness semantics are designed.
- Never represent historic event dates as current-sample labels.
- Keep public dashboard logic separate from Smuts Tech composite dashboards.
- Accept tokens only through environment or mounted file, never CLI flags.

## Open milestones

See `docs/ROADMAP.md`. First finish GitHub authentication, remote/tracking
creation, the required commit review, and the separately approved push. The
next development task is then Milestone 01, Braiins API Discovery. Do not start
collectors until the official API contract and redaction behavior are verified.

## GitHub tracking

The expected parent and milestone issue structure is recorded in
`docs/GITHUB_TRACKING.md`. Replace this note with actual issue and milestone
links after authenticated creation. Do not create an empty project board.

## Validation commands

Run from the repository root:

```powershell
go version
go mod tidy
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/braiins-pool-exporter
git diff --check
git status --short --branch
```

Run `golangci-lint run` only if it is already installed or installation has
been explicitly approved. Manually start the service, request all four
endpoints, and interrupt it to verify clean shutdown.

## Security constraints

- Never print or commit a live token or unsanitized live response.
- Never put a token in a command-line argument, URL, label, error, or panic.
- Never include wallet private keys or seed phrases.
- Sanitize account identifiers, payout addresses, transaction IDs, and
  operator-sensitive values from fixtures.
- Use only official Braiins documentation for discovery.
- Do not scrape or automate the Braiins website.
- Do not modify Helm, Kubernetes, miners, firmware, or pool settings.

## Known API unknowns

Official endpoints, authentication transport, coin path/parameter behavior,
profile and worker schemas, rewards/payout schemas, pagination, date ranges,
rate limits, numeric encoding, optional fields, worker states, timestamps,
balances, and payout thresholds all remain unverified.

## Expected future-session report

State the repository and branch, files changed, API evidence and official
sources, metric decisions, fixture sanitization performed, commands actually
run and results, Git status, security risks, unresolved unknowns, and the exact
next milestone. Stop at the requested milestone and identify every operation
still requiring approval.
