# M1-A — agent registration implementation

Implements only the registration unit of [plan 0003](0003-milestone-1-delivery.md), under the approved intent in [plan 0002](0002-agent-git-core.md). Integration target: `milestone/1-agent-git-core`; tracking: issue #6.

## Interfaces and scope

Use the standard-library-only Go module `github.com/juscghwe/homelab-manager/agent`, pinned to Go 1.27.2. Package `registration` separates bounded strict JSON decoding, deterministic validation/normalization, and loading a local file (default `/etc/homelab-manager/agent.json`). Expose typed sanitized errors and a validated registration for later M1-B consumers. No Git execution, CLI, daemon, networking, state journal or privileged operation.

## Concrete schema decisions

Keep the proposed repository map and stack list, allowing explicitly empty collections for hosts without Git repositories or Compose stacks. IDs and Compose projects are bounded lowercase ASCII tokens. Optional remote URL/branch declarations are validated syntactically; only HTTPS, SSH URLs and SSH scp-style remotes are admitted. No password, HTTPS userinfo, URL query/fragment or external Git helper syntax. Remote contact and actual Git-repository checks belong to M1-B.

JSON fields are case-sensitive; reject duplicates, unknown fields, nulls, trailing documents and excessive size/nesting. An omitted capabilities object or field defaults to Git status enabled and other known capabilities disabled. Explicit booleans are retained as declarations, never grants of privilege; unknown capability names/types are rejected. None of the declared runtime operations is implemented here.

Roots must be existing absolute directories without symlink components; reject equal/nested roots and detectable same-directory aliases. Stack and tracked-input paths are normalized repository-relative paths, reject any `..`, absolute/backslash forms, symlinks, missing paths and special files. Stack locations must be directories and cannot overlap in one repository; Compose projects must be unique host-wide. Tracked inputs may be shared across stacks, but duplicates or ancestor/descendant declarations within one stack are ambiguous and rejected.

Recognized secret/control names (`.env`, key extensions, `.git`, `.ssh`) remain hard exclusions. Ambiguous names such as `secrets`, `data` and `volumes` produce deterministic structured warnings, not rejection. Future installer confirmation must be explicit and narrowly bound to the location/configuration, including noninteractive overrides; consent cannot authorize actual secret/data reads or bypass path boundaries. Arbitrary secrets cannot be discovered from filenames. Do not enumerate directories or read any stack/input contents. Directory declarations do not establish a complete safe tracked-input set; M1-D must validate Git-tracked descendants and exclusions. Filesystem validation is a snapshot, not authorization or protection against later replacement; later operations must revalidate with constrained filesystem access.

## Security, tests and recovery

Loader reads only the explicitly selected regular JSON registration file, with bounded input and no symlink components. It rejects hard-excluded secret/control paths before opening and reports suspicious directory-name warnings, and returns no path, URL, decoder text or OS-error detail supplied by configuration. Repository path checks use metadata and root-constrained access. Registration does not verify ownership/approval policy or independently authorize future helpers.

Use temporary directories and synthetic fixtures for valid registrations, sharing, schema ambiguity, references, unsafe paths, roots, aliases, symlinks, special files, capabilities, sanitized errors and deterministic map ordering. Run `go test ./...`, `go vet ./...`, `go test -race ./...` when supported and documentation/whitespace checks. No actual host configuration or infrastructure is used.

Recovery is a reviewed revert of this library; registration loading performs no writes and requires no runtime migration. Public schema/path/remote restrictions are concrete M1-A review points, not changes to D1–D7 intent.

## Durable execution checkpoints

| Chunk | Purpose and files | Checkpoint and dependencies | Verification / completion |
| --- | --- | --- | --- |
| 1 — Recover registration subsystem | `agent/go.mod`, `agent/registration/`, agent guide, JSON example and affected plans | Preserve the existing M1-A implementation in a local recovery commit on `feat/agent-registration`, then push and open a draft PR against the milestone. Depends on reviewed plans 0002/0003. | Recovered implementation passes `go test ./...`, `go vet ./...` and whitespace checks. Race build is blocked here because gcc is absent. Remote checkpoint publication must be confirmed separately. |
| 2 — Acceptance and review | Registration edge cases, test results and PR description | Review the recovered implementation against M1-A, commit any cohesive corrections, push and mark the draft ready only when complete. Depends on chunk 1's remote checkpoint. | Record actual checks and limitations; update issue #6 and assign the PR to `juscghwe`. M1-B stays out of scope. |
| 3 — Independent housekeeping handoff | Root agent instructions in a separate `chore/agent-housekeeping` branch from canonical `main` | Only after M1-A is published, prepare a separate housekeeping PR. It must not contain this implementation or milestone architecture. | Review-only sync to the milestone follows maintainer merge of housekeeping; preserve milestone-specific instructions. |

Recovery verification: the existing subsystem was found uncommitted on the correct feature branch based on `c074e7867dbe56db4aca4760b2f58c21adf9dc07`. Unit tests and vet passed again. `go test -race ./...` with cgo enabled could not build because `gcc` is unavailable. The prior parser fuzz check passed 21,985 executions, and the prior documentation check passed 72 local links / 29 heading anchors. Those last two checks are recorded results from the implementation session, not new recovery-session executions.

Publication confirmed: recovery commit `dbcc184eee64df55efd495d70966212acbe0be59` is pushed to canonical `feat/agent-registration`; [PR #8](https://github.com/juscghwe/homelab-manager/pull/8) targets `milestone/1-agent-git-core` and is assigned to `juscghwe`. The temporary shell network block was resolved through the app's permission controls. GitHub CLI identity was verified as `juscghwe-codex` before publication.

Acceptance review confirmed the M1-A boundary, test coverage and documented restrictions; no additional executable functionality was needed during recovery. Unit tests, vet and whitespace checks pass. Race validation remains unrun because the required C compiler is absent; this is a review limitation, not a passing result. The PR is ready for maintainer review after this checkpoint is pushed. No automatic merge or M1-B implementation is authorized by this completion.

The recovery handoff above is historical. The maintainer subsequently merged #9 and requested the review revisions below for #8; housekeeping policy is now synchronized through milestone commit `e013b25` and its merge into M1-A.


## Review revisions — PR #8

Maintainer review requested development tooling/CI, empty host registration and structured suspicious-path warnings. Approved housekeeping PR #9 is now integrated into the milestone and this feature branch, preserving milestone-specific instructions. The earlier assignee edit is retained by the same rule in the approved policy.

| Chunk | Purpose / affected files | Dependencies and checkpoint | Tests / completion |
| --- | --- | --- | --- |
| 1 — Development checks | `agent/` check entry point, GitHub Actions and agent guide | Existing registration tests; use a compatible local compiler if available and install build tools in Ubuntu CI, then commit and push reproducible formatting, vet, unit and race checks. | Formatting, vet, units and CGO-disabled build pass locally; race tests run in CI when no compatible local compiler exists. GCC is a development dependency only. |
| 2 — Empty host scopes | Registration schema/validation, fixtures and agent guide | Chunk 1; explicitly allow empty repositories and stacks without dummy Compose/Git declarations, then commit and push. | Firewall-only and repository-only hosts validate; missing/malformed collections and unknown stack references still fail; capability defaults remain safe. |
| 3 — Suspicious path warnings | Registration warning model, validation/loader, fixtures and affected contracts | Chunk 2; return deterministic sanitized warnings for suspicious directory names without weakening hard path/secret-content exclusions, then commit and push. | Structured warnings, metadata-only checks, path-boundary failures and secret-content protections tested. Document future interactive confirmation and narrowly scoped noninteractive override; no installer or CLI implemented. |

Chunk 1 checkpoint: shared formatting/vet/unit/race/CGO-disabled-build script and pinned GitHub Actions workflow implemented. Formatting, vet and unit checks pass locally; local race validation is unavailable because no compatible compiler is installed; the maintainer will install GCC separately. CI explicitly installs build-essential. The PR stays draft until all review changes and validation complete. Next item: explicit empty host registration; verify hosted CI before final handoff.

Chunk 2 checkpoint: explicit empty repository/stack collections and independent capability declarations are supported; omitted/null/malformed collections and unknown stack references remain errors. Unit tests and vet pass. Next item: structured suspicious-path warnings; hosted CI must also validate the final revision.

Chunk 3 checkpoint: `Registration.Warnings()` now reports deterministic sanitized `SUSPICIOUS_PATH` concerns for ambiguous names; hard exclusions and filesystem boundaries still fail. New fixtures cover warning order/copies, unreadable synthetic inputs, loader locations, empty hosts and hard-error preservation. No installer, consent persistence, Git inspection or privileged operation was introduced. The maintainer explicitly authorized relying on Ubuntu CI race checks while the development host lacks GCC/Clang/cc; no further sudo or toolchain installation will be attempted. Next item: full local validation, final hosted CI result and re-review handoff.
