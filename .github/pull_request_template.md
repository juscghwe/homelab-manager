## Intent

What V1 requirement or approved design decision does this change implement?
What concrete problem does it solve, and what changes for the caller/reviewer?

- Plan document and approved decisions:
- Scope and known limitations:

## Affected contracts

- Docs / APIs / agent behavior / privileged behavior / topology semantics:

## Security and authority impact

What can be read, modified, or executed that could not before?

- Authorization / privilege boundaries / deployment semantics / firewall impact:
- Required human reviewer for sensitive contract changes:

## Verification

- Tests run:
- Commands and actual results (pass/fail/skipped/unrun):
- Documentation links and `git diff --check` (when applicable):
- Disposable test environment used (if privileged):
- Known gaps:

## Compatibility

Do standalone agent and headless modes still work without the UI?
State whether this was verified or remains unimplemented/unrun.

## Integration

- Target milestone branch and plan: (current: `milestone/1-agent-git-core`; see `docs/PLANS.md`). Topic PRs target the milestone; only reviewed milestone integration PRs target `main`.
- Keep one coherent concern and stage only relevant files.
- Use `<type>/<kebab-case-subject>` for topic branches and `<type>: <subject>` for commit subjects and PR titles (`feat`, `fix`, `docs`, etc.). Reserve `milestone/<number>-<goal>` and `milestone: <subject>` for milestone integration. Branch and PR types describe the overall concern; related `fix:`, `test:`, and `docs:` commits may stay on a `feat/...` branch.
- Reuse the existing branch/PR for the concern; no generic `work` branches or duplicate replacements. Delete merged topic branches after integration, preserving unmerged work and branches with open dependent PRs.
- Confirm no secrets, generated temporary data or local environment settings are included.
- Require explicit human review for authorization, privilege, deployment or firewall changes. Do not auto-merge.
