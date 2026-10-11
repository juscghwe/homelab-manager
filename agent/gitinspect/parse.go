package gitinspect

import (
	"bytes"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/juscghwe/homelab-manager/agent/registration"
)

func validBranchRef(s string) bool {
	if !strings.HasPrefix(s, "refs/heads/") || !utf8.ValidString(s) || len(s) > 1024 || strings.Contains(s, "..") || strings.Contains(s, "@{") || strings.HasSuffix(s, ".") {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) || strings.ContainsRune(`~^:?*[\`, r) {
			return false
		}
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}
func safeTrackedName(name string) bool {
	return name != "" && !strings.ContainsRune(name, 0) && !strings.HasPrefix(name, "/") && path.Clean(name) == name && name != ".." && !strings.HasPrefix(name, "../") && !registration.ContentExcluded(name)
}

// parsePaths consumes ls-files --stage or ls-tree -r -z. Names remain private;
// no blob contents are read and no index/tree OIDs are returned to callers.
func parsePaths(data []byte, tree bool, format string) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if data[len(data)-1] != 0 {
		return nil, failure(UnexpectedOutput, "paths")
	}
	var names []string
	for _, record := range bytes.Split(data[:len(data)-1], []byte{0}) {
		prefix, name, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return nil, failure(UnexpectedOutput, "paths")
		}
		fields := strings.Split(string(prefix), " ")
		if len(fields) != 3 {
			return nil, failure(UnexpectedOutput, "paths")
		}
		mode := fields[0]
		oid := fields[1]
		if tree {
			oid = fields[2]
			if fields[1] != "blob" && fields[1] != "commit" && fields[1] != "tree" {
				return nil, failure(UnexpectedOutput, "paths")
			}
		} else if fields[2] != "0" && fields[2] != "1" && fields[2] != "2" && fields[2] != "3" {
			return nil, failure(UnexpectedOutput, "paths")
		}
		if !validOID(oid, format) {
			return nil, failure(UnexpectedOutput, "paths")
		}
		if !safeTrackedName(string(name)) {
			return nil, failure(UnsafePath, "tracked_paths")
		}
		if mode == "120000" || mode == "160000" || mode == "040000" {
			return nil, failure(UnsupportedRepository, "tracked_layout")
		}
		if mode != "100644" && mode != "100755" {
			return nil, failure(UnexpectedOutput, "paths")
		}
		names = append(names, string(name))
	}
	return names, nil
}
func parseStatus(data []byte) (Dirty, error) {
	var d Dirty
	if len(data) == 0 {
		return d, nil
	}
	if data[len(data)-1] != 0 {
		return d, failure(UnexpectedOutput, "status")
	}
	for _, record := range bytes.Split(data[:len(data)-1], []byte{0}) {
		if len(record) < 3 {
			return Dirty{}, failure(UnexpectedOutput, "status")
		}
		if bytes.HasPrefix(record, []byte("? ")) {
			if !safeStatusName(string(record[2:])) {
				return Dirty{}, failure(UnexpectedOutput, "status")
			}
			d.Untracked++
			continue
		}
		fields := bytes.SplitN(record, []byte{' '}, 9)
		if record[0] == 'u' {
			fields = bytes.SplitN(record, []byte{' '}, 11)
			d.Conflicted++
		} else if record[0] != '1' {
			return Dirty{}, failure(UnexpectedOutput, "status")
		}
		expected := 9
		if record[0] == 'u' {
			expected = 11
		}
		if len(fields) != expected || len(fields[1]) != 2 || !safeStatusName(string(fields[expected-1])) {
			return Dirty{}, failure(UnexpectedOutput, "status")
		}
		xy := fields[1]
		for _, c := range xy {
			if !bytes.ContainsRune([]byte(".MADRCUT"), rune(c)) {
				return Dirty{}, failure(UnexpectedOutput, "status")
			}
		}
		if xy[0] != '.' {
			d.Staged++
		}
		if xy[1] != '.' {
			d.Unstaged++
		}
	}
	return d, nil
}
func safeStatusName(name string) bool {
	return name != "" && !strings.ContainsRune(name, 0) && !strings.HasPrefix(name, "/") && path.Clean(name) == name && name != ".." && !strings.HasPrefix(name, "../")
}
