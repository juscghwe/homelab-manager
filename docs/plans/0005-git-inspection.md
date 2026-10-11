# M1-B — read-only Git inspection

Base: `milestone/1-agent-git-core` at `6088737a55b8ee893bebc4ec443faa9a620bac93`; prerequisite PR #8 is merged. Tracking: issue #6. Scope follows plans [0002](0002-agent-git-core.md), [0003](0003-milestone-1-delivery.md) and [registration](0004-agent-registration.md).

## Design and review boundaries

Use the existing Go module and system Git. A private temporary metadata view copies only bounded HEAD/ref/index/shallow/ignore metadata, selects a minimal configuration, and points Git at the existing object database and working tree. Never copy repository executable config into that view, and never fetch or write the source repository. Reject unsafe metadata paths, alternates and unsupported layouts. The view is ephemeral scratch, not agent state or candidate preparation.

The public engine consumes validated registration and registered IDs. Its typed results distinguish HEAD/unborn/detached, dirty counts, requested remote observations and conservative ancestry. It has no CLI envelope, deployment/preparation fields or persistence. A controlled runner uses fixed argv, filtered environments, deadlines, process-group cancellation and bounded stdout/stderr; raw diagnostics never escape it.

Remote queries use registered HTTPS/SSH and branch only. Authentication uses system TLS trust, strict SSH host verification and trusted administrator-selected authentication inputs; repository helpers/configuration are never authoritative. The exact supported policy and restrictions must be documented and tested before completion. Metadata checks are a snapshot: concurrent hostile filesystem replacement is a limitation, not an isolation guarantee.

## Checkpoint roadmap

| Chunk | Purpose / files | Dependencies and checkpoint | Tests / completion |
| --- | --- | --- | --- |
| 1 | Contract and runner: `agent/gitinspect/model.go`, runner and focused tests | M1-A; commit/push tested typed contract, bounded execution and draft PR | Cancellation, limits, sanitized failures and environment policy |
| 2 | Isolated metadata view and local observations/parsing | Chunk 1; commit/push real fixture coverage | Clean/dirty/ignored, branch/detached/unborn, path screening and byte preservation |
| 3 | Optional remote observation and conservative ancestry | Chunk 2; commit/push fixture and mock-transport coverage | Not requested/absent/missing/failure/timeout; all relations and incomplete evidence |
| 4 | Security and preservation regression coverage | Chunks 1–3; commit/push any resulting corrections | Malicious helpers/config, no local network/lazy fetch, secret exclusions, source/state preservation |
| 5 | Engine guide, human Code Review Guide, contracts and full checks | Chunks 1–4; finalize PR and tracking issue | Formatting/vet/units/race/CGO-disabled build/whitespace locally and CI, document actual limits |

Chunk 1 checkpoint: typed observations, controlled runner, filtered environment, combined stdout/stderr bounds and process-group cancellation are implemented. Formatting, vet, unit tests, race tests and CGO-disabled build pass locally. At that checkpoint the engine operations remained outstanding; chunks 2–4 below complete them. Recover from Git/PR progress if interrupted; never restart completed chunks or manually squash development commits. Revert reviewed feature commits for development rollback; inspection only deletes its own ephemeral scratch.

Chunk 2 checkpoint: local HEAD/branch/detached/unborn facts and NUL-delimited dirty counts are implemented using an ephemeral metadata view. Before status hashes worktree files, index and HEAD tree paths are screened. `DirtyAvailable` distinguishes successful clean observations from unavailable counts. Real fixtures verify unusual filenames, ignored synthetic secrets, staged/unborn states, malicious config/environment refusal or isolation, and byte-level source preservation. All shared checks pass locally. Additive `registration.ContentExcluded` exposes the existing policy so registration and inspection cannot drift; no registration schema change. Next: remote observations and ancestry.

Chunk 3 checkpoint: optional registered `ls-remote` runs outside the checkout with the trusted transport environment; it distinguishes not requested/unconfigured/missing/available/failure/timeout/unsupported states. `Engine.Compare` and inspection compare full local OIDs only after verifying object evidence; shallow/missing closure stays unknown, with typed reasons. Fresh remote OIDs are tested against deliberately stale remote-tracking refs. All shared checks pass locally. Next: chunk 4, adversarial layouts/metadata bounds, network isolation and preservation regressions.

Chunk 4 checkpoint: batched metadata enumeration and stricter status parsing are in place. Real fixtures exercise metadata/worktree symlink escapes, excluded tracked inputs, malicious programs, corrupt indexes, SHA-256 and unusual filenames, missing trees, loopback promisor isolation, concurrent inspection and source/state byte preservation. Index assume-unchanged/skip-worktree flags are refused rather than reporting false clean state; inert repository-local attributes are retained for line-ending semantics. All shared checks pass locally with GCC installed, including CGO-enabled race testing. Additional fixtures now prove excluded inputs stop before status/remote execution, inaccessible metadata stays sanitized, real shallow clones stay conservative, and missing blobs cannot establish divergence. Next: documentation, human review guide and final hosted CI.

## Review handoff

Chunks 1–5 are implemented on `feat/git-inspection` in PR #10, with public interfaces, supported auth/layouts, data flow, security limitations and a human Code Review Guide in the [engine guide](../../agent/gitinspect/README.md). Local shared formatting/vet/unit/race/CGO-disabled-build checks pass using Go 1.27.2, Git 2.47.3 and GCC 14.2. Documentation links and whitespace are checked separately. Exact hosted CI results are recorded in PR #10 rather than predicted here. Main and the milestone integration branch remain unchanged by this PR; M1-C has not started.

Next development chunk is maintainer review, particularly trusted authentication selection, supported repository semantics and the concurrently hostile filesystem limitation. Do not start M1-C until its prerequisite review/integration and separate task authorization. No CLI, agent state, candidate or applied-state contract is claimed by this engine.
