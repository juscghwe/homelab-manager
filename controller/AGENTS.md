# Headless controller instructions

Read root `AGENTS.md`, `docs/INTENT.md`, `docs/DESIGN.md`, `docs/DATA_MODEL.md`, `docs/PROTOCOL.md` and `docs/SECURITY.md`.

## Responsibility
Optional, provider-neutral coordination of agents and Git sources, job approval/dispatch, maintenance schedules, topology inventory, policy evaluation and what-if preview. Headless API is a first-class product; the GUI must never be required.

## Constraints
- Never become an alternate authoritative Compose configuration store. Git remote is desired configuration; agent local worktree/journal is local observed/applied state.
- Support Git-based policies even when no agent exists for the modeled device.
- Treat policy provenance, editing authorization and runtime apply authority separately.
- Load permanent manually defined devices, policies, groups and annotations from a user-selected Git repository (remote preferred, local Git supported). Any database/cache is derived and reconstructible; preview never changes Git baseline. Preserve per-record origin and separate runtime observations.
- Open-world connectivity: no known restriction => `assumed_permitted`; unknown infrastructure does not automatically block, but do not claim verified reachability.
- No application-specific dependency crawling or Caddy parsing.
- Authenticate and authorize each job against host, stack, revision, operation and expiry; prevent replay and cross-host execution.
- Avoid possessing root/Docker-socket credentials for managed hosts.
- Provide fleet CLI/API, maintenance scheduling and optional centrally managed Renovate/equivalent update discovery and notifications. These are not worker dependencies; direct CI-to-worker operations remain supported.

## Tests
Synthetic topology graph fixtures, unknown-router/default allow, known deny, persistent manual model, preview isolation, missing agents, Git-only source, request replay and unavailable hosts.
