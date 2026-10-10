# Codex work plan and milestone contracts

Before implementing nontrivial functionality, add `docs/plans/<NNNN>-<topic>.md` with purpose, affected contracts, design, security assumptions, unit/integration tests, acceptance checks and recovery approach. Complete one vertical slice per PR. Keep open decisions explicit; avoid turning guesses into permanent architecture.

## Milestones

### 0 — Docs and contracts
Freeze terminology, API contracts and ownership boundaries. Add fixture specifications, no privileged code. Deliver docs-first PR.

The [architecture readiness audit](plans/0001-architecture-readiness.md) records findings, invariant coverage and ranked approval gates. Semantic contracts are clarified; proposed languages, wire/storage schemas and privileged recovery mechanisms are not approved merely by appearing in documentation.

### 1 — Agent-only Git core
CLI, local revision markers, dirty/tree checks, `ls-remote`, staged immutable revision, multi-repo stack mapping and fixture tests. Works as unprivileged user; no controller.

**First slice:** write `docs/plans/0002-agent-git-core.md`, obtain the D5 language/output decision, then implement read-only status against registered repository roots and tracked branch refs. Specify the CLI syntax, versioned machine-readable output and exit/error behavior in that plan before coding. Inputs are registered host/repository/stack identities and paths; outputs are the scoped observations in [DATA_MODEL](DATA_MODEL.md#agent-status-minimum-meaningful-fields). Status may query an advertised remote ref but never fetch, stage or update deployment markers. Suppress Git's optional index-refresh writes (for example, `git --no-optional-locks status`) and verify index preservation in fixtures. No controller, Docker or firewall dependency is required.

**Following slice:** explicit object acquisition, ancestry validation and isolated immutable candidate staging. Preserve local working trees and unpublished commits; distinguish candidate records from per-stack applied records. Milestone 1 can read/fixture-test applied records but cannot manufacture successful deployment records without an executor. Approve registration/state serialization and staging strategy before persistence is implemented. No-remote apply policy remains D7; offline local inspection is required.

Required fixtures (throwaway local repositories/remotes; no hosting account or real infrastructure required):

| Fixture | Expected observable result |
| --- | --- |
| Clean equal local/remote commits; no applied record | Git `synced`; applied revision unknown; no runtime success claim. |
| Staged, unstaged and non-ignored untracked files | Dirty categories, no content leak, original index/tree preserved. |
| Ignored secret/data files | Contents never read for status/diff; no candidate inclusion. |
| Ahead, behind, diverged and unrelated histories | Correct classification only with sufficient objects/ancestry; preserve local commits. |
| Missing remote ref, unavailable remote, unborn checkout, shallow/missing history | Explicit unavailable/unknown observations; local inspection still useful; no invented sync/divergence. |
| Remote ref advances after candidate approval | Exact approved commit stays pinned; no replacement by new tip. |
| Multiple stacks sharing a repository; multiple repositories per host | Correct scope and distinct applied records; repository-wide conflict checks. |
| Missing/escaping/symlink paths, option-like arguments and configured external Git helpers | Reject unsafe inputs; no shell/helper execution or secret output. |
| Explicit acquisition/staging, including failure | Status itself never fetches; candidate staging preserves active tree and applied records. |

Test the actual CLI and its output/errors as well as internal state logic. Use temporary Git remotes for deterministic tests; provider-specific adapters are deferred. This milestone verifies Git behavior, not deployment, rollback or live connectivity. Sanitized content-diff exposure needs an approved policy before implementation; status metadata can proceed independently.

### 2 — Docker executor
Opt-in Docker status/deploy operations, fixed stack allowlists, audited local privileged boundary, image ID/config hashing, pull/apply/recreate behavior and rollback journal. No implicit stop permission or blanket sudo.

Here, `stop` means a separate explicit stop command. Authorized Compose `up` can still replace/stop containers; do not promise otherwise. D1/D3/D6 gate privileged execution, confirmation and safe fingerprints/change-scope analysis.

### 3 — Firewall inventory, model and executor
Read Linux interfaces/routes/nftables. Persistent manual and Git-only devices. Optimistic path evaluator with provenance. Read-only preview. Separate nftables executor limited to owned tables; durable rollback timer and fresh-channel verification. Test only in disposable network namespaces/VMs.

Define local baseline/preview contracts and fixture-based evaluation here; choose their representation under D4. Controller persistence/API and multi-host composition arrive in 4 using these same semantics. D1/D2 must be resolved before firewall execution; no controller dependency for host-local recovery.

### 4 — Headless controller/API
mTLS enrollment, controller-agent job protocol, maintenance scheduler, Git provider adapters, topology access engine, dependency-free CI CLI integration. Keep server-side Git caching non-authoritative.

### 5 — Optional UI
Graphical inventory, groups and annotation edges; current model versus preview changes; host status, Git diff/editor, deployed rev view; operations through controller APIs only. Include clear source and certainty labels.

## Definition of done for each milestone
- Tests implement relevant `docs/ACCEPTANCE.md` criteria and pass in the available test environment.
- Security assumptions, API changes and known limitations are documented.
- Headless/agent-only modes remain usable where applicable.
- No misleading assertion of success for untested runtime behavior.
