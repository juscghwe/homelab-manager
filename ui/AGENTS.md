# Optional UI instructions

Read root `AGENTS.md`, `docs/INTENT.md`, `docs/DESIGN.md` and `docs/DATA_MODEL.md`.

## Responsibility
Graphical client of the headless API: host/stack status, Git-backed configuration editing, revision comparisons, network access map, manual groups/annotations, and preview of proposed firewall changes.

## Constraints
- Must be removable without disabling controller, agent, CLI or CI workflows.
- Do not implement independent deployment, firewall parsing or privileged behavior in UI code.
- Explicitly distinguish observed, declared in Git, manually modeled and assumed policy; visually distinguish baseline/current model from temporary proposed-change preview.
- An annotation edge must never itself imply a firewall allow rule or app dependency.
- Unmanaged devices and modeled restrictions are permanent baseline; toggling preview only adds/removes proposed deltas.
- Show source host, affected protocol/port and confidence/provenance for each connectivity result.
- Require explicit confirmation for runtime mutation; show whether action means Git commit, approved revision, or actual host apply.
- Edit manual topology as structured Git files through backend Git-provider operations; support the local-Git adapter when no hosting is configured. Persisting requires a Git commit, independently of runtime apply. No controller database becomes topology authority.
- Never show secret contents, `.env` values or raw Docker inspection environment data.

## Tests
Visual/state tests for topology baseline, change preview, permission labeling, inaccessible hosts and headless API parity; no privileged direct calls.
