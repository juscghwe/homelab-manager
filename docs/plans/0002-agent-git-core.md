# Milestone 1 — unprivileged Git core

Status: proposed implementation plan under approved D5/D6/D7 intent; documentation only. Go, system Git, versioned JSON and optional systemd were approved by the maintainer. No application code, dependencies or executable tests are added by this plan.

## First slice and boundaries

The [M1 delivery plan](0003-milestone-1-delivery.md) separates registration, inspection engine, status CLI, stack state, preparation engine and preparation CLI into M1-A through M1-F. Each PR targets the existing milestone integration branch (with documented temporary stacks only for unmerged dependencies). Review this planning bundle before starting implementation; prerequisite contracts already exist on the milestone and need not be recovered into `main`.

Implement local/read-only Git status first using a small Go binary and system Git subprocesses. Prefer the standard library for JSON, argument parsing and process execution. No embedded Git library, Docker connection, privileged helper, controller or daemon is needed. Pin a maintained Go release in the implementation PR. Systemd only becomes relevant to optional background operation, not CLI invocation.

Read [agent instructions](../../agent/AGENTS.md), [SECURITY](../SECURITY.md), [DATA_MODEL](../DATA_MODEL.md) and the [approved decisions](0001-architecture-readiness.md#maintainer-decisions-and-remaining-approval-gates). Proposed public spellings below can be refined in implementation review without reopening the approved product architecture. Privileged apply and remote authentication remain separate reviewed work.

## CLI proposal

Use `homelab-agent` as a provisional binary name. Commands use registered IDs, not arbitrary operation paths. `--config` is a local CLI option for a registration file, never a remotely supplied path or privileged-helper override. Emit one JSON envelope to stdout; diagnostics go to stderr without raw Git stderr, credentials, secret-bearing URLs or content.

| Command | Scope and behavior |
| --- | --- |
| `status --repository ID [--remote]` | First slice: local checkout observations and scoped stack records. Default is local-only; `--remote` explicitly runs `ls-remote` for the registered ref, never fetch. |
| `status --stack ID [--remote]` | Same first-slice logic, scoped to one registered stack. Exactly one repository/stack selector is required. |
| `history --stack ID` | Following slice: read local deployment records, no network or runtime mutation. Missing history is explicit, not fabricated success. |
| `validate --stack ID --revision OID` | Following slice: inspect available approved candidate inputs and registration; no privileged action. |
| `prepare --stack ID --revision OID --approval ID [--offline]` | Following slice: explicit acquisition/staging under normal approval. Without `--offline`, may acquire missing objects; with it, require a complete locally available exact commit and no network. Never silently change OID to a moving tip. |
| `apply --stack ID --candidate ID --approval ID [--offline]` | Executor milestone only: revalidate and invoke installed authorized operations. Offline requires a deliberate local-administrator invocation and locally valid approval. |
| `confirm --transaction ID --approval ID` | Executor milestone only: scoped confirmation, including fresh-channel evidence where required; cannot waive remote firewall recovery. |

Accept full Git object IDs in mutation commands and retain the repository's object format; do not assume SHA-1 length universally. Reject option-like IDs, duplicate/unknown arguments and invalid registered selectors. A future remote API uses the same result model and local authorization semantics; these CLI options do not expose an arbitrary remote shell.

## JSON status and errors

Propose a version-1 envelope: `schema_version`, `operation`, `outcome` (`ok`, `partial`, `error`), `data` and `errors`. Each error has a stable `code`, sanitized `message`, `scope` and `retryable` flag. Successful status may report dirty/diverged state; it is an observation, not a failed command or an apply authorization.

Observed identities use a common fact shape: `value`, `availability` (`available`, `not_requested`, `unavailable`, `not_configured`, `unsupported`), `observed_at` (UTC or null), and `stale`. An unavailable current remote result may retain a cached value only with its original timestamp and `stale: true`; never mark a failed fresh query available. Read-only status may consume a pre-existing cache but must not write one. A skipped fresh query labels retained cached remote evidence stale rather than inferring freshness. Output ordering is deterministic for lists keyed by ID; consumers must ignore unknown additive fields within version 1.

Proposed stack-scoped `data` contains `host`, `repository`, `stack`, `git` (`local_head`, `remote_head`, `staged_revision`, `applied_revision`, `dirty`, `relation`), `config`, and `runtime`. `dirty` has staged/unstaged/untracked booleans and counts, no file contents. `relation` is `equal`, `ahead`, `behind`, `diverged`, `unknown`, or `not_applicable`. A clean `equal` relation gives the semantic `synced` label; neither says anything about deployment. Unborn HEAD uses unavailable local value and an explicit `unborn` diagnostic; absence is not a corrupt repository. Repository-scoped status contains repository-level `git` observations and a `stacks` array of scoped records holding each stack's staged/applied facts, config and runtime. It never puts one inferred deployment revision on the repository itself.

`config` holds `tracked_identity`, the comparison revision pair and `impact` (`changed`, `unchanged`, `uncertain`, `not_requested`). `runtime` keeps `image_ids`, safe optional `configuration_identity`, execution/state/health and confirmation facts separate. Status in the first slice reports runtime as `not_requested`; it does not call Docker or claim running state. Missing local applied/candidate records are `not_configured`, not equal to HEAD.

Example partial envelope (illustrative subset of `data`, not a full executable JSON Schema):

```json
{
  "schema_version": 1,
  "operation": "status",
  "outcome": "partial",
  "data": {
    "host": "media01",
    "repository": "homelab",
    "git": {
      "remote_head": {"value": null, "availability": "unavailable", "observed_at": null, "stale": true},
      "relation": "unknown"
    }
  },
  "errors": [{"code": "REMOTE_UNAVAILABLE", "message": "Registered remote could not be queried", "scope": "repository:homelab", "retryable": true}]
}
```

| Exit | Meaning |
| --- | --- |
| 0 | Requested observations completed, including explicitly reported dirty/diverged/unborn or unconfigured optional state. |
| 1 | Internal/unexpected failure; sanitized `INTERNAL_ERROR`, never raw subprocess output. |
| 2 | Invalid CLI input or registration (`INVALID_ARGUMENT`, `INVALID_REGISTRATION`). |
| 3 | Requested observation partially unavailable, for example `REMOTE_UNAVAILABLE`; useful local data remains in the envelope. |
| 4 | Operation denied or precondition failed (`UNAUTHORIZED`, `CONFLICT`, `MISSING_OBJECT`, `UNSAFE_PATH`); mainly later mutation commands. |

Unknown ancestry alone is a valid reported result, not a reason to fetch during status. A broken/inaccessible registered checkout is an error, unlike a valid unborn branch. Missing system Git reports an explicit dependency error with exit 1. Plan tests must assert envelopes and exit codes together.

## Registration proposal

Propose host-local JSON in `/etc/homelab-manager/agent.json` for the runtime registration, parsed with Go's standard library. The [YAML example](../../examples/host-registration.yaml) is a readable illustration of the same proposed fields, not a second runtime configuration source. Topology YAML belongs to a separate Git contract and need not add a YAML dependency to the first agent slice.

Fields: `schema_version: 1`, `host`, a repository map (`root`, optional `url`, `branch`), stacks (`id`, `repository`, `path`, `compose_project`, `tracked_inputs`), and explicit capabilities. Resolve each stack's branch through its repository registration; stacks tracking different branches need separate registered checkouts. Roots are absolute approved checkout roots; stack/input paths are normalized repository-relative exact files or directories. Directory inputs expand only to tracked entries. Reject escapes, unsafe symlinks and overlapping/ambiguous registrations. Shared inputs must be explicitly listed for every affected stack. An optional URL supports local Git; `--remote` on a no-remote registration reports `not_configured` without inventing a remote.

Capabilities default off except unprivileged Git status. Authorization-sensitive registration is administrator-controlled; an agent-readable registration cannot itself expand protected helper policy. No credentials, secret contents or environment values appear here; any execution-only secret binding references are separately protected and never part of status identities. UI/API callers select registered IDs only. First-slice tests may provide local fixture registration via `--config` because no privileged operation exists there.

## Read-only Git mechanics

Use fixed subprocess argument vectors, optional locks disabled, noninteractive trusted transport and bounded timeouts. Inspect HEAD, index/worktree categories and existing objects without updating the index, refs or checkout. Use NUL-delimited machine-readable Git output and handle unusual filenames without parsing human output. Never run arbitrary repository-selected diff/textconv, hooks, filters, credential helpers or transports; define an explicit trusted transport/config allowlist without disabling TLS/SSH host verification or platform-injected authentication. Acquisition must not trigger lazy partial-clone fetching during a supposedly local command; disable implicit fetching or return unsupported/unavailable for incomplete repositories.

`--remote` invokes only `ls-remote` on the registered remote/ref. Missing remote ref differs from transport failure. Determine ancestry only from complete relevant objects already present; shallow boundaries, missing objects and incomplete history yield `unknown`, not false divergence. Two-way ancestry checks distinguish ahead/behind/diverged only with enough history. Status never deepens/fetches. Dirty checks cover the whole registered repository, even when configuration impact is stack-specific.

Tests verify exact before/after index bytes, refs, worktree contents and absence of writes to agent state. Local-only mode must produce no network requests. Preflight tracked path metadata against protected secret exclusions before invoking commands that may read worktree contents: if a known secret path is tracked, report an unsafe registration and unavailable dirty/identity evidence rather than letting Git hash that file for status. Do not inspect file contents to discover secrets or render arbitrary diffs in the first slice. Undeclared secrets inside otherwise approved tracked configuration remain a trust-boundary risk; no automatic secret detection guarantee is claimed.

## Journal and immutable candidate proposal

Following slices: use a host-local agent state directory, default `/var/lib/homelab-manager/agent/`, with versioned `transactions/<id>.json`, `stacks/<id>/last-successful.json`, candidate manifests and timestamped remote observations. Reads must tolerate absent state and distinguish corrupt records from absent history. Transaction snapshots include identity, exact revision, operation, approval reference, transition, deadlines and separate execution/health/confirmation outcomes; no secret contents or raw command output. Serialize updates, write/fsync a temporary record, rename and fsync the directory. Update the last-successful pointer only after durable confirmed completion. On restart, reconcile pointer/transaction consistency without promoting incomplete operations. Root-owned firewall recovery state is separate from the agent-owned journal.

Propose a separate agent-owned bare object store per repository and private candidate directories. Explicit preparation imports approved objects using system Git, verifies full commit/tree/blob availability, and materializes tracked content with Git plumbing and controlled path handling. No checkout hooks, smudge filters, arbitrary export rules or local ignored data. Record the exact commit, object format, registration/input-set identity and content manifest. Fail clearly for required unresolved LFS objects/submodules rather than silently fetching/executing them; supported acquisition can be designed separately. Validate candidate completeness before READY.

Compared with detached worktrees, a separate object store avoids changing active-checkout worktree/ref metadata and makes retention/locking local to the agent, at the cost of duplicate objects and careful materialization. First scope rejects symlinks in executable deployment inputs rather than following host paths. Staging is logically pinned, not a security guarantee against a compromised worker: the [D1 helper handoff](0001-architecture-readiness.md#d1-helper-interface-and-privilege-proposal) copies and verifies into protected staging before privileged use. No applied record is written by preparation.

Offline preparation uses only already available complete Git objects. Never reuse an incomplete directory as READY after interruption. Preserve active edits/index/unpublished commits through success and every failure; clean up only the tool's own incomplete candidate state. Acquisition/staging is a later slice, not hidden inside status.

## Tracked identities and change detection

D6 default: each stack explicitly declares all non-secret tracked configuration paths plus shared tracked files. Compare the last successful applied commit with the exact candidate commit, not simply the latest two repository commits. Enumerate relevant entries through system Git tree metadata without reading blob contents for reporting; include additions, deletions, modes, directory expansion and input-registration changes. A stack with no applied record, incomplete objects, unsupported references or unknown external/shared inputs has `impact: uncertain` with a reason. An unrelated repository commit leaves a stack unchanged when its complete declared inputs and registration are unchanged. Repository-wide dirty/conflict checks still protect apply.

Propose `tracked-v1`: SHA-256 over a canonical length-delimited sequence of repository object format, stack path/project and declared input specification, plus sorted `(path bytes, mode, Git blob OID)` for the eligible entries. This identifies committed tracked inputs, not effective runtime configuration. Do not report hashes/OIDs of excluded secrets. Explicit declarations are a trust contract: exclude `.env`, secret files/bindings and persistent data; known/suspected secret-bearing tracked inputs yield unavailable identity rather than a hash. Merely hashing a Git OID does not sanitize secret content. Do not scan secret contents to decide whether they are safe. Parsing non-secret Compose includes/env-file references for validation is distinct from crawling application configuration to infer service dependencies; undeclared input dependencies must make impact uncertain before selective apply.

Local HEAD, advertised remote, candidate, applied commit and tracked identity are distinct. Running image IDs come only from optional narrowly scoped Docker inspection, never inferred from tags or a Git change. Image tags may move without changing tracked identity; an explicitly authorized pull can therefore change runtime image identity. Publish an optional Compose/runtime fingerprint only when its input provenance is demonstrably free of secret contents; otherwise report unsupported/unavailable. No raw `docker inspect` environment or rendered/interpolated Compose configuration enters central status. Default Git status never invokes Docker.

## Explicit offline deployment

D7 is approved intent, not permission for an unattended shortcut. Proposed `prepare/apply --offline` is local-administrator-only, with full available revision, current locally verifiable resource/operation approval, complete inputs, normal preflight, protected helpers and required health/confirmation. No network attempt or automatic transition from failed online mode is allowed. Record `origin: local`, `offline: true`, exact candidate and the stale/not-configured remote evidence in the journal; this does not claim approval from a current remote tip.

Unknown remote freshness is permitted specifically by the local offline approval; it does not waive other missing evidence. Dirty/ahead/diverged conflicts need a separate explicit documented resolution that preserves edits/commits. Existing containers and history remain usable when Git/controller are offline. Emergency firewall recovery never depends on object acquisition or remote approval renewal. Remote-triggered jobs with insufficient approval/revision evidence are rejected rather than downgraded to local offline mode. Actual deployment waits for executor implementation; first-slice offline support is inspection only.

## Validation, risks and recovery

Implement the [Milestone 1 fixture matrix](../PLANS.md#1--agent-only-git-core), exercising the real CLI as an unprivileged process. Add output/exit cases above, no-network local status, byte-preserved index/refs, malicious helpers/arguments, partial/shallow clones, no-remote registration, stale cache, distinct per-stack applied records and unrelated/shared tracked changes. Identity tests must use synthetic secrets and prove excluded paths never contribute contents or content-derived IDs to output. Later staging tests cover interruption, complete objects, moving refs and preservation of active trees. No privilege, Docker or live infrastructure is needed for these fixtures.

Main risks are implicit Git side effects, untrusted execution helpers, false ancestry, secret-derived reporting and confusing prepared with applied. Contain them through fixed arguments, explicit availability and fixture assertions. Recovery for status is simply stopping the command: it writes no state. Later preparation failures remove only incomplete tool-owned candidate state; never reset or clean a user's checkout. A development rollback is a normal revert, not rewriting history.

Remaining explicit maintainer review concerns privileged helper trust, remote authentication, health/confirmation policy and firewall recovery from plan 0001. They do not block the read-only Git-status slice. Document actual tests when code is implemented; none has run for this plan.
