# Agent module instructions

Read root `AGENTS.md`, `docs/DESIGN.md`, `docs/SECURITY.md`, `docs/DATA_MODEL.md` and `docs/PROTOCOL.md` before changing this module.

## Responsibility
Host-local Git worktree inspection, registered stack resolution, local execution coordination, deployment journal, machine-readable status and optional systemd service. Agent-only CLI must work without controller, UI or hosted Git platform. Approved implementation: lightweight Go binary, system Git, versioned JSON CLI/API output and optional systemd background operation. A small authenticated remote operation interface lets CI trigger the worker directly without the central controller.

## Constraints
- Git operations are unprivileged and never overwrite dirty/unpublished local changes.
- `ls-remote` only advertises refs; it does not transfer commit objects. Fetch and validate exact approved commits before staging/applying.
- Never confuse `HEAD` with last successfully deployed revision.
- Default status is minimal. Runtime hashes/health/CPU/RAM depend on enabled local capabilities.
- Do not read `.env` contents or volume contents to display/configure services.
- No direct arbitrary shell or dynamic Docker flags from remote API; local executor validates configured stack IDs and fixed paths.
- Setup installs root-owned, agent-non-writable fixed-operation helpers and authorization rules. The agent has neither general sudo nor direct Docker socket access. Helpers resolve registered identifiers to protected paths and reject arbitrary flags/commands.
- Keep Docker and nftables execution privileges independent. `compose up` may stop/replace containers, despite an optional explicit `stop` command.
- Maintain durable, idempotent transaction journal and operation audit; failed deployment may only perform best-effort container/config rollback.

- Image-update discovery/Renovate belongs to the optional central backend, never the worker.
- Record execution, configured health checks and confirmation separately; Compose exit zero is not verified application health.
- Offline status/history is mandatory. Explicit local-administrator deployment of a complete available revision uses normal approval, validation and conflict checks; never turn remote failure into automatic offline apply.

## Tests
Fixture-based Git divergence, dirty/untracked files, staged commit, idempotent job retries, missing/malicious paths, controller outage and sanitized status output. Privileged tests run in disposable environments only.
