# Configuration

Braiins Pool Exporter supports command-line flags for local HTTP/logging
behavior and environment variables for Braiins Pool API collection. Runtime
configuration is immutable after startup.

## Token sources

Authenticated Braiins Pool API polling is enabled only when exactly one token
source is set.

| Setting | Default | Description |
|---|---:|---|
| `BRAIINS_POOL_TOKEN` | unset | Token value, convenient for local development. |
| `BRAIINS_POOL_TOKEN_FILE` | unset | Path to a file containing the token, preferred for production-style runtime. |

Set only one of these variables. If both are blank, the exporter starts without
making external Braiins Pool API requests and exposes only self-metrics.

Token command-line flags are intentionally unsupported because process listings
can expose command arguments.

## Environment variables

| Setting | Default | Description |
|---|---:|---|
| `BRAIINS_POOL_COIN` | `btc` | Coin selector. Only `btc` is currently verified and accepted. |
| `BRAIINS_POOL_API_BASE_URL` | official API origin | Override for tests or compatible endpoints. |
| `BRAIINS_POOL_POLL_INTERVAL` | `1m` | Interval between completed poll cycles. |
| `BRAIINS_POOL_TIMEOUT` | `10s` | Per-request HTTP timeout. |
| `BRAIINS_POOL_WORKER_METRICS_ENABLED` | `true` | Enable worker metrics when a token is configured. |
| `BRAIINS_POOL_MAX_WORKERS` | `100` | Maximum accepted workers per snapshot. |
| `BRAIINS_POOL_REWARDS_ENABLED` | `true` | Enable bounded rewards metrics. |
| `BRAIINS_POOL_PAYOUTS_ENABLED` | `true` | Enable bounded payout metrics. |
| `BRAIINS_POOL_HISTORY_DAYS` | `7` | Inclusive rewards and payouts history window, capped at 90 days. |

## Command-line flags

| Flag | Default | Description |
|---|---:|---|
| `--web.listen-address` | `:9108` | HTTP listen address. |
| `--web.telemetry-path` | `/metrics` | Metrics path. |
| `--log.level` | `info` | Log level: `debug`, `info`, `warn`, or `error`. |
| `--log.format` | `text` | Log format: `text` or `json`. |
| `--config.file` | empty | Reserved until a safe configuration format is defined. |

There is no `BRAIINS_POOL_LISTEN_ADDRESS` environment variable. Use
`--web.listen-address` when changing the HTTP listen address.

## Docker Compose

Copy the example environment file:

```sh
cp .env.example .env
```

For token-free validation, leave both token settings blank:

```env
BRAIINS_POOL_TOKEN=
BRAIINS_POOL_TOKEN_FILE=
```

For local authenticated development, set:

```env
BRAIINS_POOL_TOKEN=<token>
```

For production-style token-file handling in Compose:

```sh
mkdir -p secrets
printf '%s' '<token>' > secrets/braiins_pool_token
```

Then set:

```env
BRAIINS_POOL_TOKEN_FILE=/run/secrets/braiins_pool_token
```

Files under `secrets/` are ignored except for `secrets/README.md` and
`secrets/.gitkeep`.

## API token guidance

Create a Braiins Pool API token from the Braiins Pool account settings and use
the least-privileged access profile that can read account, worker, reward, and
payout data required by this exporter. Rotate tokens periodically and whenever
access might have been exposed.

Never commit tokens, `.env`, local secret files, raw API responses, payout
addresses, transaction identifiers, or private worker mappings.
