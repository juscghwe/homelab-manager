package registration

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)
var sshUser = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
var sshHost = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*$`)
var remotePath = regexp.MustCompile(`^[a-zA-Z0-9_./~+-]+$`)

// Validate checks and copies declarations, using filesystem metadata only.
// It never runs Git, traverses directory contents, or opens tracked input files.
// Roots/stack directories/declared input files or directories must exist.
func Validate(cfg Config) (*Registration, error) {
	if cfg.SchemaVersion != SchemaVersion {
		return nil, fail(UnsupportedSchema, "schema_version")
	}
	if !idPattern.MatchString(cfg.Host) {
		return nil, fail(InvalidID, "host")
	}
	if cfg.Repositories == nil {
		return nil, fail(InvalidSchema, "repositories")
	}
	if cfg.Stacks == nil {
		return nil, fail(InvalidSchema, "stacks")
	}
	out := cfg
	out.Repositories = make(map[string]Repository, len(cfg.Repositories))
	out.Stacks = make([]Stack, 0, len(cfg.Stacks))
	roots := map[string]*os.Root{}
	defer func() {
		for _, root := range roots {
			root.Close()
		}
	}()
	var warnings []Warning
	var infos []os.FileInfo
	var paths []string
	for i, id := range sortedKeys(cfg.Repositories) {
		field := "repositories[" + strconv.Itoa(i) + "]"
		if !idPattern.MatchString(id) {
			return nil, fail(InvalidID, field)
		}
		repo := cfg.Repositories[id]
		if !validRemote(repo.URL) {
			return nil, fail(InvalidRemote, field+".url")
		}
		if !validBranch(repo.Branch) {
			return nil, fail(InvalidBranch, field+".branch")
		}
		path, info, err := absoluteMetadata(repo.Root, field+".root")
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || path == string(filepath.Separator) {
			return nil, fail(InvalidPath, field+".root")
		}
		for j, prior := range paths {
			if overlaps(path, prior) || os.SameFile(info, infos[j]) {
				return nil, fail(OverlappingPath, field+".root")
			}
		}
		root, err := os.OpenRoot(path)
		if err != nil {
			return nil, fail(InvalidPath, field+".root")
		}
		roots[id] = root
		opened, err := root.Stat(".")
		if err != nil || !os.SameFile(info, opened) {
			return nil, fail(InvalidPath, field+".root")
		}
		paths = append(paths, path)
		infos = append(infos, info)
		warnings = append(warnings, pathWarnings(path, field+".root")...)
		repo.Root = path
		out.Repositories[id] = repo
	}
	seenIDs := map[string]bool{}
	seenProjects := map[string]bool{}
	for i, original := range cfg.Stacks {
		s := original
		field := "stacks[" + strconv.Itoa(i) + "]"
		if !idPattern.MatchString(s.ID) {
			return nil, fail(InvalidID, field+".id")
		}
		if seenIDs[s.ID] {
			return nil, fail(DuplicateID, field+".id")
		}
		seenIDs[s.ID] = true
		if !idPattern.MatchString(s.Repository) {
			return nil, fail(InvalidID, field+".repository")
		}
		root, ok := roots[s.Repository]
		if !ok {
			return nil, fail(UnknownRepository, field+".repository")
		}
		if !idPattern.MatchString(s.ComposeProject) {
			return nil, fail(InvalidID, field+".compose_project")
		}
		if seenProjects[s.ComposeProject] {
			return nil, fail(DuplicateID, field+".compose_project")
		}
		seenProjects[s.ComposeProject] = true
		path, info, err := relativeMetadata(root, s.Path, field+".path")
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fail(InvalidPath, field+".path")
		}
		warnings = append(warnings, pathWarnings(path, field+".path")...)
		s.Path = path
		for _, prior := range out.Stacks {
			if prior.Repository == s.Repository && overlaps(prior.Path, s.Path) {
				return nil, fail(OverlappingPath, field+".path")
			}
		}
		if len(s.TrackedInputs) == 0 {
			return nil, fail(InvalidPath, field+".tracked_inputs")
		}
		s.TrackedInputs = nil
		for j, input := range original.TrackedInputs {
			location := field + ".tracked_inputs[" + strconv.Itoa(j) + "]"
			path, info, err := relativeMetadata(root, input, location)
			if err != nil {
				return nil, err
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return nil, fail(InvalidPath, location)
			}
			for _, prior := range s.TrackedInputs {
				if overlaps(prior, path) {
					return nil, fail(OverlappingPath, location)
				}
			}
			warnings = append(warnings, pathWarnings(path, location)...)
			s.TrackedInputs = append(s.TrackedInputs, path)
		}
		slices.Sort(s.TrackedInputs)
		out.Stacks = append(out.Stacks, s)
	}
	return &Registration{config: out, warnings: warnings}, nil
}

func overlaps(a, b string) bool {
	return a == b || a == "." || b == "." || strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator))
}

func validBranch(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 255 || s == "@" || s == "HEAD" || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "refs/") || strings.HasSuffix(s, ".") || strings.Contains(s, "..") || strings.Contains(s, "@{") {
		return false
	}
	for _, c := range s {
		if unicode.IsSpace(c) || unicode.IsControl(c) || strings.ContainsRune(`~^:?*[\`, c) {
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

func validRemote(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 2048 || strings.HasPrefix(s, "-") {
		return false
	}
	for _, c := range s {
		if unicode.IsSpace(c) || unicode.IsControl(c) {
			return false
		}
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Opaque != "" || (u.Scheme != "https" && u.Scheme != "ssh") || u.Hostname() == "" || u.Path == "" || u.Path == "/" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(s, "#") {
			return false
		}
		if u.User != nil {
			if u.Scheme != "ssh" || !sshUser.MatchString(u.User.Username()) {
				return false
			}
			if _, password := u.User.Password(); password {
				return false
			}
		}
		if port := u.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return false
			}
		}
		return remotePath.MatchString(u.Path) && (sshHost.MatchString(u.Hostname()) || net.ParseIP(u.Hostname()) != nil)
	}
	// SSH scp-style syntax only; no local paths or external transport helpers.
	host, path, ok := strings.Cut(s, ":")
	if !ok || !remotePath.MatchString(path) || strings.HasPrefix(path, "-") {
		return false
	}
	if user, name, hasUser := strings.Cut(host, "@"); hasUser {
		if !sshUser.MatchString(user) {
			return false
		}
		host = name
	}
	return sshHost.MatchString(host)
}
