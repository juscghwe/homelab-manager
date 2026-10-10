# V1 acceptance criteria

## Local Git and CLI
- Detect dirty tracked AND untracked files without sudo; detect remote commit via `ls-remote` without fetching.
- Different commit hashes alone do not establish fast-forward/divergence; verify ancestry before classifying.
- Support single repo/multiple stacks and multiple repo/one host; only affected stack(s) need deployment when their complete tracked input scope is known. Shared/unknown inputs require explicit uncertainty and conservative impact assessment (D6), not a false unchanged result.
- Dirty/ahead/diverged state blocks unattended apply and presents a diff/reconciliation path without deleting edits.
- Controller offline does not prevent local CLI use or interrupt running containers.
- Execute the [Milestone 1 fixture matrix](PLANS.md#1--agent-only-git-core), including unborn/shallow states, missing refs and unknown ancestry; local inspection remains available when the remote fails.
- Status does not fetch or mutate checkout/index/refs/applied records. Explicit staging never overwrites active edits or marks a revision deployed; approved candidates remain pinned when remote refs move.
- Applied revision is per host/stack and may be unknown; Git `synced` does not imply deployed or healthy. Shared-repository stacks retain distinct applied records.
- Reject escaping/symlink paths and malicious Git arguments/configured execution helpers. Status reports safe metadata; secret/untracked/ignored contents never enter output. Test approved diff-output restrictions separately.
- New runtime mutations with missing required evidence are denied; dirty/ahead/diverged conflict resolution preserves edits and unpublished commits.

## Docker operations
- With capabilities disabled, unauthorized Docker commands are denied.
- Opt-in status returns images, IDs, state, health and metrics without env/secrets.
- `pull + up`, `up without pull`, and `recreate without pull` are distinct commands and produce auditable records; optional explicit `stop` is separately authorized.
- Maintenance-window scheduling and authenticated `deploy now` both honor preflight and authorization.
- Simulated failed health check triggers best-effort restoration and reports limitations, not false complete data rollback.
- Docker status/deploy/stop capabilities are checked independently; Docker permission never enables firewall apply. Effective config fingerprints must not expose secret values or hashes derived from them.
- Failed verification/confirmation never advances a successful applied marker; partial rollback remains an explicit uncertain runtime outcome.

## Firewall/network
- Read nftables data and basic host topology; no attempt to infer Caddy/Jellyfin dependencies.
- Four policy-source types are supported; independent Git source can be represented without an agent.
- Router with unknown policy defaults to *assumed permitted*, not denied; never label this as verified connectivity.
- Permanent manually modeled router rules exist in normal/current model and survive exiting preview mode.
- Preview does not write to Git, system firewall or persistent baseline model.
- Known deny changes a modeled path's result and highlights impacted connections.
- Changes affect only expressly owned nftables tables/chains and never flush Docker-owned tables.
- Broken connectivity triggers host-local rollback without depending on controller reachability.
- Test all four canonical source labels and independent edit/apply permissions, including manual policies on discovered devices. No source label alone grants runtime authority.
- Unknown policy/VLAN differences do not imply `blocked` or `no_path`; positively established physical disconnection may yield `no_path`. Modeled `permitted` is never reported as measured reachability.
- Manual baseline entries survive restart and preview discard; preview carries both proposed context and an access verdict. Annotations create neither allow rules nor service dependencies.
- Failed syntax/semantic preflight or missing independent authorization prevents any firewall mutation. An old authenticated connection, status message or mismatched/expired confirmation cannot disarm rollback.
- Rollback remains armed across agent restart and controller outage; fresh-channel confirmation is bound to the exact transaction/revision. Test the approved recovery design in disposable environments before claiming reboot/crash guarantees.

## UI and CI portability
- API/CLI operations exist without UI; UI calls identical authorized APIs.
- No Git hosting product hard dependency: plain remote Git over SSH works; Gitea/GitHub/GitLab can be adapters.
- Git-based change remains auditable, independently of CI provider. Git-only devices are never marked runtime-updated solely because Git changed.
- Standalone, headless and GUI requests enforce identical revision, preflight and capability rules; equivalent local confirmation preserves remote firewall requirements.
- Controller caches cannot overwrite desired Git configuration or agent-owned applied records. Controller-owned manual topology persists independently of preview/cache eviction.
- Matching authenticated retries return the recorded result without duplicate mutations; altered payload/identity, expired transitions and out-of-order APPLY/CONFIRM are rejected. Concurrent operations cannot race shared repository state.

## Testing environment
Use throwaway Git repositories, mocked Docker calls, disposable network namespaces/VMs for nftables integration and synthetic topology graphs. Never test privileged mutations against actual home infrastructure.
