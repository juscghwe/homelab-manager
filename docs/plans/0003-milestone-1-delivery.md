# Milestone 1 — delivery and review plan

Status: planning approved in PR #7; M1-A registration is implemented for review in its feature branch. M1-B through M1-F remain unimplemented. See the [M1-A implementation plan](0004-agent-registration.md) and [agent guide](../../agent/README.md). The umbrella issue linked below is the permanent checklist; proposed mechanisms in [plan 0002](0002-agent-git-core.md) retain their individual review gates.

Permanent tracking bundle: [Milestone 1 — Agent Git Core](https://github.com/juscghwe/homelab-manager/issues/6). Update its child-PR checklist when work starts, changes base or merges.

## Objective and boundaries

Deliver the unprivileged Go/system-Git core in six cohesive feature PRs: registration, Git inspection, status interface, stack state, immutable preparation, and preparation interface. Include tests and affected contracts in each unit. Standalone CLI use must not require a controller, UI, daemon, hosted Git product, Docker, or root.

Non-goals: Docker or firewall execution, deployment-success writers, privileged installation, remote API/enrollment, controller, UI, update discovery, arbitrary content diff output, and real-infrastructure tests. Preparation never means deployed. The planning task is complete; each implementation unit is separately authorized and reviewed.

## Canonical baseline and integration

Canonical repository: `juscghwe/homelab-manager`. Audit on 2026-10-10 confirmed:

- Stable `main`: `c054442666b1bcd9330cd6b1c142b4b82f7b57e4`.
- Existing integration branch: `milestone/1-agent-git-core` at `7fd1e2a7cf0eebadaca290a1870c97c0ecc49230`.
- [PR #5](https://github.com/juscghwe/homelab-manager/pull/5) merged the architecture and planning documentation into that milestone. D1–D7 and plans 0001/0002 are present. Their absence from `main` is intentional; no recovery PR is needed.

Create topic branches from the latest milestone branch and target it with every M1 feature PR. Use the existing integration branch as the cumulative implementation bundle. Keep `main` unchanged until the complete milestone passes acceptance and its final `milestone: complete agent Git core` PR is reviewed. Never merge automatically.

A genuine dependency on an unmerged predecessor may temporarily use that predecessor branch as a clearly labeled stacked PR base. Record its PR, base and retargeting steps in the child PR and umbrella issue. Once the predecessor merges into the milestone, update the child to target `milestone/1-agent-git-core`, verify the diff contains only its intended feature, and rerun affected checks. Do not force-push shared history; coordinate any history reconciliation that cannot be done without rewriting it. Prefer waiting for prerequisites to merge rather than creating tangled stacks.

No aggregate PR existed at audit time. A draft overview may later run from this same milestone branch to `main`; reuse it if one exists, keep its checklist current with child merges, and never merge it before acceptance. Do not create another integration branch or a duplicate overview. The umbrella issue provides the complete review view in the meantime.

## Approved invariants

The [D1–D7 decision register](0001-architecture-readiness.md#maintainer-decisions-and-remaining-approval-gates) remains authoritative:

- D1: unprivileged worker; independent root-owned narrow Docker/firewall helpers, no broad sudo or Docker socket. Approved Compose can still be root-equivalent.
- D2: owned firewall scope, durable independent rollback, fresh authenticated transaction-bound confirmation.
- D3: standalone worker, optional central coordination and GUI over equivalent operations; worker-owned journal and distinct execution/health/confirmation outcomes.
- D4: manual topology baseline in user-selected Git; observations and reconstructible caches remain separate; previews do not mutate baseline.
- D5: Go, system Git, versioned JSON, optional systemd. No new language approval is required.
- D6: explicit non-secret per-stack/shared tracked inputs, conservative uncertainty, separate Git/configuration/runtime identities; no secret-derived output or hashes.
- D7: offline inspection/history and continued workloads; explicit local authorization for complete offline revisions, never automatic fallback after remote failure.

Preserve dirty edits and unpublished commits; keep HEAD, advertised remote, candidate and per-stack applied revision distinct. A source category or editable Git ref never grants runtime authority. Unknown topology restrictions remain labeled assumptions, never measured reachability; no application-dependency crawling.

## Feature PR map

Actual feature PR references are maintained in the umbrella issue. Do not create empty implementation PRs or branches merely to populate the checklist. Each PR uses a small number of atomic Conventional Commits; supporting `fix:`, `test:`, and `docs:` commits may stay on its `feat/...` branch.

| Unit / suggested branch | Components and interface | Prerequisites | Required checks and review boundary |
| --- | --- | --- | --- |
| M1-A — `feat/agent-registration` | Go bootstrap; versioned host/repository/stack registration; validated IDs, roots, paths and tracked-input declarations. | Existing milestone contracts and review of this delivery plan. | Schema errors, duplicate/unknown IDs, repository references, missing/overlapping roots, traversal and symlink escapes, option-like selectors, capability defaults. Pin Go version in this PR. No Git inspection or privileges. |
| M1-B — `feat/git-inspection` | Reusable Git inspection engine consumes validated registration; reports HEAD, dirty categories, optional `ls-remote` observation and ancestry facts. | A. | Real disposable repository fixtures; index/ref/worktree preservation; no implicit fetch, lazy object acquisition, configured helper execution or secret leakage. No CLI or persistence. |
| M1-C — `feat/status-cli` | Thin `status --repository/--stack [--remote]` interface over A/B; versioned envelope, errors, exit codes. | A, B. | Process-level stdout/stderr/exit tests, invalid selectors, stable JSON ordering and partial remote failures. State observations not yet implemented by D/E report `not_requested`; never fabricate absent or successful state. |
| M1-D — `feat/stack-state` | Per-stack last-applied/history readers and tracked-input identities/impact; wire read-only observations into C. | A, B, C. | Missing/corrupt records, distinct stacks sharing a repo, unrelated versus shared changes, add/delete/mode/input-set changes, incomplete history and uncertainty, no secret-derived identities. No applied-record writers. |
| M1-E — `feat/immutable-preparation` | Engine for full immutable revision acquisition, isolated materialization, manifest/completeness verification and preparation-only journal. | B, D (A transitively). | Moving refs, incomplete objects, SHA format, rejected symlink/submodule/LFS inputs, interruption/retry/concurrency, durable writes and offline no-network assertions. Preserve checkout and applied state. Review approval binding and staging/persistence proposal before coding. |
| M1-F — `feat/preparation-cli` | `history`, `validate`, `prepare` interfaces over existing capabilities; explicit `--offline` and sanitized envelopes. | C, D, E. | CLI integration tests for exact revision/approval, missing history, incomplete offline candidates, invalid/duplicate args, failures and no fallback. No `apply`, `confirm`, remote API or executor. |

Recommended review/merge order: A → B → C → D → E → F. D depends on C because its PR also integrates stack observations into status; E is an engine review and F its interface review. If a split later changes these dependencies, update this plan and the issue before opening the affected PRs.

Each PR links this plan and the umbrella issue, names prerequisites and target branch, describes security implications and limitations, includes relevant fixtures and affected contract updates, and reports checks actually run. Keep all PRs unmerged pending maintainer review; do not combine the whole milestone or create individual-commit micro-PRs.

## Security and cross-component coverage

- Registration → inspection: use fixed argv and registered IDs; reject escapes and unsafe configured Git helpers/transports. Preserve TLS/SSH verification and legitimate injected authentication. Bound subprocess time/output; sanitize credentials and raw subprocess errors.
- Status → state: prove byte-for-byte index/ref/worktree preservation and no writes to state/cache. Exercise unborn, clean, staged/unstaged/untracked, ignored synthetic secrets, ahead/behind/diverged/unrelated, shallow/partial/missing-object states, no remote, missing ref, unreachable remote and stale observations.
- State → preparation: compare exact revisions and complete declared inputs; use shared-stack/shared-file fixtures. Missing evidence stays unknown/uncertain. Known excluded secret paths are rejected before any content hashing; no secret bytes or derived identifiers in output.
- Preparation → CLI: pin approved full OIDs when remote refs move, verify complete trees/blobs, isolate candidate paths, reject unsafe symlinks and unresolved LFS/submodules. Exercise interruption, concurrent jobs and atomic journal recovery without promoting incomplete state to READY.
- Local-only status/history and explicit offline preparation must generate no network traffic. Failed online access never authorizes offline mode. Preparing a revision never advances an applied marker or claims health/deployment success.

Use temporary repositories/remotes and synthetic data. These tests need no Docker, privileged helpers or real homelab hosts. The final combined CLI suite runs after the feature PRs are integrated, in addition to each PR's focused tests.

## Review gates and unresolved mechanisms

Approved intent is not approval of every mechanism. Review A's registration schema/path rules; B's trusted Git configuration/transport allowlist and supported Git versions; C's final JSON/exit contract; D's record format and safe identity specification; E's local preparation approval binding, object store, materialization, locking and crash-safe journal. Plan 0002 supplies proposals, not implemented guarantees. Secret-safe arbitrary diff exposure remains out of scope until explicitly designed.

Later milestones still require D1 helper ABI/authorization/protected handoff review; D2 owned hooks, recovery/boot ordering, verifier and console policy; D3 enrollment/revocation and health deadlines; D4 conflicting policy/NAT semantics. Controller/UI frameworks and update-discovery packaging remain deferred. These do not reopen Go/Git/JSON or require a privileged mechanism before read-only work.

## Definition of done and recovery

- A–F have individually reviewed PRs merged by the maintainer into the existing milestone branch, with actual PR links and outcomes in the umbrella checklist.
- Relevant [acceptance criteria](../ACCEPTANCE.md#local-git-and-cli), each unit's fixtures and the combined CLI suite pass as an unprivileged user. Record actual commands, versions and remaining limitations.
- Architecture, public contracts, user documentation and examples match implemented behavior; no placeholder is reported as verified runtime functionality.
- Security review gates for implemented capabilities are resolved. No acquisition in status, secret output, active-tree overwrites, false applied state or unauthorized offline fallback.
- Final milestone diff to `main` is reviewed as a whole. A draft overview is not completion, and no merge occurs automatically.

Planning changes can be reverted through a normal reviewed commit. Future preparation failures may remove only tool-owned incomplete candidates; never reset/clean user checkouts or rewrite shared history. Existing unrelated branches and pending work remain preserved.

## Subsequent milestones — outline only

| Milestone | Separate review units and boundaries |
| --- | --- |
| M2 | Docker runtime inspection and safe identities; independently reviewed privileged helper authorization/protected handoff; deployment, verification and recovery engine/interfaces. |
| M3 | Linux network inventory; Git-backed persistent topology and policy evaluator; owned nftables execution; independently tested rollback and fresh-channel confirmation. |
| M4 | Authenticated worker API/enrollment; controller coordination; Git/provider adapters and central update-discovery integrations. |
| M5 | Fleet monitoring UI; Git configuration interface; network-policy visualization/editor over the same authorized APIs. |

No subsequent milestone implementation is authorized by this outline. Keep engines, interfaces and privileged executors independently reviewable.
