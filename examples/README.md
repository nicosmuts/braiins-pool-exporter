# Examples

Milestone 00 needs no token:

```sh
go run ./cmd/braiins-pool-exporter
```

For future authenticated polling, prefer a mounted file:

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
an image, or an example file. The token is optional until API integration
begins.
