# Architecture and design

## Core components

### Agent (`agent/`)
Host-native CLI and optionally a systemd daemon. Owns local repository checkout(s), per-stack state, inspections and local operation dispatch. Prefer a small compiled binary (Go is the initial proposal). Docker and firewall executors are separate, opt-in capabilities. Local execution must work without controller, UI or CI provider.

### Controller (`controller/`)
Optional headless multi-host coordination: registration, mTLS authentication, desired-revision dispatch, job scheduling, status aggregation, topology/policy analysis and Git-provider adapters. No independent authoritative Compose store and no general-purpose privileged shell. FastAPI/Python is a proposal requiring a recorded decision before implementation, as is the agent's proposed Go toolchain.

### UI (`ui/`)
Optional thin client over the controller API. Cannot have private execution paths or bypass policy. The same operation must be possible from headless API/CLI. GUI supports manual groups, annotations, access view and change preview.

## Source-of-truth model
- **Remote Git**: approved tracked configuration by default. Provider may be Gitea, GitHub, GitLab, or plain Git/SSH.
- **Host-local working tree**: authoritative for host-local edits and conflicts; never silently clobber it. Uncommitted files and ahead/diverged commits are explicit states.
- **Host deployment marker/journal**: authoritative for what revision the agent last successfully applied; do not equate local `HEAD` with a deployment.
- **Runtime facts**: derived from the host on request, never copied into an authoritative Compose database.
- **Controller**: an orchestrator/view; ephemeral caches are allowed but cannot silently become sources of truth.
- **Independent Git sources**: may define a device/policy without an agent; Git write authority does not imply device apply authority.

The controller does own persistent manually modeled devices, policies, groups and annotations. This is topology baseline data, not an alternative desired Compose/Git configuration store. Agent observations and Git-declared records keep their origin and freshness; caching them does not transfer authority. Manual overrides, if supported, must remain separately attributed. Local model fixtures/evaluation precede the controller's storage/API implementation; storage and reconciliation choices remain open in [D4](plans/0001-architecture-readiness.md#open-decisions-ranked-by-impact).

## Responsibility and interface boundaries

| Boundary | Inputs and outputs | Authority |
| --- | --- | --- |
| CLI/controller to agent | Registered scope, fixed operation, exact candidate revision; versioned status or job outcome | Agent validates local registration, caller approval and enabled capability; same semantics in all profiles. |
| Agent Git core | Checkout/ref inspection, explicit object acquisition, isolated candidate staging | Unprivileged; preserves active edits; cannot claim a staged revision was deployed. |
| Agent to Docker/firewall executor | Validated registered resource and authorized fixed operation; sanitized result | Separate opt-in boundaries; executor revalidates its scope. No arbitrary shell/path passthrough. |
| UI to controller | Git edit proposals, topology edits, previews and authorized job requests | UI has no independent execution engine; editing, preview and runtime apply are distinct. |

See [DATA_MODEL](DATA_MODEL.md) for state ownership, [PROTOCOL](PROTOCOL.md) for transitions and [SECURITY](SECURITY.md) for enforcement. These are semantic contracts, not finalized wire routes or schemas.

## Transport and reconciliation
- Query remote ref with `git ls-remote` without fetch for a cheap comparison.
- Inspect local `HEAD`, dirty tree, untracked files and local deployment marker without sudo.
- Different hashes alone do not identify branch ancestry. Fetch into separate refs, compare ancestry, then stage an exact approved commit in a detached worktree or equivalent, preserving active edits.
- Validate configuration and host-local referenced paths. Do not copy secret contents into results, telemetry or artifacts.
- Apply an immutable approved revision locally; inspect health; report status; receive authenticated confirmation. Failure/timeout triggers best-effort restoration of previous configuration/image, not promise of application-data rollback.
- Queued deployments wait for maintenance windows unless specifically authorized to run now. Authorization and validation are never bypassed by "now".
- CI integrations submit revisions/operations, not arbitrary Compose documents or shell scripts to a privileged API.
- An advertised branch tip is an observation, not approval. A moving ref never silently replaces an approved commit. Unknown ancestry stays unknown until sufficient objects are available; status inspection does not fetch.

## Optional operations and profiles
- Agent-only: CLI, Git drift, local apply, optionally agent service.
- Headless: controller API plus agents; CI/CD and scripts interact directly.
- Full: same headless API, optional UI frontend.
- Optional telemetry: health, CPU and RAM; sanitize data, avoid raw environment variables.
- Explicit `stop` operation is optional for availability policy, though Compose `up` itself may replace or stop containers.

## Network view
- Inventory is hosts, interfaces, routes, VLANs, managed devices, unmanaged devices, and manually defined groups.
- Connectivity is assessed from known topology and firewall policies; absent evidence of a blocker, **assume allowed** (label as an assumption, never as measured reachable).
- Different VLAN labels alone are not proof of a block. If a path is physically known to be impossible, report no path.
- Permanent unmanaged device policies and Git-only policies are part of the normal/current model; they are not temporary simulations.
- Preview mode overlays *proposed deltas* only; deleting the preview restores the unchanged current model.
- User-drawn relationship arrows are annotations, not automatic permits/denies or discovered service dependencies.
- Avoid application-specific config parsing and don't infer dependence from network adjacency.

## Known limitations
Full network paths cannot be guaranteed from incomplete topology; runtime packet processing depends on chain order, NAT, conntrack and intermediate devices. Emitted results must include source/provenance and assumption labels. No firewall analysis should be advertised as a penetration test or proof of real connectivity.
