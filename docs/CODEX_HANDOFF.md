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

The foundation is published at the canonical public repository. The foundation
commit is on `main` and local/remote are synchronized. Check GitHub
authentication, milestones, issues, and Git status before tracking mutations.

## Completed milestone

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

No official Braiins API endpoint or response field has been assumed, and no
account, worker, reward, or payout collector exists.

## Foundation validation

On 2026-07-26, Milestone 00 was committed and pushed as
`a2b410f28b31834283e779913891d2b3584c026b` (`feat: establish braiins pool
exporter foundation`). Local `main` and `origin/main` are synchronized and the
working tree is clean. The repository is public. Milestone 01 has not started.
GitHub CLI authentication is currently invalid and must be repaired by the
user before any approved tracking mutation. The earlier race-test limitation
(CGO/gcc unavailable) and unavailable local `golangci-lint` remain recorded
validation caveats.

## Architecture decisions

- Poll outside Prometheus scrapes and expose an immutable cached snapshot.
- Keep API transport/schema in `internal/braiins` and metrics in
  `internal/collector`.
- Liveness never depends on the remote API.
- Revisit readiness after first-poll and staleness semantics are designed.
- Never represent historic event dates as current-sample labels.
- Keep public dashboard logic separate from deployment-specific composite
  dashboards.
- Accept tokens only through environment or mounted file, never CLI flags.

## Open milestones

See `docs/ROADMAP.md`. The remote and foundation push are complete. Before
tracking creation, verify authenticated GitHub access and obtain approval for
the exact manifest. The next development task is Milestone 01, Braiins API
Discovery. It has not started. Do not start collectors until the official API
contract and redaction behavior are verified.

## GitHub tracking

The approved model contains exactly ten milestones, numbered 00 through 09,
and exactly twelve future GitHub issues: one parent issue without a milestone
and eleven deliverable issues. Milestones 00–08 each have one deliverable;
Milestone 09 has two separate production integration issues. There is no
Milestone 10.

No GitHub tracking objects have been created. GitHub CLI authentication is
currently invalid, so tracking creation remains blocked until the user repairs
authentication. The approved structure is recorded in
`docs/GITHUB_TRACKING.md`; do not create an empty project board.

The next actions after this correction are:

1. review, commit, and push the documentation patch under the required approval
   gates;
2. repair GitHub CLI authentication;
3. create the approved tracking objects from the reconciled manifest.

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
