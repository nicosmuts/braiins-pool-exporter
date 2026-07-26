# Codex Handoff

## Objective

Continue building a public-quality Prometheus exporter for the official
Braiins Pool API without leaking secrets or adding deployment-specific
behavior to public defaults.

## Repository state

- Canonical repository: `https://github.com/nicosmuts/braiins-pool-exporter`
- GitHub owner: `nicosmuts`
- Visibility: public
- Branch: `main`
- Module: `github.com/nicosmuts/braiins-pool-exporter`
- Toolchain: Go 1.26.4
- License: Apache-2.0

The foundation, public sanitization, GitHub tracking structure, branch
rulesets, and local secret-file ignore rule are published on `main`.

## Completed milestones

Milestone 00 is implemented, validated, committed, and published:

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

Milestone 01 is documentation-backed API discovery:

- `docs/API_DISCOVERY.md` records official Braiins Pool API evidence,
  endpoints, authentication, rate-limit guidance, schema notes, precision
  decisions, sanitized-fixture provenance, and unknowns;
- `internal/braiins` contains the minimal request/response boundary, documented
  endpoint constants, precision-preserving decimal text, typed wire schemas,
  bounded response reading, and redacted errors;
- `testdata/braiins/` contains synthetic fixtures shaped from official
  documentation examples, not raw account responses;
- redaction and fixture-decoding tests are offline and require no token.

No account, worker, reward, payout, polling, cache, or retry collector exists.

## Validation caveats

A narrow live API validation checkpoint was performed on 2026-07-26 with the
token extracted from only the ignored `SECRETS.md` Braiins Pool section. The
token was copied to an OS-temporary token file outside the repository, used via
`BRAIINS_POOL_TOKEN_FILE`, and deleted. Raw responses were kept outside the
repository and deleted after structural comparison.

Live corrections recorded:

- missing and invalid token requests returned HTTP 403 with text/plain content;
- `Pool-Auth-Token` worked for authenticated profile access;
- worker records did not include `hash_rate_scoring` in the checked response,
  so that field is optional in the wire schema;
- daily rewards included `shares` and `share_prices`;
- no rate-limit headers were observed while requests were spaced by about five
  seconds;
- the daily-hashrate group endpoint returned 404 when probed with the profile
  username as the group path segment, so its group selector remains unresolved.

If CGO or `gcc` is unavailable, race-test limitations must be reported with the
exact command output. Run `golangci-lint run` only if it is already installed
or installation is explicitly approved.

## Architecture decisions

- Poll outside Prometheus scrapes and expose an immutable cached snapshot.
- Keep API transport/schema in `internal/braiins` and metrics in
  `internal/collector`.
- Use the documented `Pool-Auth-Token` header; `X-Pool-Auth-Token` is
  documented as an alternative but not used by default.
- Preserve BTC amounts and high-precision hashrates as decimal text at the
  wire boundary.
- Treat documented Unix timestamps as seconds; do not use timestamps as labels.
- Liveness never depends on the remote API.
- Revisit readiness after first-poll and staleness semantics are designed.
- Never represent historic event dates as current-sample labels.
- Keep public dashboard logic separate from deployment-specific composite
  dashboards.
- Accept tokens only through environment or mounted file, never CLI flags.

## GitHub tracking

The repository has exactly ten milestones, 00 through 09, and exactly twelve
issues: one parent and eleven deliverables. Milestone 00 is closed. Milestone
01 should be closed only after the Milestone 01 commit is pushed and issue #2
is updated with completion evidence. Issue #3 and Milestone 02 must remain open
until account collector work is explicitly started.

## Validation commands

Run from the repository root:

```powershell
go version
go mod tidy
gofmt -w .
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
go build -buildvcs=false -o bin/braiins-pool-exporter.exe ./cmd/braiins-pool-exporter
git diff --check
git status --short --branch
```

Manually start the service, request `/metrics`, `/-/healthy`, `/-/ready`, and
`/version`, then interrupt it to verify clean shutdown and sanitized logs.

## Security constraints

- Never print or commit a live token or unsanitized live response.
- Never put a token in a command-line argument, URL, label, error, or panic.
- Never include wallet private keys or seed phrases.
- Sanitize account identifiers, payout addresses, transaction IDs, invoices,
  preimages, worker names, and operator-sensitive values from fixtures.
- Use only official Braiins documentation for discovery.
- Do not scrape or automate the Braiins website.
- Do not modify Helm, Kubernetes, miners, firmware, or pool settings.

## Known API unknowns

Live behavior remains unverified for blank tokens, unsupported coins,
alternate auth header behavior, empty result shapes for workers, pagination
headers or cursors, rate-limit status codes, nullable profile or worker fields
beyond the checked response, and the correct daily-hashrate group selector.

## Expected future-session report

State the repository and branch, files changed, API evidence and official
sources, metric decisions, fixture sanitization performed, commands actually
run and results, Git status, security risks, unresolved unknowns, and the exact
next milestone. Stop at the requested milestone and identify every operation
still requiring approval.
