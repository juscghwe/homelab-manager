// Package registration loads and validates host-local agent declarations.
// It does not execute Git, inspect file contents in checkouts, or grant privileges.
package registration

import "slices"

const (
	DefaultPath   = "/etc/homelab-manager/agent.json"
	SchemaVersion = 1
	MaxBytes      = 1 << 20
)

// Config is the wire model. Decode applies defaults; Validate validates a copy.
// Do not log Config: it contains administrator-provided paths and remote data.
type Config struct {
	SchemaVersion int                   `json:"schema_version"`
	Host          string                `json:"host"`
	Repositories  map[string]Repository `json:"repositories"`
	Stacks        []Stack               `json:"stacks"`
	Capabilities  Capabilities          `json:"capabilities"`
}

type Repository struct {
	Root   string `json:"root"`
	URL    string `json:"url,omitempty"`
	Branch string `json:"branch,omitempty"`
}

type Stack struct {
	ID             string   `json:"id"`
	Repository     string   `json:"repository"`
	Path           string   `json:"path"`
	ComposeProject string   `json:"compose_project"`
	TrackedInputs  []string `json:"tracked_inputs"`
}

// Capabilities are requests, not authorization or evidence of implementation.
type Capabilities struct {
	GitStatus     bool `json:"git_status"`
	DockerStatus  bool `json:"docker_status"`
	DockerDeploy  bool `json:"docker_deploy"`
	DockerStop    bool `json:"docker_stop"`
	NFTablesApply bool `json:"nftables_apply"`
}

// Registration is a validated snapshot. Accessors return values or copies.
// Later operations must revalidate filesystem state at the time of use.
type Registration struct {
	config   Config
	warnings []Warning
}

// Warnings reports metadata-only security concerns, not permission to read
// contents or execute capabilities. The returned slice is an independent copy.
func (r *Registration) Warnings() []Warning { return slices.Clone(r.warnings) }

func (r *Registration) Host() string               { return r.config.Host }
func (r *Registration) Capabilities() Capabilities { return r.config.Capabilities }
func (r *Registration) RepositoryIDs() []string    { return sortedKeys(r.config.Repositories) }
func (r *Registration) Repository(id string) (Repository, bool) {
	v, ok := r.config.Repositories[id]
	return v, ok
}
func (r *Registration) StackIDs() []string {
	ids := make([]string, 0, len(r.config.Stacks))
	for _, s := range r.config.Stacks {
		ids = append(ids, s.ID)
	}
	slices.Sort(ids)
	return ids
}
func (r *Registration) Stack(id string) (Stack, bool) {
	for _, s := range r.config.Stacks {
		if s.ID == id {
			s.TrackedInputs = slices.Clone(s.TrackedInputs)
			return s, true
		}
	}
	return Stack{}, false
}

// Error contains only fixed schema locations and reason codes. It deliberately
// does not wrap decoder/OS errors, whose messages may contain secret data.
type Error struct {
	Code  Code
	Field string
}

func (e *Error) Error() string { return "registration: " + string(e.Code) + " at " + e.Field }

type Code string

const (
	InvalidJSON           Code = "INVALID_JSON"
	TooLarge              Code = "TOO_LARGE"
	DuplicateField        Code = "DUPLICATE_FIELD"
	UnknownField          Code = "UNKNOWN_FIELD"
	InvalidSchema         Code = "INVALID_SCHEMA"
	UnsupportedSchema     Code = "UNSUPPORTED_SCHEMA"
	InvalidID             Code = "INVALID_ID"
	DuplicateID           Code = "DUPLICATE_ID"
	UnknownRepository     Code = "UNKNOWN_REPOSITORY"
	InvalidPath           Code = "INVALID_PATH"
	MissingPath           Code = "MISSING_PATH"
	OverlappingPath       Code = "OVERLAPPING_PATH"
	InvalidRemote         Code = "INVALID_REMOTE"
	InvalidBranch         Code = "INVALID_BRANCH"
	InvalidCapability     Code = "INVALID_CAPABILITY"
	UnsupportedCapability Code = "UNSUPPORTED_CAPABILITY"
	ReadFailure           Code = "READ_FAILURE"
)

func fail(code Code, field string) error { return &Error{Code: code, Field: field} }
