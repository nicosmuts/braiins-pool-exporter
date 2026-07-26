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
- Give future HTTP clients explicit timeouts and context cancellation.
- Sanitize errors and URLs; do not log response bodies by default.
- Keep fixtures synthetic or rigorously sanitized.
- Keep labels bounded and free of private financial identifiers.

## Required future tests

Milestone 01 must add token-redaction tests covering request construction, URL
sanitization, transport errors, non-2xx responses, decoding errors, and panic
or debug formatting. Fuzzing URL/error sanitizers should be considered.

Before a public release, review Git history, repository settings, fixtures,
dependencies, container contents, workflow permissions, and documentation for
secrets and private deployment data.
