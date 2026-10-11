package registration

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Hard exclusions protect recognized secret/control-file names, not arbitrary
// secrets. Ambiguous directory names yield separate metadata-only warnings.
// Descendants are not enumerated here; future Git operations must screen them.
func safePath(s string) bool {
	if s == "" || len(s) > 4096 || !utf8.ValidString(s) || strings.ContainsAny(s, `\:*?[`) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	for _, part := range strings.Split(s, "/") {
		if part == ".." || strings.HasPrefix(part, "-") {
			return false
		}
		p := strings.ToLower(part)
		if p == ".git" || p == ".ssh" || p == ".env" || strings.HasPrefix(p, ".env.") {
			return false
		}
		for _, ext := range []string{".key", ".pem", ".p12", ".pfx"} {
			if strings.HasSuffix(p, ext) {
				return false
			}
		}
	}
	return true
}

// absoluteMetadata rejects all symlink components before inspecting descendants.
// It performs no directory enumeration or file content reads.
func absoluteMetadata(s, field string) (string, os.FileInfo, error) {
	if !filepath.IsAbs(s) || !safePath(s) {
		return "", nil, fail(InvalidPath, field)
	}
	clean := filepath.Clean(s)
	current := string(filepath.Separator)
	info, err := os.Lstat(current)
	if err != nil {
		return "", nil, pathError(err, field)
	}
	for _, part := range strings.Split(strings.TrimPrefix(clean, current), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err = os.Lstat(current)
		if err != nil {
			return "", nil, pathError(err, field)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", nil, fail(InvalidPath, field)
		}
	}
	return clean, info, nil
}

func relativeMetadata(root *os.Root, s, field string) (string, os.FileInfo, error) {
	if filepath.IsAbs(s) || !safePath(s) {
		return "", nil, fail(InvalidPath, field)
	}
	clean := filepath.Clean(s)
	current := "."
	var info os.FileInfo
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		var err error
		info, err = root.Lstat(current)
		if err != nil {
			return "", nil, pathError(err, field)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", nil, fail(InvalidPath, field)
		}
	}
	return clean, info, nil
}
func pathError(err error, field string) error {
	if errors.Is(err, os.ErrNotExist) {
		return fail(MissingPath, field)
	}
	return fail(InvalidPath, field)
}
