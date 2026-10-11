// Package gitinspect observes registered Git checkouts without changing them.
// It provides engine facts, not a CLI schema or deployment evidence.
package gitinspect

import "time"

type Code string

const (
	InvalidArgument       Code = "INVALID_ARGUMENT"
	UnknownRepository     Code = "UNKNOWN_REPOSITORY"
	CapabilityDisabled    Code = "CAPABILITY_DISABLED"
	MissingRepository     Code = "MISSING_REPOSITORY"
	InvalidRepository     Code = "INVALID_REPOSITORY"
	UnsupportedRepository Code = "UNSUPPORTED_REPOSITORY"
	UnsafePath            Code = "UNSAFE_PATH"
	UnsafeConfiguration   Code = "UNSAFE_CONFIGURATION"
	IncompleteRepository  Code = "INCOMPLETE_REPOSITORY"
	GitUnavailable        Code = "GIT_UNAVAILABLE"
	CommandFailed         Code = "COMMAND_FAILED"
	UnexpectedOutput      Code = "UNEXPECTED_OUTPUT"
	Canceled              Code = "CANCELED"
	Timeout               Code = "TIMEOUT"
	OutputLimit           Code = "OUTPUT_LIMIT"
	MetadataLimit         Code = "METADATA_LIMIT"
	RemoteFailure         Code = "REMOTE_FAILURE"
)

// Error contains only fixed reason/operation names, never Git stderr or values.
type Error struct {
	Code      Code
	Operation string
}

func (e *Error) Error() string                  { return "git inspection: " + string(e.Code) + " during " + e.Operation }
func failure(code Code, operation string) error { return &Error{Code: code, Operation: operation} }

type HeadState string

const (
	HeadAvailable HeadState = "available"
	HeadUnborn    HeadState = "unborn"
)

type Head struct {
	State    HeadState
	OID      string
	Branch   string
	Detached bool
}

// Counts are per pathname/category; one path may be both staged and unstaged.
// Rename detection is disabled, so an add/delete is counted as two paths.
type Dirty struct{ Staged, Unstaged, Untracked, Conflicted int }

func (d Dirty) IsDirty() bool { return d.Staged+d.Unstaged+d.Untracked+d.Conflicted != 0 }

type RemoteState string

const (
	RemoteNotRequested  RemoteState = "not_requested"
	RemoteNotConfigured RemoteState = "not_configured"
	RemoteAvailable     RemoteState = "available"
	RemoteRefMissing    RemoteState = "ref_missing"
	RemoteUnavailable   RemoteState = "unavailable"
	RemoteUnsupported   RemoteState = "unsupported"
)

type Remote struct {
	State      RemoteState
	OID        string
	Failure    Code
	ObservedAt time.Time
}

type Relation string

const (
	Equal         Relation = "equal"
	Ahead         Relation = "ahead"
	Behind        Relation = "behind"
	Diverged      Relation = "diverged"
	Unknown       Relation = "unknown"
	NotApplicable Relation = "not_applicable"
)

type Evidence struct {
	Shallow, Partial bool
	ObjectFormat     string
}
type Result struct {
	RepositoryID string
	Head         Head
	Dirty        Dirty
	Remote       Remote
	Relation     Relation
	Evidence     Evidence
	ObservedAt   time.Time
}

// Request.Remote is the only switch that permits contacting the registered ref.
type Request struct{ Remote bool }

// Options are trusted application settings, never repository/registration data.
// TrustedCredentialHelper is an explicit admin-selected executable/argv, used
// only for requested remote queries. It must not come from untrusted config.
type Options struct {
	GitPath                 string
	Timeout                 time.Duration
	MaxOutput               int
	MaxMetadata             int
	MaxEntries              int
	TrustedCredentialHelper []string
}
