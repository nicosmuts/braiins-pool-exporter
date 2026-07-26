## Objective

## Scope and non-goals

## Security and metric compatibility

- [ ] No tokens, private responses, account identifiers, payout addresses, or secrets are included.
- [ ] New metric names, types, units, and labels are documented and bounded.
- [ ] Public defaults contain no workshop-specific assumptions.

## Validation

- [ ] `gofmt`
- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go build ./cmd/braiins-pool-exporter`
- [ ] Documentation and changelog updated where needed
