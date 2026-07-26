# Examples

Run without a token for self-metrics only:

```sh
go run ./cmd/braiins-pool-exporter
```

For authenticated account polling, prefer a mounted file:

```sh
export BRAIINS_POOL_TOKEN_FILE=/run/secrets/braiins-pool-token
go run ./cmd/braiins-pool-exporter
```

An environment variable is supported for environments where a mounted file is
not practical:

```sh
export BRAIINS_POOL_TOKEN='replace-at-runtime'
go run ./cmd/braiins-pool-exporter
```

Never put a real token in this repository, shell history, a command-line flag,
an image, or an example file. With a token configured, account metrics appear
after the first successful profile poll, and worker metrics appear after the
first successful worker poll.

To disable direct worker labels while keeping account metrics:

```sh
export BRAIINS_POOL_WORKER_METRICS_ENABLED=false
go run ./cmd/braiins-pool-exporter
```

Production deployment is operator-managed. Future example configuration must
use placeholders, `example.com` domains, or RFC 5737 documentation addresses
instead of deployment-specific secrets, worker mappings, URLs, or topology.
