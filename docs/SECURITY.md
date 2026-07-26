# Security Design

## Protected data

The primary secret is the read-only Braiins API token. API responses may also
contain account identifiers, worker names, balances, payout addresses,
transaction identifiers, and financial history that operators consider
private.

The exporter must never request wallet private keys or seed phrases.

## Trust boundaries

- Environment variables and mounted files enter the local process.
- The future HTTP client sends authenticated requests only to a configured,
  validated API origin.
- Prometheus and operators read the exporter's unauthenticated HTTP interface.
- Logs, metrics, panic output, test fixtures, and Git history may become public.

The HTTP interface therefore exposes only sanitized health/build information
and reviewed metric labels. Network-level access control is a deployment
responsibility.

## Controls

- Accept a token through exactly one of `BRAIINS_POOL_TOKEN` or
  `BRAIINS_POOL_TOKEN_FILE`.
- Reject URLs containing user info, query strings, or fragments.
- Never support a token command-line flag.
- Never format the secret-bearing configuration directly in logs.
- Use a redacting secret type and an explicit safe summary.
- Give HTTP clients explicit timeouts and context cancellation.
- Sanitize errors and URLs; do not log response bodies by default.
- Do not follow Braiins API redirects; redirect targets are not part of the
  verified API contract.
- Keep fixtures synthetic or rigorously sanitized.
- Keep labels bounded and free of private financial identifiers.

Milestone 02 account collection exposes only account-level metrics approved in
`docs/METRICS.md`. It does not expose usernames, account identifiers, worker
names, payout addresses, transaction IDs, raw endpoint URLs, authorization
headers, or arbitrary error strings. The API request result label is a bounded
category enum.

Milestone 03 worker collection exposes API worker names directly as the
`worker` label after an explicit privacy review. Worker names are
operator-controlled and may be sensitive, so operators must not expose the
exporter's HTTP interface publicly. The exporter does not include
deployment-specific alias mappings or worker-to-device mappings. Worker label
cardinality is bounded by `BRAIINS_POOL_MAX_WORKERS`, label length is capped,
blank labels are rejected, and unknown worker states are mapped to the bounded
`unknown` state instead of becoming arbitrary labels.

Milestone 04 rewards and payouts collection exposes only bounded summary
labels: reward component, payout rail, and normalized payout status. Reward
dates, payout timestamps, payout destinations, transaction IDs, Lightning
invoices, preimages, financial account names, and event identifiers are used
neither as labels nor as public fixture values. Deduplication keys are internal
to snapshot construction and are not logged or exported.

Milestone 05 hardening keeps retry and rate-limit handling privacy-safe. Retry
decisions use typed status, transport, timeout, decode, and validation errors;
response bodies, full URLs, authorization headers, `Retry-After` header values,
worker names, payout identifiers, and raw arbitrary errors are not labels.
HTTP 429 is exposed only as `rate_limited`, and malformed rate-limit headers
fall back to a conservative bounded delay without being logged or exported.

Milestone 06 adds the reusable default Grafana dashboard without embedding
private runtime values. The dashboard JSON contains no datasource UID, job,
instance, IP address, worker name, account identifier, payout destination,
transaction identifier, token, live query result, or production URL. Runtime
worker labels can still be visible to Grafana users because the exporter emits
the Braiins API worker name as the `worker` label; operators must treat the
dashboard as private operational telemetry if worker names are sensitive.

Milestone 07 adds container and release engineering without embedding secrets.
The Docker build context excludes `.env`, local secret files, `SECRETS.md`,
build outputs, and transient logs. Docker Compose supports token-free
validation, environment-token development, and token-file operation through the
ignored `secrets/` directory. GitHub Actions use the repository-scoped
`GITHUB_TOKEN` for GHCR publishing only on semantic version tag pushes; ordinary
pull requests and pushes to `main` validate but do not publish images or create
releases.

Focused security review findings:

- Accepted: tokens are accepted only through environment or mounted file and
  remain redacted in configuration summaries.
- Accepted: custom API base URLs are validated as absolute HTTP(S) origins
  without credentials, queries, or fragments; redirects are refused.
- Accepted: HTTP response bodies are read only for decoding successful
  responses and are never included in status, decode, or retry errors.
- Fixed: transient retry, 429, and response-size failures now map to bounded
  categories rather than raw error text.
- Fixed: rewards and payouts now have explicit 1,000-record per-window caps to
  bound deduplication memory.
- Accepted: the default dashboard uses only documented exporter metrics and
  stores no deployment-specific values.
- Accepted: container workflow permissions are split between read-only CI and
  tag-gated release publishing with package/write and contents/write.
- Accepted: Dependabot monitors Go modules, GitHub Actions, the Dockerfile, and
  Docker Compose files on a conservative weekly schedule.
- Deferred: final release-contents review still belongs to the first stable
  release milestone before any tag is created.

## Required future tests

Milestone 01 adds token-redaction tests covering request construction, unsafe
URLs, transport errors, non-2xx responses, decoding errors, and formatting.
Fuzzing URL/error sanitizers should still be considered later.

Live API validation must use only `BRAIINS_POOL_TOKEN` or
`BRAIINS_POOL_TOKEN_FILE`. A repo-local ignored file such as `SECRETS.md` is
not an implicit credential source for automated discovery. If live responses
are captured later, keep them outside the repository until every field is
sanitized and reviewed.

Before a public release, review Git history, repository settings, fixtures,
dependencies, container contents, workflow permissions, and documentation for
secrets and private deployment data.
