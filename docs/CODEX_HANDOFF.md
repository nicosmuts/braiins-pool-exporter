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
- Toolchain: Go 1.26.5
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

Milestone 02 implements the first account collector:

- token presence enables a single profile polling loop;
- account hashrate windows, current balance, and account worker counts are
  exposed from the cached profile snapshot;
- API request, last success, and data-age metrics expose bounded operational
  state;
- Prometheus scrapes never call Braiins Pool directly;
- the last-known-good snapshot remains visible during transient failures;
- readiness requires a first successful account snapshot only when account
  collection is enabled.

Milestone 03 implements the worker collector:

- worker metrics are enabled by default when account collection is enabled;
- the Braiins API worker name is emitted directly as the bounded `worker`
  label after privacy/cardinality review;
- `BRAIINS_POOL_MAX_WORKERS` bounds accepted worker snapshots, defaulting to
  100 workers;
- known worker states are `ok`, `low`, `off`, and `dis`; unknown raw states are
  exposed only as `unknown`;
- missing optional hashrate, shares, scoring hashrate, and last-share fields
  are omitted rather than converted to false zero;
- successful worker responses replace the worker snapshot, so disappeared
  workers disappear immediately;
- worker poll failures preserve last-known-good worker metrics and expose
  staleness independently from account freshness;
- worker first-poll failure does not block account readiness.

Milestone 04 implements rewards and payouts:

- rewards and payouts are enabled by default when account collection is
  enabled and can be disabled independently;
- `BRAIINS_POOL_HISTORY_DAYS` controls the inclusive UTC date window,
  defaulting to seven days and capped at 90 days;
- the collector performs one date-filtered request per endpoint because no
  pagination parameters or metadata are documented or live-observed;
- exact repeated records inside a response are deduplicated internally without
  exporting event identifiers;
- BTC reward components are aggregated with exact decimal arithmetic and
  converted to `float64` only for Prometheus exposition;
- payout amounts and fees are aggregated as integer satoshis with overflow
  checks;
- reward dates, payout timestamps, destinations, transaction IDs, invoices,
  preimages, financial account names, and event identifiers are never metric
  labels;
- rewards and payouts have independent last-known-good snapshots, request
  counters, freshness metrics, and failure behavior;
- rewards and payouts do not block readiness.

Milestone 05 hardens exporter polling and security:

- each logical API poll uses at most three HTTP attempts;
- retry eligibility is limited to transient transport failures, request
  timeouts, HTTP 429, and HTTP 5xx;
- non-rate-limit backoff is deterministic at one second then two seconds,
  capped at five seconds;
- HTTP 429 is classified as `rate_limited` and honors valid `Retry-After`
  seconds or HTTP-date values up to the five-second cap;
- cancellation interrupts in-flight requests and retry waits;
- API redirects are refused;
- public request-result categories are bounded and privacy-safe;
- stale-but-valid last-known-good data remains exported, with data older than
  five poll intervals considered operationally stale through data age;
- poll cycles are serialized and the next cycle is scheduled only after the
  previous cycle completes;
- rewards and payouts are capped at 1,000 records per bounded window;
- concurrent poll/scrape tests and synthetic benchmarks cover the cached
  collectors.
- race validation passed in Docker using a clean public clone of commit
  `f916e665b3892bde63f5d8a2a2111d64b0dd5c34`, `golang:1.26.4-bookworm`,
  linux/amd64, `CGO_ENABLED=1`, and gcc/cc 12.2.0.

Milestone 06 adds the reusable default Grafana dashboard:

- dashboard JSON lives at `grafana/braiins-pool-exporter.json`;
- UID is `braiins-pool-exporter` and title is `Braiins Pool Exporter`;
- `DS_PROMETHEUS` is a Prometheus datasource variable;
- `job` and `instance` are portable multi-select include-all variables based
  on `braiins_pool_exporter_ready`;
- `worker` is an optional multi-select include-all runtime filter based on
  `braiins_pool_worker_state`, with no worker values stored in JSON;
- panels cover exporter readiness, selected instances, API result rate,
  failed poll ratio, endpoint data age, last successful poll, account
  hashrate, rewards balance, account worker counts, worker states, worker
  status, worker hashrate, worker shares, last-share age, rewards, payout
  amounts, and payout fees;
- static Go tests validate dashboard JSON, UID/title, variables, datasource
  usage, metric references, units, portability filters, counter handling,
  absence of alert definitions, and forbidden public values.

Milestone 07 adds container and release engineering:

- `Dockerfile` builds a static linux binary with `CGO_ENABLED=0`,
  `-buildvcs=false`, version metadata injection, OCI labels, and a distroless
  non-root runtime without a shell or package manager;
- `compose.yaml` starts exactly three services: exporter, Prometheus, and
  Grafana;
- Prometheus scrapes the exporter using `prometheus/prometheus.yml`;
- Grafana provisions the Prometheus datasource and default dashboard from
  `grafana/provisioning/`;
- `.env.example` documents only supported configuration variables;
- `secrets/` supports ignored local token files while tracking only
  `.gitkeep` and `README.md`;
- `.github/workflows/ci.yml` validates pull requests and `main` pushes without
  publishing artifacts;
- `.github/workflows/release.yml` publishes multi-arch GHCR images and creates
  GitHub Releases only for `v*.*.*` tag pushes.

Pre-release repository hardening after Milestone 07 adds Dependabot for Go
modules, GitHub Actions, Dockerfile, and Docker Compose dependencies; improved
issue and pull request templates; workflow concurrency; current Node 24-based
GitHub Action majors; and `docs/RELEASE.md` as the release checklist and
procedure.

Milestone 08 prepares `v0.0.1` as the first public development release. The
release notes must describe account, worker, reward, and payout metrics;
hardened polling, caching, retries, and rate-limit handling; the default
Grafana dashboard; the production Docker image; the Docker Compose
exporter/Prometheus/Grafana stack; CI, race testing, Dependabot, SBOM, and
provenance; and the warning that interfaces and metrics may still change before
`v0.1.0`.

No Kubernetes, Helm, production deployment, or stable production release
exists. The initial public development release tag `v0.0.1` exists.

After `v0.0.1`, an optional Avalon miner exporter Compose profile was prepared
for Issue #11. It uses upstream
`ghcr.io/brav0charlie/avalonhome-prometheus-exporter:v0.4.0` as a pinned,
optional `miner` profile service named `avalonhome-prometheus-exporter`.
Default `docker compose up -d` remains limited to Braiins Pool Exporter,
Prometheus, and Grafana. Miner mode uses
`docker compose -f compose.yaml -f compose.miner.yaml --profile miner up -d`,
local placeholder defaults for six Avalon 1047 miner targets and `AVALON_PORT=4028`, and
the separate `prometheus/prometheus.miner.yml` scrape config.

Live read-only validation originally showed upstream `v0.3.2` was suitable as
a baseline local integration for AvalonMiner 1047 devices. The optional profile
now uses upstream `v0.4.0`, which includes the 1047 support work. Keep Issue #11
dashboard work separate until the standalone Braiins and Avalon dashboards have
been validated with the expanded six-miner setup.

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

Milestone 05 race validation was completed on 2026-07-26 after Docker became
available. The passing checkpoint used a clean public clone at
`f916e665b3892bde63f5d8a2a2111d64b0dd5c34` in `golang:1.26.4-bookworm` and ran
`go mod download`, `go vet ./...`, `go test -count=1 ./...`, and
`CGO_ENABLED=1 go test -race -count=1 ./...`.

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
- With account polling enabled, readiness requires one accepted account
  snapshot; later staleness is observable through data age.
- Worker freshness is independent from account freshness and worker polling
  does not block readiness.
- Rewards and payouts use rolling bounded-window summaries, not historic event
  labels or backfill samples.
- Retry and rate-limit behavior is bounded, cancellation-aware, and
  endpoint-agnostic.
- Data age is the staleness signal; stale data is not silently hidden.
- Never represent historic event dates as current-sample labels.
- Keep public dashboard logic separate from deployment-specific composite
  dashboards. The default dashboard queries only documented exporter metrics
  and remains parameterized through datasource, job, instance, and optional
  worker variables.
- Accept tokens only through environment or mounted file, never CLI flags.

## GitHub tracking

The repository has exactly ten milestones, 00 through 09, and exactly twelve
issues: one parent and eleven deliverables. Milestones 00 through 07 are
closed, issues #1 through #8 are closed, and parent issue #12 is updated
through Milestone 07. Issue #9 and Milestone 08 own the first public release
preparation and must remain open until release-specific review is complete.

## Validation commands

Run from the repository root:

```powershell
go version
go mod tidy
gofmt -w .
go vet ./...
go test -count=1 ./...
go test -count=1 ./grafana
go test -race -count=1 ./...
go build -buildvcs=false -o bin/braiins-pool-exporter.exe ./cmd/braiins-pool-exporter
docker build -t ghcr.io/nicosmuts/braiins-pool-exporter:dev .
docker compose config
docker compose up --build -d
docker compose ps
docker compose down -v
git diff --check
git status --short --branch
```

For release preparation, also validate `.github/dependabot.yml`, inspect
tracked files and recent history for secrets/private data, verify GitHub
workflow permissions and branch-protection compatibility, and confirm no release
tag exists before proceeding.

Manually start the service, request `/metrics`, `/-/healthy`, `/-/ready`, and
`/version`, then interrupt it to verify clean shutdown and sanitized logs.
Without a token, account metrics must be absent. With a safely configured
token file, account metrics appear only after a successful profile poll.
Worker metrics appear after a successful worker poll unless disabled with
`BRAIINS_POOL_WORKER_METRICS_ENABLED=false`.
Rewards and payouts metrics appear after each endpoint's first successful poll
unless disabled with `BRAIINS_POOL_REWARDS_ENABLED=false` or
`BRAIINS_POOL_PAYOUTS_ENABLED=false`. Dashboard changes should also run
`go test -count=1 ./grafana` to validate JSON, variables, metric references,
units, portability, and forbidden public values.

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
headers or cursors, rate-limit status codes, nullable profile fields beyond
the checked response, nullable worker fields beyond the checked response,
future reward/payout schema additions, and the correct daily-hashrate group
selector.

Milestone 08 is the active release boundary for `v0.0.1` and should complete
the first public development release only after all validation, security, CI,
tag, release, image, and GitHub tracking checks pass. Do not begin production
integration before the corresponding approvals and milestones.

## Expected future-session report

State the repository and branch, files changed, API evidence and official
sources, metric decisions, fixture sanitization performed, commands actually
run and results, Git status, security risks, unresolved unknowns, and the exact
next milestone. Stop at the requested milestone and identify every operation
still requiring approval.
