# Homelab Manager (working title)

Git-native, modular management of Linux hosts, Docker Compose deployments, and network access policies.

> **Status: architecture scaffold only.** No production implementation or tested privileged operations exist yet.

## Deployment profiles

- **Standalone / Agent-only:** lightweight unprivileged Go worker using system Git, a CLI and small authenticated remote operation interface with versioned JSON. Optional systemd background operation, Docker status/deployment and firewall operations via narrow installed helpers. CI can trigger it directly without a controller.
- **Headless:** adds the central backend and fleet CLI/API, coordinating agents, schedules, Git-provider integration and topology evaluation. Optional Renovate/equivalent image-update discovery is managed alongside the backend, never required by workers.
- **Full:** headless deployment plus an optional graphical editor for configuration, network-access maps, and change previews.

**Primary workflow:** remote Git repository (Gitea, GitHub, GitLab, or plain SSH). **Git itself is required; a Git hosting product and CI runner are not.** CI systems may call the same API/CLI as the UI. Existing services continue running when the controller or Git remote is unavailable.

## Documentation map

1. [`docs/INTENT.md`](docs/INTENT.md) — product intent and boundaries.
2. [`docs/DESIGN.md`](docs/DESIGN.md) — architecture and deployment flows.
3. [`docs/SECURITY.md`](docs/SECURITY.md) — privilege and authority model.
4. [`docs/DATA_MODEL.md`](docs/DATA_MODEL.md) — canonical terms and status semantics.
5. [`docs/PROTOCOL.md`](docs/PROTOCOL.md) — communication contracts and rollout transaction.
6. [`docs/ACCEPTANCE.md`](docs/ACCEPTANCE.md) — required tests for V1.
7. [`docs/PLANS.md`](docs/PLANS.md) — implementation sequencing for Codex.
8. [`docs/plans/0001-architecture-readiness.md`](docs/plans/0001-architecture-readiness.md) — audit findings, invariant coverage and decisions requiring approval.

Agent instructions live in [`AGENTS.md`](AGENTS.md) and module-level `AGENTS.md` files.

## Guardrails

- Do not put `.env` contents, secrets, or persistent data volumes into Git.
- Do not provide a remotely exposed general Docker socket or arbitrary host shell.
- Docker Compose deployment on a rootful daemon is effectively host-administrative; keep it explicitly authorized.
- The topology graph represents **network access policy**, not inferred application dependencies.
- Manually maintained topology is **persistent in Git**, including unmanaged device policies. Preview mode applies only to proposed edits; runtime observations and caches remain separate.
- No deployment or firewall mutation is allowed while drift/conflict is unresolved, without an explicit conflict-resolution operation.

## First milestone

Begin with the [Milestone 1 read-only Git status slice](docs/PLANS.md#1--agent-only-git-core), the [implementation plan](docs/plans/0002-agent-git-core.md) and its proposed interfaces. Go, system Git, JSON and optional systemd are approved. Implement the host-local Git state model and CLI with fixtures and tests **before** writing privileged Docker and firewall executors. Examples are illustrative contracts, not runnable deployment configuration or finalized schemas.
