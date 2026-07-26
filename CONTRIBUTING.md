# Contributing

Thank you for helping improve Braiins Pool Exporter.

## Before opening a change

1. Search existing issues and milestones.
2. Discuss metric names and API semantics before implementing a new family.
3. Never attach real tokens, account identifiers, wallet addresses, payout
   addresses, or sensitive API responses to an issue or commit.
4. Keep workshop-specific integrations outside the reusable exporter.

## Development

Use Go 1.26.4 and run:

```sh
go mod tidy
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/braiins-pool-exporter
```

If `golangci-lint` is installed, also run `golangci-lint run`.

Changes should include focused tests. Prefer the standard library,
table-driven tests, explicit timeouts, deterministic metric descriptors, and
small packages with clear ownership.

## Pull requests

- Use a focused branch and Conventional Commit-style title.
- Explain the objective, non-goals, security impact, and validation performed.
- Update user-facing docs and `CHANGELOG.md` where appropriate.
- Do not combine unrelated cleanup with a functional change.
- Do not add generated secrets, local environment files, or unsanitized
  fixtures.

By contributing, you agree that your contribution is licensed under
Apache-2.0.
