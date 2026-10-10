# Data model and terminology

## Core entities
- `Host`: stable identity, managed/unmanaged status, discovered interfaces/routes and optional agent ID.
- `Network`: network segment or user-declared logical group; avoid interpreting logical membership as physical bridging/VLAN connectivity.
- `Device`: router, switch, Linux host or other modelled component. May be discovered, manually entered, agent-managed or Git sourced.
- `Stack`: host + repository + tracked branch/ref + path + Compose project name. Multiple stacks can share one repository and a host can reference many repositories.
- `PolicySource`: `agent`, `git_independent`, `discovered_unmanaged`, or `manual`. Source is distinct from `can_edit_git` and `can_apply_runtime`.
- `AccessRule`: source, destination, direction, protocol, port ranges, action, provenance, authority status; explicit rule order/semantics when known.
- `Annotation`: manually authored edge/group/label; no implied connectivity or service dependency.
- `Deployment`: immutable target Git commit, host/stack, command, approval, transaction, deadline and outcome.

These are semantic fields, not a finalized serialization schema. A stack references one repository; many repositories per host does not mean many repositories per stack. Device source and rule source can differ (for example, a manual rule on a discovered router). Preserve provenance per record; neither source nor an editable Git ref grants runtime authority. `agent` is the canonical source label for a managed agent.

## Agent status (minimum meaningful fields)
- `git.local_head`: checkout commit.
- `git.remote_head`: advertised remote commit, or unavailable.
- `git.dirty`: tracked staged/unstaged changes or non-ignored untracked files in the registered repository; report categories without leaking contents. Shared repositories have repository-wide conflict checks. Ignored local secrets/data are not candidate configuration.
- `git.applied_revision`: last successfully applied tracked revision for the host/stack, owned by the agent journal; unknown/absent before a verified successful deployment. Status, fetch and stage never advance it. Stacks sharing a checkout can have different applied revisions.
- `compose.desired_hash`: sanitized effective service configuration fingerprint when inspect capability enabled.
- `compose.running_hash`: observed Compose hash/label when available, with caveat that not all runtimes expose it.
- `container.image_id`: running image content identity when Docker read capability enabled; registry digest may be additional metadata.
- `runtime.state`, `runtime.health` and `runtime.metrics`: optional, only when enabled.

Attach scope, observation time and availability/error information. An unavailable remote, absent applied record, unborn local branch or disabled runtime capability is not an empty successful result. An old remote observation may be retained only as explicitly stale. A desired Git commit, local HEAD, staged candidate and applied revision are distinct; runtime evidence remains separate. Never reconstruct deployment success from HEAD alone.

## Git state labels
- `synced`: clean local HEAD equals the current advertised remote commit; describes Git synchronization only. Report applied-revision equality and runtime verification separately, including unavailable evidence.
- `update_available`: remote ref differs and is known to be ahead; only after ancestry established.
- `dirty`: uncommitted tracked or non-ignored untracked changes in the registered repository, as defined by `git.dirty` above.
- `ahead` / `diverged`: unpublished local commits / incompatible history; don't auto-reset.
- `remote_unavailable`: cannot query remote; do not conclude nothing changed.
- `ancestry_unknown`: unequal commits with insufficient history/objects to classify ahead, behind or divergence; shallow history and failed acquisition must not produce a false classification.
- `drifted_runtime`: observed runtime differs from last applied/desired configuration (when measurable).

Labels are independent observations, not a single mutually exclusive state enum: a dirty tree may also be ahead, and a synchronized checkout may not be deployed. `update_available` describes the remote relative to local HEAD, not whether the runtime already uses that revision. Unknown required evidence blocks unattended mutation without blocking read-only inspection. See [Milestone 1 fixtures](PLANS.md#1--agent-only-git-core).

## Connectivity result labels
- `permitted`: all evaluated known policies permit a modeled path; not necessarily tested.
- `assumed_permitted`: no known blocker, but one or more policies/devices unknown. DEFAULT for unknown restrictions.
- `blocked`: a known applicable policy rejects a path.
- `no_path`: discovery positively establishes no usable path; never infer this from unknown router policy or different VLAN IDs alone.
- `preview_change`: result context for a proposed delta, not an alternative connectivity verdict. A preview still reports `permitted`, `assumed_permitted`, `blocked` or `no_path`, with baseline and proposed provenance kept distinct.

Every result records whether evidence was observed, Git-declared, manually modelled, or inferred by open-world default. Do not silently collapse `permitted` and `assumed_permitted` into "confirmed reachable".
