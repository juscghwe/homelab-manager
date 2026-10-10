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
- Reuse the existing PR for the concern. After integration, delete its merged topic branch; after milestone completion, delete the merged milestone branch once no open PR depends on it. Do not delete unmerged work or create duplicate replacement branches/PRs.
- Explain what changed and why, affected architectural contracts/public interfaces, security implications, known limitations and the relevant plan document.
- State which checks actually ran and their outcomes; distinguish failed, skipped and unrun checks. A successful build does not establish architectural correctness.
- Respect every affected module's `AGENTS.md`. Do not invent results or claim unverified functionality.
- Require explicit human review before merging changes to authorization, privilege boundaries, deployment semantics or firewall execution. Do not automatically merge any PR.
- For documentation work, validate relative references, run `git diff --check`, inspect the changed-file list, and confirm the working tree after committing. If unrelated user changes exist, report them rather than deleting them.

The current architecture audit and approval gates are in [0001-architecture-readiness](docs/plans/0001-architecture-readiness.md). Open decisions are not implementation authorization.
