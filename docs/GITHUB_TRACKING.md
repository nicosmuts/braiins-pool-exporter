# GitHub Tracking Bootstrap

This file is a recovery aid, not a replacement for GitHub Issues. Create the
objects below only after `gh auth status` confirms access to `nicosmuts` and
the public canonical repository is verified.

## Milestones

Create exactly ten milestones, Milestones 00 through 09, using the exact
titles in [ROADMAP.md](ROADMAP.md). Do not create a decorative project board
unless an organization convention is discovered.

Create all ten milestones open. After the completed Milestone 00 foundation
issue is created, assigned, and verified, close that issue as completed and
then close Milestone 00. Leave Milestones 01 through 09 open. There is no
Milestone 10.

## Parent issue

Title: `Track Braiins Pool Exporter from foundation to workshop integration`

Milestone: none. The parent spans the complete project and must not affect an
individual milestone's progress.

Objective: deliver a reusable, secure exporter and later integrate a published
image into the workshop without mixing public and workshop-specific concerns.

Scope: link eleven deliverable issues across ten milestones—one issue for each
of Milestones 00–08 and two issues for Milestone 09—and record cross-stage
decisions. Together with this parent, the tracking structure contains exactly
twelve issues.

Non-goals: implementation details, secret storage, and an empty project board.

Acceptance criteria:

- Milestones 00–08 each have one major deliverable issue and Milestone 09 has
  two separately reviewable deliverable issues;
- dependencies and stage gates are explicit;
- public-project and workshop scopes remain separate;
- the parent checklist links all eleven deliverable issues.

Dependencies: the canonical public repository exists and authenticated access
to `nicosmuts/braiins-pool-exporter` is verified.

Validation: compare the issue checklist with `docs/ROADMAP.md`.

Security: no token, private response, account identifier, or workshop secret is
included.

## Milestone issues

Create one issue for each of Milestones 00–08 and two issues for Milestone 09.
Each issue should use the relevant roadmap section as its objective and scope,
and contain:

- non-goals that stop at the milestone boundary;
- acceptance criteria matching that stage's outputs;
- dependencies on preceding milestone issues;
- exact test, documentation, or deployment validation;
- security considerations, especially token redaction, fixture sanitization,
  workflow permissions, and visibility review.

For Milestone 00, note that no API call, Docker image, Helm change, cluster
change, release, or public visibility change is allowed. For the two
Milestone 09 workshop issues, explicitly identify the separate Helm repository
and approval gates.

Suggested issue titles:

1. `Milestone 00: bootstrap repository foundation`
2. `Milestone 01: discover and document the official Braiins API`
3. `Milestone 02: implement the account collector`
4. `Milestone 03: implement the worker collector`
5. `Milestone 04: implement rewards and payouts`
6. `Milestone 05: harden polling, caching, and security`
7. `Milestone 06: build the default Grafana dashboard`
8. `Milestone 07: add containers and release engineering`
9. `Milestone 08: prepare the first public release`
10. `Milestone 09: deploy through workshop GitOps`
11. `Milestone 09: build the workshop mining operations dashboard`

Assign both workshop issues to Milestone 09, preserving deployment and
dashboard work as separately reviewable phases.

## Additional labels

Retain the nine existing GitHub default labels unchanged. Add only these three
labels when the tracking manifest is approved for creation:

- `security`: secret handling, privacy, permissions, or supply-chain review;
- `deployment`: container publishing, release delivery, or runtime deployment;
- `workshop`: work owned by separate workshop infrastructure or configuration.

Do not add a `milestone`, `observability`, or `tracking` label. Milestone
assignment already represents milestone membership, the entire project is
observability work, and the parent issue does not justify a one-off label.
