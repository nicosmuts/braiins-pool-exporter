## Summary

## Scope

## Non-goals

## Security and metric compatibility

- [ ] No tokens, private responses, account identifiers, payout addresses, or secrets are included.
- [ ] No private worker names, private URLs, IP addresses, deployment topology, or raw API responses are included.
- [ ] New metric names, types, units, and labels are documented and bounded.
- [ ] Public defaults contain no environment-specific assumptions.

## Documentation

- [ ] README, docs, Grafana README, or changelog updated where needed.
- [ ] No planned functionality is described as already available.

## Validation

- [ ] `gofmt`
- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go build ./cmd/braiins-pool-exporter`
- [ ] `docker build`
- [ ] `docker compose config`
- [ ] Dashboard validation, if Grafana files changed

## Release impact

- [ ] No release tag or GitHub Release is created by this pull request.
- [ ] Backward compatibility and release-note impact are described when applicable.
