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
