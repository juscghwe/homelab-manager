# Architecture readiness audit — Task 001

Status: documentation audit completed; proposed contracts require PR review. No application or privileged operation implemented or tested.

## Purpose and scope

Prepare Milestone 1 using the existing product intent. Review covered the root and all four module instruction files, the seven contract documents, README, both examples, the initial plan and PR template. This branch addresses one concern: documentation and development governance. It introduces no dependencies, framework selection, CI, deployment or infrastructure changes.

Severity means implementation risk: **high** can compromise authority, data or recovery; **medium** can create incompatible behavior; **low** affects contribution clarity. Findings below describe gaps in the scaffold, not observed application vulnerabilities.

## Findings and resolutions

| ID | Severity | Finding | Documentation resolution |
| --- | --- | --- | --- |
| A01 | High | Controller cache wording obscures the requirement to persist manual models. | DESIGN separates agent-owned deployment/configuration state from controller-owned manual topology and annotations. Git-backed records retain Git authority. Storage implementation remains D4. |
| A02 | High | PREPARE compares a moving remote ref while jobs approve an exact commit. Missing ancestry could be mistaken for divergence. | DATA_MODEL and PROTOCOL distinguish advertised observations, verified ancestry and pinned candidates; no silent substitution or reset. |
| A03 | High | CONFIRM assumes a controller; generic local confirmation could bypass remote firewall recovery. | PROTOCOL gives standalone equivalent local authorization, preserves fresh-channel remote firewall confirmation, and distinguishes rollback obligations. D2/D3 gate execution. |
| A04 | High | Idempotent retries conflict with rejecting every replay; stack serialization misses shared repository races. | PROTOCOL distinguishes matching retries from changed-payload replay and requires coordination of shared mutable repository state. Persistence mechanism remains open. |
| A05 | High | Git diff and effective Compose fingerprints could expose host-local secrets; inspection of untrusted Git can invoke helpers. | SECURITY constrains paths, content output and external Git execution; effective-config fingerprint design remains D6. |
| A06 | Medium | `synced` conflates checkout, remote, applied and runtime state; applied revision lacks explicit stack scope. | DATA_MODEL makes these independent facts and labels, with unavailable/unknown states and per-stack deployment records. Staging never records deployment success. |
| A07 | Medium | `preview_change` appears as connectivity verdict; policy origin may differ from device origin. | DATA_MODEL separates result context from access verdict and keeps per-record provenance. Examples now illustrate all four device sources and defined annotation endpoints. |
| A08 | Medium | Milestone 3 model ownership overlaps Milestone 4 controller; CLI behavior and fixture coverage lack an actionable first slice. | PLANS assigns local model/evaluator work to 3 and controller persistence/API to 4; defines Milestone 1 inputs, outputs, boundaries and fixture matrix. |
| A09 | Medium | Examples lack consistent version markers; Docker deployment is enabled in a generic sample. | Both examples explicitly remain illustrative, carry a version marker, and default privileged capabilities off. No finalized schema is claimed. |
| A10 | Low | Branch, commit and review discipline is unspecified. | AGENTS and PR template establish scoped branches, Conventional Commits, actual validation results and human review gates. |

## Invariant traceability

| Required invariant | Contract | Observable acceptance coverage |
| --- | --- | --- |
| 1. Git primary; hosting optional | [DESIGN](../DESIGN.md#source-of-truth-model) | [Local Git](../ACCEPTANCE.md#local-git-and-cli), [portability](../ACCEPTANCE.md#ui-and-ci-portability) |
| 2. Agent owns checkout and applied state | [DATA_MODEL](../DATA_MODEL.md#agent-status-minimum-meaningful-fields), DESIGN | Local Git: staging/inspection preserve applied records; UI/CI: caches cannot become desired configuration |
| 3. Equivalent standalone/headless/GUI semantics | [PROTOCOL](../PROTOCOL.md#deployment-sequence) | UI/CI: identical target, preflight and authorization outcomes |
| 4. Unprivileged Git; separate Docker/firewall capabilities | [SECURITY](../SECURITY.md#privileged-execution) | Local Git, Docker operations and Firewall/network negative cases |
| 5. Preserve dirty/divergent state | [DATA_MODEL](../DATA_MODEL.md#git-state-labels), PROTOCOL | Local Git fixture matrix and preservation assertions |
| 6. Four independent source categories | [DATA_MODEL](../DATA_MODEL.md#core-entities) | Firewall/network: provenance and permissions tested independently |
| 7. Unknown restrictions imply assumed permission | [DESIGN](../DESIGN.md#network-view) | Firewall/network: unknown router, VLAN and physical-path cases |
| 8. Manual baseline persists; preview is proposed delta | [PROTOCOL](../PROTOCOL.md#topology-preview) | Firewall/network: restart persistence and preview isolation |
| 9. No application dependency crawling | DESIGN network view | Firewall/network: annotations do not create allow/dependency edges |
| 10. Firewall preflight, separate approval and local rollback | [SECURITY](../SECURITY.md#fail-safe-behavior), PROTOCOL | Firewall/network: failed preflight, stale confirmation, daemon restart and controller outage |

## Open decisions, ranked by impact

All D-items require maintainer approval recorded in a linked plan or ADR before the dependent implementation. Recommendations are proposals, not permission. Unaffected documentation and fixture design may proceed.

| Rank / ID | Decision and competing options | Recommended default and trade-off | Gate |
| --- | --- | --- | --- |
| 1 / D1 | Privileged boundary: narrowly scoped helper versus separate privileged service; authentication/authorization and enrollment lifecycle. | Keep separate fixed-operation Docker and nftables interfaces, with host-local policy enforcement; prefer mTLS for remote identity. A service improves lifecycle control but adds protocol/recovery work. Neither option removes Compose's effective root authority. | Before privileged executor implementation (2/3); enrollment and credential lifecycle before 4. |
| 2 / D2 | Firewall owned tables/hooks, durable timer/journal backend and fresh management-channel verification; service manager timer versus independent watchdog. | Independent host-local recovery that survives agent restart; bind confirmation to transaction/revision and fresh authenticated channel. Backend and restart/reboot guarantees need disposable-environment evidence. | Before 3 executor; never substitute an existing connection or controller-only timer. |
| 3 / D3 | Standalone and headless confirmation policy, Compose health/deadlines, journal durability and recovery from ambiguous outcomes. | One local state machine with profile-specific authorization/confirmation adapters; preserve last successful marker plus explicit incomplete outcome. Simpler local confirmation cannot satisfy remote firewall reachability by itself. | Before 2 deployment; detailed wire encoding before 4. |
| 4 / D4 | Manual topology persistence and evaluator: local serialized model versus database; graph traversal and policy ordering/NAT scope. | Define provenance-preserving local fixtures first, then controller-owned persistent manual baseline. Database supports concurrent edits but adds migrations. Neither storage option replaces Git-backed configuration. | Model/evaluator contract before 3; controller storage before 4. |
| 5 / D5 | Agent language, CLI serialization/exit codes, registration schema, journal format and isolated staging mechanism. Go versus another small host-native implementation; detached staging checkout versus exported immutable tree. | Go with system Git is the existing proposal; structured versioned output and separate staging preserve inspectability. Detached checkouts retain Git metadata; exports need explicit content/path validation. No language or framework is approved by this audit. | First Milestone 1 implementation plan must record approval before adding dependencies or application code. |
| 6 / D6 | Stack change scope and safe configuration fingerprint: explicit tracked input set versus broader repository invalidation; sanitized hash design versus omitting hash when unsafe. | Conservatively mark potentially affected stacks when inputs are unknown; omit unsafe fingerprints. Exact change scope must handle shared tracked inputs without reading host-local secret contents. | Before selective deployment/Compose inspection in 2; do not promise exact impact detection in 1. |
| 7 / D7 | Missing-remote operation: full local-only deployment profile versus future extension. | Implement offline local inspection in 1; keep new no-remote deployment policy unresolved. Cached observations never prove current remote state. | Before any local-only apply support. |

Controller framework (FastAPI/Python is only a proposal) and UI framework remain deferred to their module plans. A stack references one repository in the current model; combining multiple repositories into one stack would require an explicit contract change, not an inference from multiple repositories per host.

## Implementation handoff

Next: create `0002-agent-git-core.md` and resolve D5's first-slice language/output decisions with the maintainer. Use the [Milestone 1 contract](../PLANS.md#1--agent-only-git-core) for unprivileged status and Git fixtures first. Inspect local and advertised revisions without fetching during status; test unknown ancestry and offline results. Add separate explicit acquisition/staging only in the following slice. Docker, firewall, controller, UI and actual applied-marker writes are outside that first slice.

## Validation and recovery

For this docs-only change: inspect all contract edits against the invariant table; check local Markdown links and heading anchors, YAML parsing and example references; run `git diff --check`; inspect the file list before explicitly staging documentation. No executable documentation checker, application suite or build existed at audit time. These checks validate documents, not runtime behavior. Record actual outcomes in the PR.

Recovery is a normal revert of this documentation commit after review; no runtime migration or host mutation occurs. Security-sensitive contract changes require explicit human review; do not auto-merge.
