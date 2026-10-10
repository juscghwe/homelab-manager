# Protocol contracts (V1 target)

The specifics of HTTP routes, serialization and language-specific DTOs are unresolved implementation decisions; preserve these invariants. [Open decisions](plans/0001-architecture-readiness.md#open-decisions-ranked-by-impact) identify approval gates. This document does not select a framework.

## Agent -> controller status
Report stable host ID, scope, timestamp, git refs/state, last applied deployment transaction, optional sanitized Docker metrics, and discovered topology/policy facts. mTLS identifies the agent; an untrusted JSON field cannot override identity.

## Controller -> agent jobs
A job MUST bind: host, registered stack or owned firewall policy identifier, operation, exact approved revision, authorization identity, unique transaction ID, expiry/deadline, and explicit approval outcome. Job fields may never supply arbitrary sudo commands or filesystem paths.

## Deployment sequence
1. `PREPARE` — authenticate and authorize the request, inspect registered scope and local conflicts, observe the advertised remote ref, explicitly acquire/validate exact approved objects and sufficient ancestry, then stage without overwriting the active tree; validate. A moving branch does not change the approved commit. If a freshness policy requires renewed approval, stop and report it; never silently deploy the new tip. Missing objects or unresolved required ancestry prevent readiness.
2. `READY` — return transaction ID, exact staged commit, preflight result, affected stack/policies, last applied revision.
3. `APPLY` — explicitly authorized transition; arm local rollback mechanism when required, run only enabled operations.
4. `VERIFY` — local health and status check; report runtime identities and a new outbound status message.
5. `CONFIRM` — the authorized controller or standalone local caller confirms the matching transaction/revision after required verification; the agent records successful completion. Remote firewall changes additionally require a fresh authenticated management connection bound to that transaction/revision before disarming host-local rollback. An ordinary status message or surviving connection is insufficient.
6. `ROLLBACK_ATTEMPT` — on failure or timeout, host performs best-effort local revert; journal every transition. A Compose rollback does not undo persistent-data migrations.

Standalone CLI supplies equivalent local authorization and confirmation; it never requires a controller to use Git status or apply a locally authorized operation. Local confirmation cannot waive the fresh-channel requirement for a remote firewall change. Exact health checks, confirmation deadlines and standalone firewall recovery design require approval before executor implementation (D2/D3).

Only confirmed successful completion advances the relevant host/stack applied marker (or the owned policy's record). A failed or partially rolled-back operation records its actual outcome and leaves the last successful revision identifiable; that old marker does not assert the current runtime was restored. Local inspection, candidate staging and Git-only commits never advance an applied marker.

Rollback is operation-specific: firewall mutation requires durable host-local rollback before apply; Compose failure triggers a best-effort restoration attempt, with no guarantee for persistent data. Read-only inspection/preview does not arm a rollback timer. Loss of the controller cannot prevent already-armed local firewall recovery.

## Retry rules
- Authenticate and authorize each message. A retry matching the original transaction, identity, target, operation and approved revision returns the recorded state/outcome without rerunning a completed mutation. Reusing its ID with altered content or identity is rejected.
- Expired/replayed requests cannot authorize a new transition; unknown IDs are accepted only as newly authorized PREPARE requests, never as APPLY/CONFIRM. An authorized status read may report a past outcome without replaying it.
- Mutations to the same stack/policy are serialized. Stacks sharing a repository also coordinate changes to shared refs/staging metadata; per-stack serialization alone does not protect shared Git state. Revalidate preflight at the mutation boundary and reject intervening conflicts.
- A network interruption never implies success; agents retain local journal and recovery timer.

## Topology preview
`GET current model` may combine agent facts, independent Git sources, discovered unmanaged nodes and persistent manual modeling. `POST preview delta` MUST be read-only and return provenance, affected access paths and uncertainty/assumption labels. Persist/apply is a separate, explicit operation. For a Git-only device, a commit is never mistaken for verified runtime application.

These operation names describe semantics, not approved HTTP routes. Preview results retain their connectivity verdict and are marked as proposed context; discarding a preview does not remove manual baseline entries. Manual baseline persistence is controller-owned in the headless profile; cached agent/Git facts retain their source authority and timestamps. See [DESIGN](DESIGN.md#source-of-truth-model).

## Versioning
Version API schemas from V1 onward. Persist `schema_version` with serialized documents. No breaking protocol change without an explicit migration and interoperability tests.
