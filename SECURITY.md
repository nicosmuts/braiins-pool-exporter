# Security Policy

## Supported versions

There is no supported release yet. Security fixes are applied to the default
branch during initial development.

## Reporting a vulnerability

Use the repository's private vulnerability reporting or GitHub Security
Advisory feature. Do not open a public issue containing exploit details,
tokens, account identifiers, payout addresses, or other sensitive data.

Include a minimal description, affected revision, reproduction steps, and
impact. Replace all real secrets and personal identifiers with obvious
placeholders.

## Secret handling

The exporter never needs wallet private keys or seed phrases. A Braiins API
token must be supplied only through `BRAIINS_POOL_TOKEN` or a read-only file
referenced by `BRAIINS_POOL_TOKEN_FILE`. Do not put tokens in command-line
arguments, metrics, logs, URLs, fixtures, or committed configuration.

See [docs/SECURITY.md](docs/SECURITY.md) for the engineering threat model.
