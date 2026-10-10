# Security and privilege model

## Actors and authority
- Git repo reader may observe tracked configuration; Git writer may propose/commit changes.
- Controller may coordinate authorized jobs but must not gain implicit unrestricted root access.
- Host-native agent performs Git checks as dedicated unprivileged service identity.
- Docker executor is an explicitly authorized privileged boundary. A rootful Docker Compose deployment of arbitrary approved YAML can be **root-equivalent**: Compose may request host binds, privileged containers, namespaces, capabilities or devices.
- Firewall executor has independent authorization; Docker deployment credentials do not authorize firewall changes.
- A modeled device (including one from Git) is not automatically a managed device.

## Credentials
- mTLS for the small worker remote interface (direct CI as well as controller) is the proposed default; credential issuance, revocation and caller scope still need review before that interface is implemented. Agent private keys generated and retained on host, with one-time, authenticated enrollment and revocation.
- Git credentials are scoped per repository/operation; avoid controller-shared keys that can write across all hosts.
- CI may use scoped short-lived credentials where available. Prevent replay, enforce authorized host/stack/revision and transaction IDs.
- Do not transmit/collect raw `.env`, secrets, keys, resolved secret config, `docker inspect` environment fields, or persistent volumes.

## Unprivileged Git inspection

- Treat repository content, paths and Git configuration as untrusted inputs. Execute fixed Git argument vectors without shell interpolation; validate registered roots and reject paths escaping them (including symlinks).
- Status reports categories and safe metadata. Diffs are limited to approved tracked configuration paths; never read ignored/untracked file contents, host-local secret files or volume data for display. A tracked file may still contain a secret: unrestricted diff output is not automatically safe. Approve content-output rules before exposing diffs through CLI/API/UI.
- Inspection must not execute repository-selected hooks, external diff/textconv helpers, filters or arbitrary credential helpers. Explicit fetch/staging must document trusted transport/helper configuration, avoid untrusted checkout hooks/filters, and preserve TLS/SSH host verification. Do not disable legitimate platform authentication blindly.
- Unknown ancestry, missing objects and unavailable remote observations are explicit states, not evidence that applying is safe. Dirty/ahead/diverged trees require a separate explicit resolution that preserves local work.
- Do not compute or expose hashes over resolved secret-bearing configuration as a substitute for sanitization; hashes can reveal low-entropy secret values. D6 approves explicit non-secret tracked inputs; the [identity proposal](plans/0002-agent-git-core.md#tracked-identities-and-change-detection) specifies how to avoid reading secret contents for reporting.

## Privileged execution
- The agent stays unprivileged, without general sudo, Docker-group membership or unrestricted Docker socket access. The installer provisions small privileged executables and explicit per-operation authorization (for example fixed sudoers entries). Helpers, policy, registered paths and authorization configuration are root-owned and not writable by the agent.
- Never expose a Docker TCP API or mount unrestricted `docker.sock` into the management agent/UI/controller.
- A locally privileged executor may accept only explicit operations and configured stack IDs; never arbitrary shell fragments, file paths or Docker flags from remote callers.
- For every operation, use fixed allowlisted local paths, defined unit/capability boundaries, and a documented approval policy. Separate read-only inspection from mutation where possible.
- Authorization is about WHO approves WHAT revision and operation, not simply whether a certificate exists. A Git hash alone does not prove authorization.
- Review the [fixed helper interface proposal](plans/0001-architecture-readiness.md#d1-helper-interface-and-privilege-proposal) before writing privileged code. A constrained wrapper does not remove root-equivalent powers in approved Compose content.
- For nftables, own isolated policy tables/chains; do not flush Docker-owned or unrelated firewall state. Verify supported hook semantics prior to application.

## Fail safe behavior
- Reject unattended updates when local Git is dirty/ahead/diverged or a required approval is absent.
- Preserve existing running services if controller/Git become unreachable.
- Before remote firewall application, stage valid rules, snapshot managed state, arm a durable host-local rollback deadline independent of agent/controller processes, then apply atomically when possible. Verify a **fresh** authenticated management connection and explicitly confirm before disarming rollback.
- An explicit local administrator may deploy a complete available revision offline only with locally verifiable approval and all normal validation/conflict checks. Offline mode is never an automatic remote-error fallback or permission to overwrite local work.
- Review agent restart, reboot, interrupted transactions and confirmation/timeout races in disposable environments before claiming recovery guarantees. Console confirmation is separately authorized and cannot bypass remote recovery silently.
- Report partial failures and limits. Rolling back a Compose revision does not restore databases or persistent volumes after migrations.

## Security review gates
Required negative tests: forged job/revision; replayed confirmation; path traversal and shell injection; agent outage; missing Git commit; dirty local tree; Docker status secrets leakage; unauthorized stop; unauthorized firewall apply; firewall breaks connectivity; rollback timer survives a daemon restart; no unintended edits outside owned firewall tables.
