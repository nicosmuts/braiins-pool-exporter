# Self-contained continuation prompts

These prompts deliberately stop at one milestone. Before any work, read the
workspace and repository `AGENTS.md` files and obey their approval gates.

## Prompt 2 — Braiins API Discovery

```text
Work in:
C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter

Complete only Milestone 01 — Braiins API Discovery for Braiins Pool Exporter.
The repository should already contain the validated Milestone 00 Go service.
First read the parent workspace AGENTS.md, this repository's AGENTS.md,
docs/CODEX_HANDOFF.md, docs/ARCHITECTURE.md, docs/METRICS.md, and
docs/SECURITY.md. Inspect git status and do not overwrite unrelated changes.

Use current official Braiins documentation as the authoritative source. Browse
only official Braiins domains for API-contract facts, cite exact pages, and
record access dates. Verify endpoints, authentication transport, coin
path/parameter behavior, profile/account fields, workers, rewards, payouts,
pagination, date ranges, rate limits, numeric encoding, optional/missing
fields, worker state values, timestamp formats, balances, and payout
thresholds. Do not infer undocumented behavior from memory.

If live validation is necessary, accept a token only through
BRAIINS_POOL_TOKEN or BRAIINS_POOL_TOKEN_FILE. Never ask for the token in chat,
print it, put it in a command line, attach it to a logged URL, or commit it.
Create only synthetic or rigorously sanitized fixtures. Remove account IDs,
worker identifiers that are private, payout/wallet addresses, transaction IDs,
and personal or operator-sensitive data. Document every sanitization.

Implement verified API wire types and request construction only as needed to
lock down the schema; do not build the complete account, worker, reward, or
payout collectors. Add token/URL/error redaction tests. Finalize the initial
metrics contract in docs/METRICS.md, clearly separating accepted metrics from
deferred candidates. Record an architecture decision for polling, pagination,
precision, and historical data.

Prohibited actions: scraping or browser automation; miner configuration;
wallet keys; Helm/Kubernetes changes; live deployment; release/tag; public
visibility change; workshop-specific defaults; speculative metrics; commit,
push, or GitHub mutations without the approvals required by AGENTS.md.

Acceptance criteria:
- official API evidence and access dates are documented;
- the listed contract unknowns are resolved or explicitly remain unknown;
- fixtures contain no secrets or private identifiers;
- token, URL, transport, response, decode, and formatting redaction tests pass;
- schemas preserve numeric precision and optionality;
- docs/METRICS.md defines only verified initial metric families;
- no full collector or live scraping loop is implemented;
- gofmt, go vet ./..., go test ./..., go test -race ./...,
  go build ./cmd/braiins-pool-exporter, and git diff --check are run;
- CODEX_HANDOFF.md is updated.

Final report: official sources, verified contract, unresolved questions,
files changed, fixture sanitization, metric decisions, exact validation
results, git status, security review, operations still awaiting approval, and
the recommended Milestone 02 task. Stop after Milestone 01.
```

## Prompt 3 — Account Collector

```text
Work in:
C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter

Complete only Milestone 02 — Account Collector. Prerequisite: Milestone 01 has
documented the official Braiins API contract, committed sanitized fixtures,
and finalized account metric names. Read all applicable AGENTS.md files,
docs/CODEX_HANDOFF.md, docs/ARCHITECTURE.md, docs/METRICS.md, and the API
discovery record. Inspect git status before editing.

Implement the verified account/profile/statistics client behavior and
Prometheus collector: documented account hashrate windows, balance fields only
where officially available, API request/error/success/freshness signals, and
an immutable last-known-good account snapshot. Poll outside scrape requests
using context.Context, an injected HTTP client with explicit timeout, and a
race-safe bounded state holder. Preserve the API's numeric precision and
define behavior for missing fields, initial failure, transient failure, and
stale data. Revisit and document readiness semantics.

Use only verified fixtures; add table-driven tests for success, optional
fields, malformed data, non-2xx responses, cancellation, timeouts, cache
replacement, staleness, Prometheus descriptors, and token/error redaction. No
test may require external network access or a real token.

Prohibited actions: worker, reward, or payout collectors; historical backfill;
speculative profitability; workshop logic; Helm/Kubernetes changes; Docker or
release work; logging tokens or response bodies; unbounded labels; commit,
push, or GitHub mutations without required approvals.

Acceptance criteria:
- account metrics exactly match the verified docs contract;
- polling never occurs during collection;
- cache and readiness behavior are deterministic and race-safe;
- failures do not expose secrets and staleness is observable;
- account balance is omitted when the official API cannot provide it;
- documentation and changelog are updated;
- gofmt, vet, unit tests, race tests, build, linter if available, and
  git diff --check pass;
- CODEX_HANDOFF.md identifies Milestone 03 as next.

Final report: repository/branch, files changed, metrics added with labels and
units, cache/readiness semantics, security analysis, validation commands and
results, git status, approvals still required, and remaining API assumptions.
Stop after Milestone 02.
```

## Prompt 4 — Worker Collector

```text
Work in:
C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter

Complete only Milestone 03 — Worker Collector. Prerequisites: verified worker
schemas and metric contract from Milestone 01, plus the account polling/cache
foundation from Milestone 02. Read applicable AGENTS.md files and all handoff,
architecture, metrics, security, and API discovery docs. Inspect git status.

Implement worker discovery and verified metrics for worker state, supported
hashrate windows, shares, and last-share timestamps. Support zero, one, or
many workers without hard-coded names or Smuts Tech mappings. Deliberately
define the worker label, privacy implications, disappearance behavior, state
mapping, unknown states, duplicate records, missing timestamps, and stale
snapshot behavior. Bound all cache and label dimensions.

Reuse the established poll/cache path rather than making requests during
Prometheus scrapes. Add sanitized multi-worker fixtures and table-driven tests
for ordering, state mapping, optional fields, duplicate/missing workers,
cardinality controls, stale data, cancellation, errors, metric descriptors,
and races.

Prohibited actions: rewards or payouts; miner/device queries or configuration;
workshop IPs and aliases; arbitrary error labels; historical dates as labels;
Grafana dashboard queries not backed by existing metrics; Helm/Kubernetes,
Docker, release, visibility, or unapproved Git operations.

Acceptance criteria:
- all worker metrics come from verified official fields;
- descriptor output is deterministic across worker order;
- labels are bounded and contain no tokens, addresses, transaction IDs, or
  arbitrary values beyond the reviewed worker identity/state contract;
- multi-worker behavior and disappearance semantics are documented;
- tests require no live API or token and pass with the race detector;
- docs, changelog, and CODEX_HANDOFF.md are updated;
- gofmt, vet, tests, race tests, build, linter if available, and diff check
  are run and reported.

Final report: metrics, label/cardinality decisions, fixtures, edge-case
behavior, files changed, exact validation, security review, git status,
approval gates, and recommended Milestone 04 work. Stop after Milestone 03.
```

## Prompt 5 — Rewards and Payouts

```text
Work in:
C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter

Complete only Milestone 04 — Rewards and Payouts. Prerequisites: official
reward/payout schemas, precision, pagination, and date behavior are verified
and documented; account and worker collectors are stable. Read all applicable
AGENTS.md, handoff, architecture, metrics, security, and discovery documents.
Inspect git status before editing.

First write an architecture decision for historical data. Do not attach
historic dates, reward IDs, payout IDs, wallet addresses, or transaction IDs
as labels on samples emitted at the current scrape time. Prefer current
bounded summaries and natural Prometheus history. If backfill is genuinely
required, design a separate timestamp-aware persistent path, but do not
implement that path in this milestone without a reviewed requirement.

Implement only verified reward and payout summary metrics. Preserve BTC
precision, handle pagination and date boundaries deterministically, define
deduplication and partial-page failure behavior, and keep memory bounded.
Create sanitized/synthetic fixtures and tests for decimal precision, empty
history, multiple pages, repeated items, boundary dates, optional fields,
errors, cancellation, stale cache, metric output, redaction, and races.

Prohibited actions: wallet monitoring; profitability estimates; price feeds;
unbounded event labels; workshop-specific history stores; Grafana dashboard;
Helm/Kubernetes; Docker/release work; unapproved commits, pushes, or visibility
changes.

Acceptance criteria:
- the historical-data decision is explicit and Prometheus-correct;
- emitted metrics contain no event identifiers, addresses, or date labels;
- BTC values are tested without unsafe binary-float conversion at decode or
  aggregation boundaries;
- pagination, deduplication, freshness, and failure semantics are documented;
- all tests are offline and sanitized;
- docs, changelog, and handoff are updated;
- gofmt, vet, tests, race tests, build, optional linter, and diff check pass.

Final report: architecture decision, exact metrics, precision and pagination
behavior, fixture review, files changed, validation, risks, git status,
approval gates, and next hardening work. Stop after Milestone 04.
```

## Prompt 6 — Default Grafana Dashboard

```text
Work in:
C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter

Complete only Milestone 06 — Default Grafana Dashboard. Prerequisite:
Milestones 02 through 05 are complete and docs/METRICS.md reflects the exact
verified exporter output. Read applicable AGENTS.md files, handoff, metrics,
architecture, Grafana README, and dashboard contribution guidance. Inspect git
status and obtain representative, sanitized Prometheus data.

Create a reusable dashboard under grafana/ with UID
braiins-pool-exporter and title Braiins Pool Exporter. Include account 5-minute,
1-hour, and 24-hour hashrate where emitted; worker state counts; worker status
table; worker hashrate trends; last-share age; verified reward and payout
summaries; exporter/API health; and data freshness. Every query must reference
an existing documented metric and handle missing series gracefully. Use a
Prometheus datasource variable and portable job/instance filters. Provide
installation instructions, screenshots or rendered evidence, and dashboard
validation.

Do not include Avalon metrics, Smuts Tech IPs or worker aliases, CoinGecko,
wallet data, Kubernetes metrics, workshop provisioning, profitability
estimates, or nonexistent metric queries. Do not alter Helm or a live Grafana
instance. Do not make unapproved GitHub or release changes.

Acceptance criteria:
- valid importable dashboard JSON with stable UID/title;
- only exporter-emitted metrics are queried;
- no hard-coded environment-specific datasource, job, instance, IP, or worker;
- account, worker, rewards/payouts, health, and freshness sections are clear;
- variables, units, legends, thresholds, no-data states, time ranges, and
  accessibility are reviewed;
- automated JSON/query checks and a real rendered visual review are performed;
- README, changelog, and CODEX_HANDOFF.md are updated.

Final report: files, panel/query inventory, metric cross-check, visual
validation evidence, portability/security review, exact checks, git status,
approval gates, and remaining dashboard limitations. Stop after Milestone 06.
```

## Prompt 7 — Container and Release Engineering

```text
Work in:
C:\Users\Nico\dev\nicosmuts\braiins-pool-exporter

Complete only Milestone 07 — Containers and Release Engineering. Prerequisite:
the exporter and public dashboard are validated, and the canonical GitHub remote
exists. Read all applicable AGENTS.md files, handoff, development, security,
license, and contribution docs. Inspect repository settings, workflows, and
git status. Treat workflows and credentials as security-sensitive.

Add a minimal production multi-stage Dockerfile that builds a static,
non-root, read-only-friendly binary and exposes 9108 without embedding tokens.
Add .dockerignore, documented runtime examples, version/commit/build-date
injection, health behavior, and multi-architecture planning for
ghcr.io/nicosmuts/braiins-pool-exporter. Add pull-request validation and a
trusted release/image workflow with least-privilege permissions, immutable
commit-SHA tags, appropriate attestations/SBOM/provenance planning, and no
deployment credentials available to untrusted pull-request code. Deliberately
document the Dependabot/Renovate decision and release procedure.

Do not publish an image, create a tag/release, change repository visibility,
modify Helm/Kubernetes, or grant broad runner/cluster permissions unless each
action receives its own explicit approval. Do not use mutable latest tags for
deployment. Do not copy tokens or local files into images.

Acceptance criteria:
- reproducible multi-stage image builds and runs as non-root;
- image contains only required runtime artifacts and public metadata;
- amd64/arm64 build plan is validated;
- PR workflow runs format, vet, tests, race tests where practical, build, and
  dashboard checks without secrets;
- publishing workflow is trusted, least-privilege, and creates immutable tags;
- version endpoint reflects injected sanitized metadata;
- container smoke tests verify all endpoints and clean shutdown;
- security/license docs, changelog, and handoff are updated.

Final report: Docker layers/runtime identity, workflow permissions/triggers,
tag scheme, SBOM/provenance decision, exact validation and scan results, files
changed, git status, artifacts not published, approvals still required, and
Milestone 08 readiness. Stop after Milestone 07.
```

## Prompt 8 — Helm/GitOps Deployment

```text
Work only in:
C:\Users\Nico\dev\smuts-tech\helm

Complete only the GitOps Deployment phase of Milestone 09 — Workshop
Integration for Braiins Pool Exporter. Prerequisites: a reviewed immutable
image exists at
ghcr.io/nicosmuts/braiins-pool-exporter using a verified commit-SHA or release
tag; the exporter repository's public/default dashboard is validated; and the
user has explicitly authorized implementation in the Helm repository.

Read the parent workspace AGENTS.md, helm/AGENTS.md, relevant chart,
observability, Argo CD, secrets, runner, and deployment documentation. Inspect
git status and existing conventions before proposing changes. Report the
exact Helm files to be changed and the deployment sequence before editing if
required by workspace policy.

Add the smallest convention-aligned chart/configuration needed to deploy the
exporter with port 9108, resource limits, probes for /-/healthy and /-/ready,
Prometheus scraping, and the immutable image. Reference an existing
secret-management mechanism; never create, print, commit, or retrieve a live
Braiins token. Mount a Secret file where practical. Provision the reusable
public dashboard only if the repository's established mechanism supports it.
Keep permissions namespace-scoped and do not grant cluster-admin.

Prohibited actions: modifying the exporter source; using latest; embedding a
token; changing miners/pool/firmware; adding price, wallet, Avalon, or
profitability logic; applying manifests, syncing Argo CD, deploying, or
touching the live cluster without separate explicit deployment approval;
unapproved staging, commits, pushes, merges, or releases.

Acceptance criteria:
- rendered manifests use the verified immutable image and no secret value;
- probes, Service, scrape configuration, resources, security context, and
  Secret mount match repository conventions;
- public dashboard provisioning, if included, is portable and unmodified;
- helm lint/template and repository-specific validation pass;
- rendered output is reviewed for secrets and excessive RBAC;
- deployment and rollback commands are documented but not executed without
  approval;
- handoff identifies the separate live-deployment approval gate.

Final report: repository/branch, files, rendered workloads/RBAC/resources,
image existence evidence, validation results, secret contract, rollout and
rollback plan, git status, and every remaining approval. Stop before live
deployment unless that exact action was separately approved.
```

## Prompt 9 — Workshop Mining Dashboard

```text
Work in the repository that owns workshop Grafana provisioning, expected to be:
C:\Users\Nico\dev\smuts-tech\helm

Complete only the Mining Operations Dashboard phase of Milestone 09 —
Workshop Integration.
Prerequisites: Braiins Pool Exporter is deployed and scraped; Avalon exporter
metrics are available; verified price/profitability data sources and their
units are documented; and the user has authorized workshop-dashboard changes.
Read parent and repository AGENTS.md plus observability/dashboard conventions.
Inspect git status and the existing workshop mining dashboards before editing.

Design a Braiins-inspired workshop dashboard that combines pool-reported
hashrate, local miner hashrate, expected fleet hashrate, worker health,
last-share age, rewards, balance and payouts where emitted, BTC/USD and BTC/ZAR
price from a separate source, estimated fiat value, historical earnings,
projected daily/monthly/annual earnings, uptime, and clearly labeled
profitability assumptions. Preserve the standalone public dashboard; create
workshop-owned composition instead of adding local assumptions to the exporter.

Every query must be mapped to an existing metric and unit. Document formulas,
time-window alignment, missing-data behavior, pool-vs-local identity mapping,
currency conversion, and which values are authoritative versus estimated.
Keep worker/IP mappings in workshop configuration. Treat wallet monitoring as
a separate optional subproject and never request private keys or seed phrases.

Prohibited actions: changing exporter metrics to fit the dashboard without a
separate exporter issue; embedding tokens, addresses, or secrets; configuring
miners/pools/firmware; inventing price/network/hashprice data; presenting
estimates as guaranteed earnings; modifying or deploying live resources,
committing, pushing, or merging without the required separate approvals.

Acceptance criteria:
- all panels use verified present metrics and correct units;
- pool/local comparisons have explicit mappings and aligned windows;
- estimates expose their assumptions and distinguish observed values;
- no secrets, wallet keys, public-default workshop data, or unbounded labels;
- no-data, stale-data, partial-fleet, and price-feed failure states are clear;
- dashboard JSON/provisioning validates and is visually reviewed at desktop
  and operations-display sizes;
- docs include data-source prerequisites, formulas, troubleshooting, and
  rollback;
- repository-specific tests and rendered-manifest checks pass.

Final report: dashboard structure, complete query/formula mapping, data
sources, assumptions, visual evidence, files changed, exact validation, privacy
and security review, git status, rollout/rollback plan, and approvals still
required. Stop after the workshop dashboard milestone.
```
