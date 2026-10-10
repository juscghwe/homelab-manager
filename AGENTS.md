# Repository instructions for coding agents

## Mission
Build a modular Git-native Linux/Docker deployment and network-access manager. Before coding, read `docs/INTENT.md`, `docs/DESIGN.md`, and `docs/SECURITY.md`. For work in a module, read its nested `AGENTS.md` explicitly.

## Document authority
- `docs/INTENT.md`: product scope and non-goals.
- `docs/DESIGN.md`: architecture and allowed responsibilities.
- `docs/SECURITY.md`: authorization, privilege boundaries and recovery requirements.
- `docs/DATA_MODEL.md`: terminology, states and source/authority distinctions.
- `docs/PROTOCOL.md`: agent-controller and deployment semantics.
- `docs/ACCEPTANCE.md`: observable V1 completion criteria.
- `docs/PLANS.md`: milestone order and planning rules.
- Module `AGENTS.md`: local design and testing constraints.

Do not silently change an architectural invariant. When a requirement is ambiguous, preserve the stricter security boundary and document the decision/assumption for review. Update impacted documentation and tests in the same change as code.

## Global invariants
1. Remote Git is the default configuration delivery method. Git hosting brands and CI/CD platforms are adapters, never hard dependencies. No parallel authoritative configuration store in the controller, including manually maintained topology: its baseline lives in a user-selected Git repository.
2. Agent-only, headless, and full GUI profiles share the same deployment engine and revision semantics. UI must never be required for a CLI or CI workflow.
3. Credentials, `.env` files, secrets, persistent volume data, and private host keys stay host-local; do not ingest or expose their contents via the UI/API/logs.
4. Git read/diff operations run unprivileged. Docker administration and firewall application need explicit, separated privileged capabilities through installer-provisioned, root-owned helpers that the agent cannot modify. The agent has no Docker socket access or general sudo permission. No unrestricted root shell, remote Docker socket, or arbitrary `sudo` argument passthrough.
5. A user-approved Compose file may itself grant effective root-equivalent host privileges via rootful Docker. Do not describe wrappers as eliminating that risk.
6. Topology access evaluation defaults to **assumed allowed** in the absence of known restrictions, but must label assumptions and must not claim tested reachability. Do not infer application dependencies by inspecting Caddy or other app configs.
7. Four source categories are first-class: managed agent; independent Git source (without agent); discovered unmanaged device; manually modeled device. Source, edit permission and apply authority are separate fields.
8. Unmanaged simulated/modelled devices and their declared policies are persistent baseline state. Preview mode is ONLY for proposed changes. No mutation happens from running a preview.
9. Firewall modifications never flush unrelated/Docker-owned rules. Remote firewall changes must have durable timed rollback independent of agent/controller processes and a fresh authenticated connection plus transaction-bound confirmation.
10. Treat dirty local Git trees and divergence as conflicts; never silently reset or overwrite them. Docker deployments are non-transactional with respect to application data and migrations.

## Working rules
- Keep changes small and isolated to the relevant module. Avoid speculative frameworks and service-specific application discovery.
- For significant functionality, write/update a plan under `docs/plans/` before implementation; include risks, tests, and rollback.
- Prefer machine-readable data contracts and deterministic fixture-based tests.
- For every PR: state the change, threat-model impact, changed public interfaces, tests run, and known limitations.
- Privileged integration tests must run only in disposable test environments; never point tests at real homelab infrastructure.
- Never claim a feature, rollback, reachability result, or end-to-end test is working without verifying it.

## Git contribution rules

### Branches
- `juscghwe/homelab-manager` is the sole canonical target; forks are not authoritative. Preserve the existing milestone branch and use its actual current tip as the development baseline.
- Never implement changes directly on `main`, `master`, or a milestone integration branch. Create topic branches from the applicable `milestone/<number>-<kebab-case-goal>` branch; reuse an existing branch for the same concern rather than creating parallel replacements. An environment-supplied generic branch must be renamed before publishing.
- Use the existing isolated cloud checkout; do not create an additional Git worktree unless the user explicitly requests it. This contribution rule does not select the application's future candidate-staging implementation.
- Topic branch types are `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `perf`, `style`, and `chore`. Use `<type>/<kebab-case-subject>` for topic branches and `<type>: <subject>` for commit subjects and PR titles. Reserve `milestone/<number>-<kebab-case-goal>` for integration branches and `milestone: <subject>` for milestone integration PRs and their merge commits. `main` is the stable integration destination, not a topic branch.
- Keep one integration branch per active milestone, created from `main`; do not create speculative branches for future milestones. The current integration branch is `milestone/1-agent-git-core`. For example, create `feat/agent-git-status` from it and target it with the feature PR. See [milestone integration](docs/PLANS.md#branch-and-pr-integration).
- The branch and PR type describe the overall concern; each commit's type describes its own change. A `feat/...` branch may contain related `fix:`, `test:`, and `docs:` commits. Do not create a new branch for every supporting fix. Avoid generic names such as `work`, `temp`, or `patch`, and replacement suffixes such as `-recreated`.
- Keep one coherent concern per branch and PR. Do not combine unrelated modules or unrelated formatting/refactoring.
- Inspect status before editing; preserve pre-existing changes and unpublished commits. Never reset or overwrite them to obtain a clean tree.

### Commits
- Before remote mutations, verify `gh api /user --jq .login` returns `juscghwe-codex`. New commits use `juscghwe-codex <340593538+juscghwe-codex@users.noreply.github.com>` for both author and committer; never fall back to personal-account credentials.
- Use the shared typed subject convention above (including `milestone` for milestone completion), for example `docs: clarify deployment state ownership`. Keep commits atomic and understandable.
- Review the diff and stage only relevant, explicit paths; avoid indiscriminate staging.
- Do not amend or rewrite existing shared history. Never force-push to protected branches.
- Do not commit credentials, secrets, generated temporary data or local environment settings.

### Pull requests and integration
- Topic PRs target their documented milestone branch, not `main`. A genuine unmerged prerequisite may temporarily use a clearly documented stacked PR against its predecessor branch. After the predecessor merges, retarget/reconcile the child to the milestone without force-pushing shared history, verify its intended-only diff and rerun affected checks. Use `<type>: <subject>` titles matching the overall concern. Only the complete, reviewed milestone integrates into `main`, with a `milestone: <subject>` title.
- Maintain the umbrella issue and [M1 delivery plan](docs/plans/0003-milestone-1-delivery.md) as the scope/dependency/acceptance map. The existing milestone branch is the cumulative bundle; reuse any aggregate draft PR and never merge it before milestone completion. Keep individual feature PRs as the review units.
- Assign new PRs and issues to the maintainer `juscghwe`; authentication and new commit identities remain `juscghwe-codex`.
- Reuse the existing PR for the concern. After integration, delete its merged topic branch; after milestone completion, delete the merged milestone branch once no open PR depends on it. Do not delete unmerged work or create duplicate replacement branches/PRs.
- Explain what changed and why, affected architectural contracts/public interfaces, security implications, known limitations and the relevant plan document.
- State which checks actually ran and their outcomes; distinguish failed, skipped and unrun checks. A successful build does not establish architectural correctness.
- Respect every affected module's `AGENTS.md`. Do not invent results or claim unverified functionality.
- Require explicit human review before merging changes to authorization, privilege boundaries, deployment semantics or firewall execution. Do not automatically merge any PR.
- For documentation work, validate relative references, run `git diff --check`, inspect the changed-file list, and confirm the working tree after committing. If unrelated user changes exist, report them rather than deleting them.

The current architecture audit and approval gates are in [0001-architecture-readiness](docs/plans/0001-architecture-readiness.md). Open decisions are not implementation authorization.

## Durable development workflow

### Recovery-first startup

At the beginning of a task, including after an interruption:

1. Identify the canonical repository (`juscghwe/homelab-manager`) and the correct base branch from actual refs, PR metadata and the task. A fork is not authoritative. Keep `origin` pointed at `https://github.com/juscghwe/homelab-manager.git` in active checkouts; inspect and preserve historical remote-tracking refs before pruning anything.
2. Inspect the working tree, staging area, local and remote branches, recent commits/reflogs, remotes and existing worktrees. Preserve unrelated edits and unpublished work.
3. Check for an existing feature branch and PR before creating either. Reuse completed work and avoid duplicate replacement branches or PRs.
4. Review the latest durable Git checkpoint and the repository-local plan or PR progress record. Continue from the next incomplete development chunk.

Never restart implementation merely because a conversation was interrupted. Git and the actual filesystem are authoritative evidence of work; assistant conversation state is not a substitute. Do not reset, clean, overwrite or discard work to obtain a convenient starting point.

### Plan coherent development chunks

For substantial implementation, establish a short roadmap before coding. Reuse the relevant milestone plan and add execution progress there or in a focused plan under `docs/plans/`, rather than duplicating architectural contracts. Prefer 3–6 logical chunks, each recording:

- Purpose and affected files/modules.
- Dependencies and expected Git checkpoint.
- Relevant tests and a concrete completion condition.

Keep completed checkpoints, actual verification results and the next outstanding item discoverable without reconstructing a conversation. A feature PR may contain several meaningful commits; each chunk need not become its own PR.

### Save, validate, commit and publish

Prefer small, independently understandable functional changes. Commit each coherent stage once it reaches a safe checkpoint. A commit may include several related files; do not create one commit per file merely to increase frequency. Avoid spending an entire large implementation task with extensive uncommitted changes or mixing unrelated modifications that cannot be committed independently.

Use this persistence order:

1. Save project files in an identifiable persistent checkout.
2. Preserve coherent changes in local Git commits on a named topic branch.
3. Push safe checkpoints to the canonical GitHub repository.
4. Maintain a draft PR as the remote review and recovery point.

Temporary execution directories, conversation history and local-only uncommitted files are insufficient long-term recovery mechanisms. Never keep substantial implementation solely in a disposable directory. When using an isolated Codex worktree, identify its branch and push meaningful checkpoints to GitHub.

Before starting the next implementation stage, save and validate the previous coherent stage, review its diff, stage only relevant paths, commit it, push when safe, and record the next outstanding item. Never push secrets, credentials, unrelated changes, generated temporary artifacts or known unsafe content merely to create a checkpoint. If publication is blocked, preserve a local commit and record the exact blocker; do not claim remote durability.

A valid intermediate commit may implement only part of a feature if its incomplete state is explicit. Record which tests passed, failed or were not run; never claim unperformed validation. Unfinished work stays on a topic branch and, when published, in a draft PR. Mark it ready only when its acceptance criteria are met and any remaining validation limitations are explicit.

There is no requirement to squash development commits manually before review. The maintainer may choose squash merging to consolidate approved work; agents must not rewrite published history or automatically squash, merge or discard it.

### Authentication, naming and PR boundaries

- Before remote mutations, verify `gh api /user --jq .login` returns `juscghwe-codex`. Never fall back to personal-account credentials. New Git author and committer identities must both be `juscghwe-codex <340593538+juscghwe-codex@users.noreply.github.com>`; verify commit metadata before pushing.
- Assign new PRs and issues to the maintainer `juscghwe`; authentication and new commit identities remain `juscghwe-codex`.
- Use `<type>/<kebab-case-subject>` topic branches and Conventional Commit/PR subjects `<type>(<optional-scope>): <subject>` (omit the parentheses when there is no scope). Types include `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `perf`, `style` and `chore`. Supporting fixes and documentation belong on the same feature branch when related. Avoid generic `work`, `temp` or `patch` branches and unnecessary replacement branches.
- Reserve `milestone/<number>-<kebab-case-goal>` for milestone integration branches and `milestone: <subject>` for final milestone PRs. Discover and preserve the existing integration branch. Feature work targets its documented milestone branch; a genuine unmerged prerequisite may use a documented temporary stacked PR base. Explicitly authorized repository-wide housekeeping can branch from and target `main` independently. Never implement directly on `main` or a milestone integration branch.
- Keep one cohesive feature or interface per PR. Prefer separate engine, CLI, privilege-boundary and integration review units when their responsibilities differ. Avoid giant mixed-feature PRs and individual-helper/file micro-PRs.
- Each implementation PR includes relevant tests and affected documentation. Explain scope, dependencies/base, security impact, public interface changes, actual validation results and limitations. Keep dependency references and milestone tracking current; after prerequisite integration, verify the dependent PR contains only its intended feature diff.
- Never force-push shared branches or auto-merge PRs. The maintainer remains responsible for review, approval and merge.

### Interruption and limited execution budgets

Context limits, usage exhaustion, connection loss and hard interruptions are normal operational conditions. Checkpoint incrementally throughout execution; do not rely on a final cleanup step. Frequent normal checkpoints are mandatory because an interruption may occur without warning.

If the remaining execution budget is insufficient for another complete chunk, do not begin an unrelated or high-risk change. Preserve the current safe implementation, commit and push coherent work where possible, and record unresolved work and actual verification results. Leave incomplete PRs in draft. Do not promise automatic resumption when resource limits reset or claim an interrupted task is complete.

### Propagating repository-wide instructions

A housekeeping PR against `main` remains independent of ongoing milestone implementation. After the maintainer approves and merges it, bring its instruction changes into the active milestone through a reviewable, non-destructive integration. Preserve the milestone's newer architecture and scoped instructions: never replace its `AGENTS.md` wholesale with an older copy from `main`, rewrite milestone history or merge automatically. Until that reviewed integration is available, apply these checkpoint principles to ongoing work and explicitly record that policy synchronization is still pending.
