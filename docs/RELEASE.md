# Release process

This project has not published a stable production release yet. The `v0.0.x`
line is for public review, local validation, and early integration testing. It
is not a production-stability
declaration, and interfaces, configuration behavior, image metadata, and
Prometheus metrics may still change before `v0.1.0`.

## Versioning policy

The project intends to follow Semantic Versioning for public release tags.

- The first public development release is `v0.0.1`.
- The first planned stable pre-1.0 compatibility target is `v0.1.0`.
- Tags must use the `vMAJOR.MINOR.PATCH` form, for example `v0.1.0`.
- Metric names, types, units, and label sets are treated as compatibility
  contracts for a released line, but the `v0.0.x` development line may still
  change before `v0.1.0`.
- Breaking metric or configuration changes require clear changelog and release
  notes.

## v0.0.1 release notes

`v0.0.1` is the initial public development release of Braiins Pool Exporter. It
includes account, worker, reward, and payout collectors for the official Braiins
Pool API; hardened polling, caching, retry, backoff, and rate-limit behavior;
the default Grafana dashboard; a production-oriented Docker image; and a Docker
Compose stack with exporter, Prometheus, and Grafana.

The release also includes CI validation, Linux race testing, Dependabot
coverage for project dependencies, tag-gated GHCR publishing, and BuildKit SBOM
and provenance attestations for the published container image. It remains an
initial development release: do not treat it as a stable production interface,
and expect possible metric or configuration refinements before `v0.1.0`.

## Image policy

The canonical image path is:

```text
ghcr.io/nicosmuts/braiins-pool-exporter
```

Release images are built only by `.github/workflows/release.yml` from
`v*.*.*` tag pushes. The workflow publishes:

- the exact semantic version tag;
- the `major.minor` tag;
- the commit-derived `sha-<short-sha>` tag;
- `latest`.

Operators should prefer immutable version tags or image digests for deployments.
The `latest` tag is a convenience pointer for discovery and local testing, not
a production deployment pin.

## Build metadata

The release workflow injects sanitized build metadata into the `/version`
endpoint and `braiins_pool_exporter_build_info` metric:

- `VERSION` from the Git tag;
- `COMMIT` from the Git commit SHA;
- `BUILD_DATE` from the workflow clock in UTC.

Never inject configuration values, tokens, usernames, payout details, or other
runtime data into build metadata.

## Supply-chain metadata

The release workflow requests BuildKit SBOM and provenance attestations for the
published container image. Release artifact checksums are not yet produced
because the current release workflow publishes only the container image and
GitHub-generated release notes.

## Pre-release checklist

Before creating any release tag:

1. Confirm `main` is clean, synchronized with `origin/main`, and green in CI.
2. Run the full validation sweep from `docs/DEVELOPMENT.md`.
3. Confirm `go mod tidy` produces no diff.
4. Review `go.mod`, `go.sum`, Docker base images, Compose images, and GitHub
   Actions for stale or unsupported versions.
5. Review tracked files and recent Git history for secrets, private account
   data, worker names from live environments, payout addresses, transaction
   IDs, private URLs, IP addresses, and local-only files.
6. Confirm `.env`, `SECRETS.md`, and `secrets/*` are ignored and excluded from
   Docker build context.
7. Validate Docker build, tokenless Compose startup, Prometheus scraping,
   Grafana datasource provisioning, and dashboard provisioning.
8. Confirm README, configuration, security, development, metrics, dashboard,
   and changelog documentation match actual behavior.
9. Confirm the `Unreleased` changelog content is suitable for the release.
10. Create the release tag only after explicit approval.

## Release procedure

After approval:

```sh
git fetch origin
git checkout main
git pull --ff-only
git tag -a v0.0.2 -m "v0.0.2"
git push origin v0.0.2
```

The tag push triggers the release workflow. Do not manually publish images or
create an additional GitHub Release outside the workflow unless a failed
release requires an explicitly approved recovery action.

## Rollback and recovery

If a release workflow fails before publishing, fix the issue on `main`, delete
the local failed tag if needed, and recreate the tag only with explicit
approval.

If a release publishes an image or GitHub Release with incorrect content, stop
and document the exact artifact, tag, and exposure. Do not rewrite public tags
or delete artifacts without a separate remediation decision.
