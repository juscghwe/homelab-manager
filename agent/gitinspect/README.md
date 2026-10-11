# Read-only Git inspection (M1-B)

`gitinspect` is a Linux Go library consumed by later agent interfaces. It reuses M1-A's validated registration, Go 1.27.2, the standard library and system Git **2.43 or newer**. There is no status CLI, persistence, acquisition, candidate, applied-state, Docker or privileged operation. Scope and checkpoints: [plan 0005](../../docs/plans/0005-git-inspection.md); dependencies: [plan 0003](../../docs/plans/0003-milestone-1-delivery.md).

## Public entry points and observations

```go
reg, err := registration.Load("") // existing M1-A JSON schema, local file only
if err != nil { return err }
engine, err := gitinspect.New(gitinspect.Options{})
if err != nil { return err }
result, err := engine.Inspect(ctx, reg, "homelab", gitinspect.Request{})
// Request{Remote: true} explicitly permits the registered ls-remote query.
// Consume independent observations even on error; never infer clean from zeros.
if err != nil { return err }
if result.DirtyAvailable && result.Dirty.IsDirty() {
    // Observed conflict, not a failed Git command or permission to overwrite.
}
```

Import both packages under `github.com/juscghwe/homelab-manager/agent/`. `ctx` is a non-nil `context.Context` supplied by the caller. `New(Options) (*Engine, error)` accepts trusted executable/authentication/limit settings. `Inspect(context.Context, *registration.Registration, registeredID, Request) (Result, error)` resolves the ID through registration accessors, checks `git_status`, then revalidates the filesystem. `Compare(context.Context, *registration.Registration, registeredID, leftOID, rightOID) (Comparison, error)` evaluates explicit full commit IDs using local objects only. Neither method accepts paths or arbitrary Git arguments. Engine and registration snapshots support concurrent reads.

| Result | Meaning |
| --- | --- |
| `Head` | `available` with full commit OID, `unborn`, or `unavailable`; branch name and detached state are distinct. A broken branch object is not unborn. |
| `Dirty`, `DirtyAvailable` | Per-path staged, unstaged, untracked and conflicted counts. One pathname can count in several categories. Ignored paths do not count. Rename detection is disabled: add/delete counts separately. Only `DirtyAvailable` establishes usable clean/dirty evidence. |
| `Remote` | `not_requested`, `not_configured`, `not_observed` (requested but local preflight failed), `unsupported`, `ref_missing`, `available`, or `unavailable`. Available OID is a fresh advertisement, never a downloaded/verified candidate. |
| `Relation`, `RelationReason` | Local HEAD relative to the fresh advertised OID: equal/ahead/behind/diverged/unknown/not_applicable. Remote absence is not applicable; unborn/unavailable observations stay unknown when comparison was requested. |
| `Evidence`, timestamps | Object format, shallow/partial metadata flags, local observation time and separate successful remote-observation time. These are observations, not an atomic repository snapshot or freshness cache. |

`Compare` accepts SHA-1 or SHA-256 full OIDs matching the repository. Equal requires both commit objects available but no history traversal. Non-equal comparisons require no shallow boundary, complete reachable object evidence (including trees/blobs), and successful two-way ancestry checks. Missing objects stay `unknown` with `MISSING_OBJECTS`; shallow histories stay `unknown` with `SHALLOW_HISTORY`. Available unrelated complete histories classify as diverged. Complete objects may be compared even if no current ref names them. Cached remote-tracking refs never substitute for `ls-remote`.

Errors are `*gitinspect.Error` with fixed `Code` and `Operation`. They never wrap OS/Git error text, filenames, credentials or configuration values. Inspect can return partial observations plus an error; errors before status leave `DirtyAvailable` false. Remote transport/authentication failures share `REMOTE_FAILURE`, because interpreting raw stderr would create a localization and secret-disclosure boundary. Timeout/cancellation/output-limit failures have distinct codes. Remote observation failure is recorded in `Remote.Failure`, preserving local facts without authorizing offline fallback. Later M1-C must map these facts into its own versioned envelope and exit codes.

## Data flow and repository preservation

1. `Inspect` selects an already validated repository by registered ID. M1-A's schema remains authoritative; there is no competing configuration model.
2. `openView` rechecks absolute root components and opens `os.Root`, which constrains the module's metadata reads. It reads bounded regular metadata with no-follow/nonblocking opens and checks opened-file identity.
3. Source configuration is parsed as inert data with `git config --file - --no-includes`. A private 0700 temporary metadata directory receives only a minimal allowlisted configuration, HEAD, refs, packed refs, index, shallow boundaries and inert `info/exclude`/`info/attributes`. No hooks, credential settings, include files, SSH commands, URL rewrites, replacement refs semantics or filter definitions become executable policy.
4. Git uses the private metadata/index and the existing worktree/object database. `screenPaths` reads index/tree **metadata**, screens all tracked paths using shared `registration.ContentExcluded`, refuses unsafe modes and filesystem components, and rejects assume-unchanged/skip-worktree index flags. Only then can status hash eligible working-tree files.
5. `parseStatus` converts NUL-delimited porcelain-v2 bytes into counts; no filenames, content, diffs or blob IDs reach the public result. HEAD and advertised commit OIDs remain distinct from content identities.
6. If explicitly requested, `observeRemote` runs `ls-remote` outside the checkout, using only the registered URL and branch. `compare` uses already available objects; it does not repair missing evidence.

The source index, refs, HEAD, configuration, worktree, ignored files and agent records are not written. Optional Git locks and lazy fetch are disabled. Source object alternates and symlink/special-file metadata are refused. Git receives an empty local protocol allowlist and no promisor configuration, an independent defense against lazy acquisition. Only private scratch is created/deleted; this is neither agent state nor candidate staging. Normal return/cancellation defers cleanup. A hard process termination can leave private scratch in the system temporary directory; no persistent cleanup service is implemented.

## Execution and authentication policy

`gitRunner.run` uses fixed argument vectors, explicit working directories, `exec.CommandContext`, a process group and group-kill cancellation. Captured stdout **and discarded stderr** share one synchronized byte limit. No raw stderr is returned or logged. Defaults: total inspection timeout 10 seconds, output 4 MiB per command, copied metadata 8 MiB, enumerated object/ref entries 50,000. Directory listings use batches of 128; path length/depth are bounded. Trusted options can raise limits only to defined maxima. These limits bound captured output and metadata, not Git's internal allocation or every filesystem syscall; this is not a resource or network sandbox.

Incoming executable/config/trace/askpass injection is removed. Local Git uses `/usr/bin:/bin`, unavailable HOME/XDG configuration, disabled system/global configuration, no replacement objects, no lazy fetching, no optional locks and no prompts. `GitPath` defaults to `/usr/bin/git`; a custom absolute path is trusted application configuration, never registration data.

Remote Git permits **HTTPS and SSH only**, refuses redirects, uses no repository configuration or URL rewrite, and cannot use local-file, ext or arbitrary transports. Authentication support is deliberate:

- HTTPS uses normal certificate verification, system trust and explicitly trusted process CA settings (`SSL_CERT_FILE`, `SSL_CERT_DIR`, `GIT_SSL_CAINFO`, `GIT_SSL_CAPATH`). It retains trusted HTTP(S)/ALL proxy settings and bypass lists. Remote HOME permits the administrator's standard `.netrc` behavior. Embedded HTTPS credentials were already rejected by M1-A.
- SSH uses `/usr/bin/ssh -F /dev/null -oBatchMode=yes -oStrictHostKeyChecking=yes`. Standard keys, trusted `SSH_AUTH_SOCK`, registered usernames/ports and known-host verification remain available. Unknown host keys fail; inspection never auto-accepts them. User/system SSH configuration, ProxyCommand, custom identity selection and incoming `GIT_SSH_COMMAND` are not loaded.
- An administrator can explicitly provide `Options.TrustedCredentialHelper` as an absolute executable plus argv. Git's helper ABI invokes a shell; `shellQuote` quotes **each trusted argument**, never interpolating repository/remote data. Helper get/store behavior belongs to that selected trusted program and may affect its own credential store. This opt-in is not supplied by registration or repository config. Future CLI authentication selection needs its own reviewed trusted configuration policy.
- Repository/global Git credential helpers, askpass programs, TLS-verification bypasses, configured Git executable paths and arbitrary transport commands are unsupported. They are not silently imported. Environments relying on them must select a reviewed trusted helper or supported SSH-agent/key authentication; the library does not downgrade host/certificate verification to make them work.

TLS/SSH live provider authentication is not an end-to-end claim of these fixtures. Remote states/invocation are tested at a mockable runner boundary; local-only network isolation uses a loopback HTTP promisor and executable marker traps. Credential-helper quoting also runs through real Git using synthetic credentials.

## Supported layouts and remaining risks

Ordinary non-bare SHA-1/SHA-256 checkouts with an in-directory `.git` are supported, including unborn/detached/shallow/partial repositories when required local evidence is available. Linked worktrees/gitfiles, bare repositories, object alternates, config includes, filter drivers (including LFS), unknown extensions/reftable, sparse/split indexes, tracked symlinks/submodules and hidden index flags are refused or unavailable. Missing/corrupt/inaccessible objects/metadata are not repaired. Larger repositories may hit explicit limits.

Only validated structural config (object format, filemode, ignorecase, precomposeunicode, autocrlf/eol) and repository-local inert attributes/excludes participate. User/system ignore/attributes files are ignored, so counts may differ from the user's Git configuration. Git index stat-cache behavior still applies; status is an observation, not a forensic proof of every byte. Metadata/ref files are copied sequentially: concurrent edits can produce unavailable/inconsistent observations rather than one atomic snapshot.

Static no-symlink checks and `os.Root` protect metadata opened by Go. External Git still resolves worktree/object paths after those checks. A concurrently hostile process with write access could replace filesystem entries during that interval; this implementation does **not** provide syscall-level confinement against that adversary. Operate on administrator-approved checkouts without hostile concurrent writers; stronger confinement requires separate design/review, not a claim that temporary metadata alone eliminates that risk. Filesystem reads can update atime; preservation tests compare contents/names, not every filesystem timestamp.

Known excluded paths (`.env`, `.env.*`, `.git`, `.ssh`, key extensions) are screened before worktree hashing. Ignored/untracked files are enumerated as metadata only; they are not hashed by status. Suspicious-path warnings are neither proof of secret contents nor permission to read them. Administrators must ensure tracked eligible files are non-secret and contain no persistent volume data; arbitrary secrets cannot be identified by filename. No installer consent persistence or secret-content scanner is implemented. Never log registration/configuration, internal command argv, helper outputs or raw subprocess errors.

## Code Review Guide

Read in this order and trace the referenced tests yourself:

1. **Public contract:** [model.go](model.go), `New`, `Inspect` in [inspect_linux.go](inspect_linux.go), and `Compare` in [ancestry_linux.go](ancestry_linux.go). Structs group explicit facts; zero counts do not establish availability. Understand which failures retain local evidence and why there are no prepared/applied fields.
2. **Execution boundary:** [runner_linux.go](runner_linux.go), especially `gitEnvironment`, `gitRunner.run`, `captureWriter.Write`, `normalizedOptions` and `shellQuote`. `context.Context` carries cancellation/deadlines; `exec.CommandContext` starts a child without interpolating a shell command, while process-group cancellation kills descendants. A mutex serializes stdout/stderr accounting because Go copies those streams concurrently. The private `runner` interface allows tests to substitute remote responses without weakening production transport policy.
3. **Filesystem boundary:** [view_linux.go](view_linux.go), `openView`, `settings`, `read`, `walk` and `scanObjects`, then `screenPaths` in [inspect_linux.go](inspect_linux.go). Identify every source read, copied file and scratch write. `defer` ties cleanup to function return; it cannot guarantee cleanup after SIGKILL. Review the exact gap between Go's root-constrained checks and external Git's later path resolution.
4. **Local observations:** [parse.go](parse.go), `parsePaths`, `checkIndexFlags` and `parseStatus`, together with `Inspect` orchestration. Byte slices preserve NUL-delimited filenames including non-UTF-8 bytes. Count aggregation avoids emitting those bytes or reading/displaying file contents. Inspect malformed records and unavailable counts, not just the happy path.
5. **Remote and ancestry:** [remote.go](remote.go), `observeRemote`/`parseRemote`, then `compare` in [ancestry_linux.go](ancestry_linux.go). Follow the registered branch to a freshly advertised OID and prove that no fetch occurs. Trace the completeness gate before negative ancestry tests; `errors.As` classifies typed safe errors, never wraps raw stderr.
6. **Evidence:** [inspect_test.go](inspect_test.go), [runner_test.go](runner_test.go), [remote_test.go](remote_test.go), [ancestry_test.go](ancestry_test.go), and [security_test.go](security_test.go). Fixtures use real disposable repositories; remote replies are controlled at the runner boundary. `snapshot` compares every source file's bytes before/after, including index, refs and synthetic state. Marker executables and the loopback server attempt violations rather than merely asserting intended safety.

Questions for independent review:

- Could a repository-defined executable reach `gitEnvironment` or the temporary configuration? Trace `settings`, `TestMaliciousConfigurationIsNotExecuted` and `TestUnsafeLayoutsAndSourcePaths`.
- Can local inspection contact a promisor or lazily acquire missing objects? Trace `view.command` and `TestNoLocalHTTPOrLazyObjectAcquisition`; examine why environment restrictions are not advertised as a network sandbox.
- Does the remote comparison ever use cached refs as fresh evidence? Follow `observeRemote` through `TestRemoteUsesFreshAdvertisedOIDNotCachedRef`.
- Which code prevents excluded input hashing, and which test proves the original index and agent records remain unchanged? Inspect `screenPaths`, `TestExcludedInputsStopBeforeWorktreeHashing`, `TestIndexFlagsCannotConcealDirtyFiles` and `TestConcurrentInspectionPreservesSourceAndState`.
- What happens when a shallow boundary or missing parent/tree/blob makes negative ancestry inconclusive? Trace `compare`, `TestShallowAndMissingObjectsStayUnknown` and `TestRealShallowCloneAndMissingBlob`.

## Validation and file map

Run the shared `agent/scripts/check.sh` from the repository root (see [agent guide](../README.md)). It requires Go 1.27.2, Git 2.43+, and a compatible development C compiler for race tests. It checks gofmt, vet, units, CGO-enabled race tests and CGO-disabled build. Ubuntu CI runs the same suite. No tests use personal repositories, real homelab infrastructure, Docker, nftables or host registration.

| Files | Responsibility |
| --- | --- |
| `model.go` | Public observations, availability, comparison, options and sanitized error codes. |
| `inspect_linux.go` | Registration selection, local orchestration and pre-hashing path/index checks. |
| `runner_linux.go` | Trusted execution, bounded output/cancellation, filtered environment and auth argv quoting. |
| `view_linux.go` | Isolated inert metadata, path safety, bounded enumeration and scratch ownership. |
| `parse.go` | Machine-readable path/status parsing and safe branch validation. |
| `remote.go` | Explicit registered advertisement query and strict response parsing. |
| `ancestry_linux.go` | Local-only explicit revision comparison and completeness checks. |
| `*_test.go` | Fixtures, adversarial attempts, error/limit tests and preservation evidence. |
| `../registration/paths.go` | Additive `ContentExcluded` accessor for the existing exclusion policy; no schema redesign. |

Maintainer decisions for this PR: accept/revise the narrow supported layouts and config semantics, explicit trusted-authentication allowlist, and concurrent-writer threat boundary. These are concrete mechanism reviews under the approved architecture, not completed privileged authorization or final CLI contracts.
