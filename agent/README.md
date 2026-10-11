# Agent libraries (M1-A / M1-B)

The first implemented component is a **library**, `registration`, using Go **1.27.2** and only the standard library. Linux is the supported host platform; the file loader is Linux-specific. M1-B adds the [read-only Git inspection library](gitinspect/README.md). No agent command, state/history, preparation, Docker/firewall operation, daemon or network API exists yet. The module path is `github.com/juscghwe/homelab-manager/agent`.

## Local loader and package boundary

```go
reg, err := registration.Load("") // /etc/homelab-manager/agent.json
// Alternatively: registration.Load("/absolute/development/agent.json")
if err != nil {
    var validation *registration.Error
    if errors.As(err, &validation) {
        // Code and Field are sanitized; map them into a later CLI envelope.
    }
    return
}
for _, id := range reg.RepositoryIDs() {
    repo, _ := reg.Repository(id)
    _ = repo // validated declaration for the M1-B inspection engine
}
```

Import `registration` from the module path above and `errors` from the standard library. `Load` decodes and validates. `Decode(io.Reader) (Config, error)` performs bounded JSON/schema-shape validation only, without filesystem access; its output is not yet safe for operations. `Validate(Config) (*Registration, error)` validates and normalizes a copy using filesystem metadata. Programmatic Config values have their explicit booleans respected; only Decode applies omitted-field defaults.

A successful `Registration` exposes `Host`, `Capabilities`, sorted `RepositoryIDs`/`StackIDs`, structured `Warnings`, and `Repository(id)`/`Stack(id)` lookups. Returned slices are copies, so callers cannot mutate the validated snapshot. No global configuration or future operation stubs are provided. M1-B consumes these registered identities, roots and refs through the accessors and applies its documented trusted Git execution policy. It revalidates mutable filesystem state before use. `registration.ContentExcluded(path)` exposes the existing filename exclusion policy for pre-hashing checks; it does not establish content eligibility.

## Runtime JSON schema

See the complete [runtime JSON example](../examples/host-registration.json). Its paths must be adapted to existing local directories/files; it is not a ready-to-run deployment. [The YAML illustration](../examples/host-registration.yaml) is not accepted by this loader and is not a second runtime schema.

| Field | Implemented semantics |
| --- | --- |
| `schema_version` | Required integer `1`; no implicit version migration. |
| `host` | Required stable host ID. |
| `repositories` | Required object keyed by unique repository ID; `{}` is valid. Each entry requires `root`, an existing absolute checkout directory. |
| `repositories.*.url` | Optional remote declaration. Omit for local-only registration; an explicitly empty string is rejected. HTTPS, SSH URLs and SSH scp-style syntax only; no password, HTTPS userinfo, query, fragment, external helper, `file:` URL or arbitrary local remote path. Conservative ASCII remote path characters: letters/digits and `_./~+-`. SSH usernames are supported; credentials belong in separately trusted transport configuration. No remote contact is performed. |
| `repositories.*.branch` | Optional short branch name (e.g. `main` or `feature/status`). Omit when not selected; explicitly empty, `HEAD`, full `refs/...`, option-like and invalid ref spellings are rejected. No branch existence or Git check occurs. |
| `stacks` | Required array; `[]` is valid. Each entry requires unique `id`, known `repository`, directory `path`, host-wide unique `compose_project`, and nonempty `tracked_inputs`. |
| `stacks[].path` | Repository-relative stack directory; `.` is allowed. Stack locations in the same repository cannot be equal or nested. |
| `stacks[].tracked_inputs` | Repository-relative exact regular files or directories, including shared paths outside the stack directory but inside the same repository. No glob expansion. Inputs may be shared across stacks; duplicate/equal/nested inputs within a stack are rejected. Directory descendants are not scanned here. |
| `capabilities` | Optional object of known boolean declarations. Omission or `{}` defaults only `git_status` to true. Other flags default false; explicit false disables Git status. Unknown names, non-booleans and nulls are errors. |

Host/repository/stack IDs and Compose projects use `[a-z0-9][a-z0-9_-]{0,62}`. Accepted capability names are `git_status`, `docker_status`, `docker_deploy`, `docker_stop`, `nftables_apply`. Explicit true for a future capability is stored as a request, not implemented behavior or authorization. A registration cannot grant the future helpers privileges; independently protected policy remains mandatory.

A host need not register any Compose stacks or Git repositories. For example, a host declaring a future firewall capability can use:

```json
{"schema_version":1,"host":"firewall01","repositories":{},"stacks":[],"capabilities":{"git_status":false,"nftables_apply":true}}
```

Both collections must be explicit (not omitted or null). The programmatic `Config` API likewise requires non-nil map/slice values. Repositories without stacks are also valid; a declared stack still requires an existing repository reference. Empty scopes do not authorize any operation, and the example does not implement firewall management.

JSON keys are case-sensitive. Reject duplicate keys anywhere (including escaped equivalents), unknown fields, null anywhere, malformed types, trailing JSON and invalid UTF-8. Input is limited to 1 MiB and 32 nested containers. This strictness prevents silently accepted misspellings or ambiguous duplicate declarations.

## Path validation and normalization

Syntactic checks reject empty paths, control characters, backslashes/Windows forms, glob characters, colon, option-like components and **any** `..` component, even if cleaning would stay in the root. Relative paths are required for stacks/inputs; absolute paths for roots and the local configuration loader. Accepted `.` components, repeated separators and trailing separators are normalized with `filepath.Clean`. Length is bounded at 4096 bytes.

Existence checks use metadata: roots and stack locations must be directories; inputs must be regular files or directories. All symlink components are rejected, including links whose targets stay inside a checkout. Equal/nested roots and directly detectable same-directory aliases are rejected. Within a checkout, checks use `os.Root` to constrain traversal. This does not detect arbitrary bind-mount topology or guarantee stable paths after validation; later operations must revalidate, and privileged handoff has its own protected-copy rules.

The loader accepts only an absolute `.json` path to a regular file, rejects symlink components and known excluded names before opening, and bounds reads. Linux opens refuse final symlinks and nonblocking FIFO substitution; the opened file must match the checked identity. A caller-provided alternate path is a local administrative API, never a remotely supplied override. Ownership, administrator approval and concurrent malicious filesystem changes are not authenticated by this library.

Hard exclusions (case-insensitive) remain `.git`, `.ssh`, `.env`, `.env.*`, and names ending `.key`, `.pem`, `.p12` or `.pfx`. They apply to declared paths and cannot be bypassed by acknowledging a warning. They are conservative protections, not a detector for arbitrarily named secrets. No tracked input, secret or volume contents are read by validation. A permitted directory such as `.` can contain excluded descendants; M1-D must screen actual tracked entries before reading or hashing them. M1-A does not establish Git membership or completeness of a stack's declared inputs.

### Suspicious names and future consent

Names `secret`, `secrets`, `data`, `volume` and `volumes` (whole components, case-insensitive) produce a structured `Warning{Code: SuspiciousPath, Field: ...}` instead of a validation error. For example, an existing repository under `/srv/secrets/service` is a valid location with a `SUSPICIOUS_PATH` warning at `repositories[0].root`. The check uses metadata only and does not decide whether that directory contains actual secrets. Loader paths, repository roots, stack locations and tracked inputs can each warn; one warning is emitted per affected field.

`reg.Warnings()` returns an independent copy containing only fixed codes and schema locations, never raw paths, contents or OS messages. Locations address the source declarations (repositories in sorted ID order, stacks and inputs in their original array order), not the normalized/sorted snapshot. Loader configuration-path warnings come first, followed by repository and stack/input warnings in validation order. Callers must retain the source registration to map warnings into consent prompts.

The library is noninteractive: a warning does not itself reject registration, record consent, enable an operation or authorize content reads. A future installer must explain each warning and require explicit confirmation before accepting the location. Noninteractive setup must require an explicit override bound narrowly to that warning code, host/resource, normalized path and exact configuration being accepted; a blanket `allow_suspicious_paths` switch is not accepted by this schema. A changed location/configuration requires renewed consent. The installer, persistence of consent and CLI presentation are not implemented in M1-A.

Consent approves a location only. It never overrides traversal/symlink boundaries, hard exclusions, protected helper policy or the ban on reading/hashing actual secrets and persistent volume data. Future operations must independently enforce content eligibility; a warning acknowledgement is not evidence that an input is non-secret. Directory descendants are not scanned for names or contents here.

## Errors and deterministic behavior

Failures are `*registration.Error` with a fixed reason `Code` and a schema `Field` such as `repositories[0].root` or `stacks[1].tracked_inputs[0]`. Errors never wrap raw JSON/OS messages or echo IDs, paths, remote URLs or configuration values. Do not log raw Config/Repository/Stack values either. Codes distinguish malformed/schema/unsupported/duplicate declarations, invalid identities/references/paths/remotes/capabilities, missing paths, overlap, oversized input and read failures.

Decoding checks syntax/duplicate keys in document order, then schema shape. Validation checks version/host/collections, repositories in sorted ID order, and stacks/inputs in array order. It returns the first failure; this order is stable for the same configuration regardless of Go map iteration. Registration input must not be concurrently mutated by callers; returned snapshots are safe for concurrent read access.

## Tests

Install Go 1.27.2 and development build tools on Linux. On Debian/Ubuntu:

```sh
sudo apt-get update
sudo apt-get install --no-install-recommends build-essential
```

From the repository root, run `agent/scripts/check.sh`. It checks formatting without rewriting files (`gofmt -l`), runs standard-library static analysis (`go vet`), unit tests, CGO-enabled race tests, and a CGO-disabled build. The GitHub Actions agent workflow runs this same script on pushes and pull requests. GCC/build tools are development-only dependencies; registration builds with `CGO_ENABLED=0` and does not require them at runtime.

Individual checks from `agent/`:

```sh
go test ./...
go vet ./...
CGO_ENABLED=1 go test -race ./...
CGO_ENABLED=0 go build ./...
```

The race detector requires cgo and a working compatible C compiler (GCC or Clang, selectable with `CC`). If none is installed locally, run formatting, vet, unit tests and the CGO-disabled build separately and explicitly record local race testing as unavailable; the full CI script still requires and executes race tests on Ubuntu. The library does not require development build tools at runtime. Registration tests use temporary directories and synthetic data without invoking Git. Inspection tests use disposable Git repositories and controlled loopback/mock transport; neither suite accesses actual host registration/Docker/firewall state. Coverage includes schema ambiguity, capability defaults, empty/firewall-only hosts, multiple repos/stacks/shared inputs, structured suspicious-path warnings, normalized duplicates, deterministic errors, traversal/symlinks, FIFO rejection, unreadable input files, error redaction and snapshot isolation. A `FuzzDecode` target exercises bounded JSON parsing separately from filesystem validation.

Review details and recovery are in [M1-A implementation plan](../docs/plans/0004-agent-registration.md). Subsequent units remain separate PRs under [the delivery plan](../docs/plans/0003-milestone-1-delivery.md).
