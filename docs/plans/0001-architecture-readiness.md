# Architecture readiness audit — Task 001

Status: audit updated with explicit maintainer decisions D1–D7. Approved intent is recorded below; concrete mechanisms remain proposals where marked. Documentation-only: no application or privileged operation implemented or tested.

## Purpose and scope

Prepare Milestone 1 using the existing product intent. Review covered the root and all four module instruction files, the seven contract documents, README, both examples, the initial plan and PR template. This branch addresses one concern: documentation and development governance. It records the approved Go/system Git/JSON/systemd direction without installing dependencies or implementing code, CI, deployment or infrastructure changes.

Severity means implementation risk: **high** can compromise authority, data or recovery; **medium** can create incompatible behavior; **low** affects contribution clarity. Findings below describe gaps in the scaffold, not observed application vulnerabilities.

## Findings and resolutions

| ID | Severity | Finding | Documentation resolution |
| --- | --- | --- | --- |
| A01 | High | Controller cache wording obscures the requirement to persist manual models. | D4 resolves manual topology authority to a user-selected Git repository. The backend loads committed records; any graph/cache is derived. Agent-owned deployment records remain local. |
| A02 | High | PREPARE compares a moving remote ref while jobs approve an exact commit. Missing ancestry could be mistaken for divergence. | DATA_MODEL and PROTOCOL distinguish advertised observations, verified ancestry and pinned candidates; no silent substitution or reset. |
| A03 | High | CONFIRM assumes a controller; generic local confirmation could bypass remote firewall recovery. | PROTOCOL gives standalone equivalent local authorization, preserves fresh-channel remote firewall confirmation, and distinguishes rollback obligations. D2/D3 gate execution. |
| A04 | High | Idempotent retries conflict with rejecting every replay; stack serialization misses shared repository races. | PROTOCOL distinguishes matching retries from changed-payload replay and requires coordination of shared mutable repository state. Persistence mechanism remains open. |
| A05 | High | Git diff and effective Compose fingerprints could expose host-local secrets; inspection of untrusted Git can invoke helpers. | SECURITY constrains paths, content output and external Git execution; D6 approves explicit tracked inputs and excludes secret-derived reporting; the concrete identity proposal is in plan 0002. |
| A06 | Medium | `synced` conflates checkout, remote, applied and runtime state; applied revision lacks explicit stack scope. | DATA_MODEL makes these independent facts and labels, with unavailable/unknown states and per-stack deployment records. Staging never records deployment success. |
| A07 | Medium | `preview_change` appears as connectivity verdict; policy origin may differ from device origin. | DATA_MODEL separates result context from access verdict and keeps per-record provenance. Examples now illustrate all four device sources and defined annotation endpoints. |
| A08 | Medium | Milestone 3 model ownership overlaps Milestone 4 controller; CLI behavior and fixture coverage lack an actionable first slice. | PLANS assigns local Git-backed model/evaluator work to 3 and central aggregation/API to 4; plan 0002 defines the Git-core handoff. |
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

## Maintainer decisions and remaining approval gates

The maintainer explicitly approved the following intent. These decisions supersede the earlier open alternatives, including controller-owned topology configuration and deferring offline local deployment. Do not ask again to select Go, Git-backed topology or narrow helpers. Proposed command/schema spellings are concrete defaults to review in their implementation plans, not new product features.

| ID | Approved intent | Remaining mechanism / timing |
| --- | --- | --- |
| D1 | Unprivileged worker; installer-provisioned, root-owned narrow operation helpers; no direct Docker socket or broad sudo; Docker and firewall independently authorized. Compose content remains a host-administrative trust boundary. | Review helper ABI, sudoers policy, candidate handoff and authorization verification before Milestones 2/3. See proposal below. |
| D2 | Durable local rollback independent of agent/controller; fresh authenticated connection and transaction-bound confirmation for remote firewall updates; only owned rules affected. | Review systemd/boot recovery, table/hook scope, verifier and console path before 3; verify restart/reboot/crash behavior in disposable VMs. |
| D3 | Lightweight standalone worker with CLI/authenticated CI interface; optional central fleet backend; optional GUI on the same APIs. Update discovery belongs centrally. Worker owns journal, health outcomes and success confirmation. | Review remote caller enrollment before the small remote interface, and concrete health/confirmation deadlines before 2. Backend/UI framework and optional Renovate packaging are deferred to 4/5. |
| D4 | User-selected Git repository is authoritative for manually maintained topology; YAML preferred, remote or local Git supported. Observations/derived caches remain separate. | Minimal schema/reconciliation proposed below; evaluate ordering/NAT support before 3. No authoritative configuration database is needed for V1. |
| D5 | Go agent, system Git, versioned JSON CLI/API, optional systemd background operation. | [Plan 0002](0002-agent-git-core.md) proposes commands, errors, registration, journal and staging; no repeat language approval gate. First slice is unprivileged read-only status. |
| D6 | Explicit per-stack tracked inputs including shared files; conservative uncertainty; separate Git/configuration/image identities; no secret-content reporting or hashing. | Plan 0002 proposes metadata-based identities and change detection. Safe optional runtime identity exposure is reviewed before Docker inspection. |
| D7 | Offline inspection/history mandatory; existing workloads continue. Explicit local-administrator deployment of complete available revisions is supported with normal local approval/validation. No automatic offline fallback. | Plan 0002 proposes explicit offline commands; local authorization/conflict handling must be reviewed before apply, not before status. |

Only security-sensitive mechanisms and unresolved product policies need further explicit maintainer approval: D1 helper authorization/handoff, D2 recovery/confirmation, D3 remote enrollment and health deadlines, and the handling of conflicting/unsupported firewall semantics in D4. Proposed data structures and Git-status mechanics can be refined through ordinary implementation-plan review within approved scope. No privileged behavior is authorized by a proposal alone.

## D1 helper interface and privilege proposal

**Proposed default, not installed:** separate root-owned executables per operation under `/usr/libexec/homelab-manager/`: `compose-pull`, `compose-up`, `compose-recreate`, `compose-stop`, `compose-status`, and separate `nft-prepare`, `nft-apply`, `nft-confirm`, `nft-rollback`, `nft-status`. Keep each small and share only reviewed validation routines. No generic shell dispatcher, raw `sudo docker`, arbitrary `nft` arguments or agent Docker-group membership.

Invoke each executable with no command-line arguments; use bounded versioned JSON on stdin. A mutation request has `schema_version`, `resource_id`, `transaction_id`, `approved_revision`, `candidate_id` and `approval_id`; confirmation also references fresh-channel evidence. Status accepts only the registered resource ID. Identifiers are validated tokens, never paths. The helper resolves every path, Compose project, allowed operation and approval record from protected registration. No caller-selected Docker flags, environment overrides or shell text. Pull alone records image preparation, not successful deployment; only the full verified apply transaction may advance the applied marker.

Installer proposal: root owns the helpers and parent directories, `/etc/homelab-manager/privileged.json`, approval trust configuration and firewall recovery state; the worker cannot write them. Generate separate sudoers rules only for enabled executables, explicitly requiring empty argv (sudoers `""` argument constraint), noninteractive execution and no SETENV. Validate generated rules with `visudo -c`. Do not authorize directories, wildcard command paths, `systemctl`, Docker or nftables binaries directly. Deployment command mapping is fixed: `up --pull never -d`, recreate adds `--force-recreate`, pull is a separate authorized operation, and explicit stop has its own rule. Exact installed Compose compatibility/flag order is tested before choosing supported versions.

The helper must verify the caller, operation, resource, approved revision/content manifest, expiry and transaction against an independently verifiable approval record; stdin claims alone are not authority. An administrator-approved protected record is the proposed local path; remote signed approval and key enrollment need separate review. The unprivileged Git stage prepares a candidate. The helper securely copies approved non-secret tracked content into protected staging without following symlinks or accepting path escapes, verifies the copy against the approved manifest, and executes only that protected copy. This closes the agent-writable-tree race; Git inspection/fetch never runs as root. Host-local secret bindings are fixed by protected registration and used only for execution, never copied into Git candidates or status. Exact trust/manifest verification must pass adversarial tests before implementation is accepted.

Separate executables make sudo rules easy to inspect at the cost of more entry points; a single privileged long-running service would add persistent privilege and is not the chosen direction. Helper restrictions minimize authority but cannot make arbitrary trusted Compose content unprivileged.

## D2 independent firewall recovery proposal

**Proposed default:** systemd-managed one-shot recovery plus a fixed periodic timer and boot recovery unit, installed by setup. Use protected per-transaction records under `/var/lib/homelab-manager/firewall/`; never depend solely on a transient in-memory timer. The worker cannot edit recovery records or disarm timers through general service-manager permissions.

1. Unprivileged acquisition selects the exact approved ruleset. The helper validates syntax, policy scope and owned tables/hooks before mutation; reject unsupported interactions rather than flush unrelated rules.
2. Under a resource lock, save the previous owned state, approved candidate identity, transaction, deadline, boot ID and confirmation state. Atomically replace/fsync the record and its parent directory; fail before apply if durable recording or recovery service readiness fails.
3. Arm recovery before applying. The timer periodically scans pending records independently of the worker. Apply only owned state as an nftables batch; journal ambiguous/partial outcomes without claiming success.
4. Require a new authenticated connection to the registered management endpoint after apply, bound to host, transaction, revision and a one-use challenge. Proposed verifier: the initiating CI/controller opens that connection; an equivalent approved management peer is needed if that route cannot work. Outbound heartbeat alone is insufficient. The helper validates scoped confirmation evidence rather than trusting an agent-supplied boolean.
5. Confirmation and timeout use the same lock and durable state transition: expired/rolled-back transactions cannot be confirmed; confirmed transactions cannot later be rolled back by an old timer. Agent restart does not alter the timer or extend the deadline.
6. On timeout, restore only the captured owned state and record the result. A failed restore stays visibly pending/failed and is retried by the recovery service; new mutations are denied until resolved.
7. On reboot or uncertain clock/boot continuity, the boot unit restores any unconfirmed transaction conservatively instead of extending its deadline. Use monotonic deadlines within a boot; pending state from a prior boot triggers recovery regardless of wall-clock changes. Order recovery after the configured base firewall loader and before the worker accepts new mutations; test distro-specific startup ordering so another loader cannot silently reapply the unconfirmed candidate.

A fixed timer is simpler and restart-resistant but has bounded scheduling delay; propose a 120-second confirmation window and one-second scan interval for review, configurable within protected policy. Persistent records plus boot recovery handle power loss better than agent timers, at the cost of careful fsync/order testing. A crash after apply but before journaling is treated as unconfirmed and recovered. Confirmed configuration boot persistence must be integrated with the host firewall loader; no universal boot guarantee is claimed.

A local console transaction may use a separately authorized confirmation path. It cannot change a remote transaction's origin or disable its required fresh-channel verification. Review owned nftables tables/hooks, the trusted management verifier, console policy and host firewall-loader integration before implementation. Disposable VM tests must cover agent kill/restart, reboot at each transition, disk-write failure, deadline races, stale evidence and coexistence with Docker rules.

## D3 success and remote-interface proposal

Keep the CLI usable without a daemon. Optional systemd operation enables the small authenticated worker API; CI and the central backend are callers of the same operations. Propose mTLS with host-local trusted caller identities scoped to resource/operation, including direct CI; certificate enrollment/revocation and delegated approval verification require review before implementing this interface. No controller connection is necessary to inspect status/history or complete a properly authorized standalone transaction.

Propose runtime verification of expected services plus configured Compose health checks, with a protected default 120-second deadline and per-stack override. A missing health check yields `not_configured`, never `passed`; verify observable running/exited state according to the explicitly registered service expectation (including one-shot jobs). Lack of user-facing Docker status capability must not let deployment skip required internal verification: the apply helper returns a minimal sanitized verification result. Health failure prevents confirmation and triggers the configured best-effort recovery; successful execution without application checks may be confirmed but remains explicitly application-health-unverified. Final deadlines and one-shot/health policy need review before the Docker executor.

Renovate/equivalent discovery is optional, managed alongside the central backend; it proposes updates/notifications and does not grant deployment authority. Choose packaging and controller framework in Milestone 4. The worker never bundles the update scanner.

## D4 topology schema and reconciliation proposal

**Proposed default:** one `topology.yaml` document at a configured path/ref in the selected Git repository. Use `schema_version: 1`, stable IDs, and collections `devices`, `groups`, `annotations`, `policies`, plus an explicit `sources` registry for independent Git inputs. Device fields are `id`, `kind`, `source`; optional `observation_ref` binds a modeled device to a namespaced observation ID, never a fuzzy name/IP match. Groups list device IDs; annotations use IDs, endpoints and a label; policies use IDs, owner, provenance category, permanent baseline membership, protocol/endpoints/ports/action and explicit order when known. The [example](../../examples/topology-model.yaml) is illustrative, not a validated application schema.

Git storage does not change semantic source: a manually authored rule on a discovered router remains `manual`. Resolve independent source references against the registry and attach repository/ref/commit/path provenance. Effective `can_edit_git`/`can_apply_runtime` is derived from actual authorization/agent registration; YAML cannot grant permissions. Agent status, measurements, timestamps and runtime authority never become authored configuration merely by appearing in an evaluated graph.

Load an exact committed baseline and validate IDs, references and supported rule semantics. Overlay fresh observations as separate attributed records. Unknown/stale observations remain visible as uncertainty. Conflicting facts remain separately attributed; do not silently overwrite manual/Git records or invent rule precedence. Proposed default for ambiguous policy order is an uncertainty diagnostic with every known restriction preserved, never a measured reachability claim. Detailed evaluator handling of conflicts/NAT needs review before Milestone 3.

GUI/headless edits submit an expected base commit through a Git-provider adapter (including local Git), then create an explicit commit/proposal. A changed base or dirty local checkout causes a conflict, never a reset or silent last-writer overwrite. Independent Git sources are read at explicit revisions without gaining apply authority. Preview applies an in-memory proposed delta to a specified baseline; it never commits or changes observations/baseline. Discarding cache and reloading Git plus fresh observations reconstructs the graph, preserving manual nodes and rules.

One YAML document minimizes schema/storage machinery; larger multi-file layouts can follow measured need. Local Git needs explicit commit/conflict handling but no hosted service. No authoritative topology database is required.

## Implementation handoff

Use [0002-agent-git-core](0002-agent-git-core.md) for the first unprivileged Git-status slice. Go/system Git/versioned JSON and optional systemd are approved; no further language choice is needed. The plan proposes stable status/errors, registration, journal layout, immutable preparation, tracked identities and explicit offline behavior. Implement status first with its fixture matrix; defer acquisition/staging to the following slice and privileged/remote-interface implementation to reviewed plans. This PR remains documentation-only.

## Validation and recovery

For this docs-only change: inspect all contract edits against the invariant table; check local Markdown links and heading anchors, YAML parsing and example references; run `git diff --check`; inspect the file list before explicitly staging documentation. No executable documentation checker, application suite or build existed at audit time. These checks validate documents, not runtime behavior. Record actual outcomes in the PR.

Recovery is a normal revert of this documentation commit after review; no runtime migration or host mutation occurs. Security-sensitive contract changes require explicit human review; do not auto-merge.
