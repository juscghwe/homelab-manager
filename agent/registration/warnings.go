package registration

import "strings"

// Warning contains a stable reason and schema location, never raw paths or
// configuration values. Future installers must obtain location-specific consent
// for SuspiciousPath; that consent cannot authorize secret reads or privileges.
type Warning struct {
	Code  WarningCode
	Field string
}

type WarningCode string

const SuspiciousPath WarningCode = "SUSPICIOUS_PATH"

func pathWarnings(path, field string) []Warning {
	for _, part := range strings.Split(path, "/") {
		switch strings.ToLower(part) {
		case "secret", "secrets", "data", "volume", "volumes":
			return []Warning{{Code: SuspiciousPath, Field: field}}
		}
	}
	return nil
}
