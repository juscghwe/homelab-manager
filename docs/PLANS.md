# Codex work plan and milestone contracts

Before implementing nontrivial functionality, add `docs/plans/<NNNN>-<topic>.md` with purpose, affected contracts, design, security assumptions, unit/integration tests, acceptance checks and recovery approach. Complete one vertical slice per PR. Keep open decisions explicit; avoid turning guesses into permanent architecture.

## Branch and PR integration

`main` receives completed, reviewed milestones. Keep one integration branch per active milestone, named `milestone/<number>-<kebab-case-goal>`, created from `main`. Create it when work on the milestone starts rather than pre-creating branches for the entire roadmap.

The current integration branch is `milestone/1-agent-git-core`; it collects the architecture/readiness documentation needed by Milestone 1 as well as its implementation slices. Documentation alone does not complete the milestone or establish that any runtime behavior works.

Create focused topic branches from the milestone, for example `feat/agent-git-status`, `fix/git-ancestry`, or `docs/record-maintainer-decisions-and-git-core-plan`. Their PRs target `milestone/1-agent-git-core`. Reuse a topic branch and its PR for related follow-up work, including supporting fixes and tests. Existing documentation PR #5 is the initial documentation contribution to this milestone.

Branch names use `<type>/<kebab-case-subject>`; commit subjects and PR titles use `<type>: <subject>`. Topic types are `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `perf`, `style`, and `chore`. A feature PR keeps its `feat:` title even when it contains related `fix:`, `test:`, or `docs:` commits. Generic `work`/`temp`/`patch` branches and duplicate replacement branches are not part of this workflow.

When the milestone's acceptance criteria are verified, open one integration PR from its milestone branch to `main`, titled for example `milestone: complete agent Git core`. Record the actual verification and remaining limitations; require human review and do not auto-merge. Use a typed subject for any merge or squash commit as well. Delete merged topic branches after integration, and the merged milestone branch after completion once no open PR depends on it. Preserve all unmerged work.

## Milestones

### 0 — Docs and contracts
Freeze terminology, API contracts and ownership boundaries. Add fixture specifications, no privileged code. Deliver docs-first PR.

The [architecture readiness audit](plans/0001-architecture-readiness.md) records findings, invariant coverage and maintainer-approved D1–D7 intent and concrete proposals. Go/system Git/JSON/systemd, Git-backed manual topology, narrow helpers and explicit authorized offline deployment are approved. Privileged mechanisms still require their specific review and tests.

### 1 — Agent-only Git core
CLI, local revision markers, dirty/tree checks, `ls-remote`, staged immutable revision, multi-repo stack mapping and fixture tests. Works as unprivileged user; no controller.

**First slice:** follow [0002-agent-git-core](plans/0002-agent-git-core.md), using approved Go/system Git/versioned JSON, then implement read-only status against registered repository roots and tracked branch refs. The plan specifies proposed CLI syntax, versioned output and exit/error behavior for implementation review. Inputs are registered host/repository/stack identities and paths; outputs are the scoped observations in [DATA_MODEL](DATA_MODEL.md#agent-status-minimum-meaningful-fields). Status may query an advertised remote ref but never fetch, stage or update deployment markers. Suppress Git's optional index-refresh writes (for example, `git --no-optional-locks status`) and verify index preservation in fixtures. No controller, Docker or firewall dependency is required.

**Following slice:** explicit object acquisition, ancestry validation and isolated immutable candidate staging. Preserve local working trees and unpublished commits; distinguish candidate records from per-stack applied records. Milestone 1 can read/fixture-test applied records but cannot manufacture successful deployment records without an executor. Review the proposed registration/state serialization and staging strategy before persistence is implemented. D7 approves explicit local-administrator deployment of complete available revisions with normal validation/approval, never automatic remote-error fallback. Offline inspection is required in this first slice; actual apply remains an executor milestone.

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

Here, `stop` means a separate explicit stop command. Authorized Compose `up` can still replace/stop containers; do not promise otherwise. D1/D3/D6 intent is approved; review exact helper authorization, health/confirmation policy and safe fingerprint mechanisms before implementation.

### 3 — Firewall inventory, model and executor
Read Linux interfaces/routes/nftables. Persistent manual and Git-only devices. Optimistic path evaluator with provenance. Read-only preview. Separate nftables executor limited to owned tables; durable rollback timer and fresh-channel verification. Test only in disposable network namespaces/VMs.

Implement the D4 Git-backed baseline and preview contract with fixture-based evaluation here. Central aggregation, Git-provider edit APIs and multi-host composition arrive in 4 with the same semantics. D1/D2 intent is approved; review the concrete helper and independent systemd/boot recovery proposals before firewall execution. No controller dependency for host-local recovery.

### 4 — Headless controller/API
mTLS enrollment, controller-agent job protocol, maintenance scheduler, Git provider adapters, topology access engine, dependency-free CI CLI integration. Keep server-side Git caching non-authoritative, including manual topology. Add optional centrally managed Renovate/equivalent discovery and notifications; workers remain independent. The worker's small authenticated direct-CI interface is a separate reviewed slice and must not require this central backend.

### 5 — Optional UI
Graphical inventory, groups and annotation edges; current model versus preview changes; host status, Git diff/editor, deployed rev view; operations through controller APIs only. Include clear source and certainty labels.

## Definition of done for each milestone
- Tests implement relevant `docs/ACCEPTANCE.md` criteria and pass in the available test environment.
- Security assumptions, API changes and known limitations are documented.
- Headless/agent-only modes remain usable where applicable.
- No misleading assertion of success for untested runtime behavior.
