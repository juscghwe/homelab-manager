# Intent and product contract

## Problem
Managing dozens of Linux VMs and Docker Compose stacks through per-host SSH sessions creates configuration drift, missed image updates and poor visibility into network access policies.

## Intended product
A **small Git-native agent** runs on Linux hosts. The agent knows about approved repositories and stack paths, reports Git/runtime identities, and can execute explicitly allowed local operations. A **headless controller** can coordinate agents; an entirely **optional GUI** can edit tracked configurations, visualize access policies, and preview proposed changes. Existing Git hosts and CI providers remain replaceable.

## V1 deliverables
1. Lightweight unprivileged Go agent using system Git: CLI and small authenticated remote operation interface, versioned JSON, optional systemd operation; status, Git remote/local/deployed revision checks, dirty-state checks, diff, validate, stage, apply and rollback attempts. CI may trigger the worker directly without a controller.
2. Docker Compose: controlled pull/apply/recreate; explicit optional stop; maintenance windows and immediate authorized deployment; optional sanitized status and resource telemetry.
3. Firewall: Linux nftables inventory, permanent modeled devices/rules, topology-access view, preview of proposed changes, narrow own-table policy application with local rollback.
4. Optional headless backend and fleet CLI/API: agent coordination, enrollment/authentication, dispatch, scheduling, job status, aggregated host facts, Git-provider integration, topology evaluation and preview. Renovate or equivalent update discovery/notifications is an optional centrally managed capability, not an agent dependency.
5. Optional GUI: group nodes, connect/annotate nodes manually, inspect effective modeled access, edit Git-backed text/structured policies, preview differences, review/apply where authorized.
6. Remote Git by default, with both one-repo-many-stacks and many-repos-one-or-more-stacks layouts. Hosted Git platform optional; a reachable plain Git remote is valid.

Manually maintained topology (devices, groups, relationships, annotations and modeled rules) is authoritative in a user-selected Git repository, preferably structured YAML. Remote hosting is preferred; local Git is supported. Runtime observations retain their provenance; caches/derived graphs are reconstructible. No competing authoritative topology database.

Offline inspection/history and existing workloads remain available. A local administrator may explicitly deploy a complete already available revision with normal authorization and validation. Remote failure never authorizes automatic offline deployment; firewall recovery remains local.

## Non-goals
- Not a replacement for the Docker daemon, Git server, or CI runner.
- Not a general-purpose root remote execution tool.
- No automatic crawling of Caddy/application configurations to infer service dependencies.
- No demand for an agent on routers/switches before those devices can appear in the model.
- No claim that observed network access means an application works.
- No automatic migration/rollback of databases, volumes or application state.
- No complete multi-vendor firewall editor in V1; nftables is the first writable backend, with other Linux rule sources as later adapters.

## Success criteria
A user can manage the same host through CLI, CI, or GUI; can see Git drift without sudo; can deploy an approved immutable revision with least-privilege control-plane interfaces; can visualize assumed/known access paths even with unmanaged network devices; can preview a change without changing baseline state; and can recover from a remote firewall lockout through a locally armed rollback.
