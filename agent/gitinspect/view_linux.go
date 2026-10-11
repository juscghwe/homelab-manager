//go:build linux

package gitinspect

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// view owns only private scratch. Source config is parsed as inert data through
// `config --file - --no-includes`; Git inspection never loads that config.
type view struct {
	directory, worktree, objects, format, headRef string
	root                                          *os.Root
	shallow, partial                              bool
	budget, entries                               int
	options                                       Options
}

func (v *view) close() { v.root.Close(); os.RemoveAll(v.directory) }
func openView(ctx context.Context, r runner, repoRoot string, opts Options) (*view, error) {
	if err := checkAbsoluteDirectory(repoRoot); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, failure(InvalidRepository, "metadata")
	}
	v := &view{root: root, worktree: repoRoot, objects: filepath.Join(repoRoot, ".git", "objects"), format: "sha1", options: opts}
	ok := false
	defer func() {
		if !ok {
			root.Close()
			if v.directory != "" {
				os.RemoveAll(v.directory)
			}
		}
	}()
	for _, name := range []string{".git", ".git/objects", ".git/refs"} {
		info, err := v.metadataInfo(name)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, failure(UnsupportedRepository, "layout")
		}
	}
	config, err := v.read(ctx, ".git/config", false)
	if err != nil {
		return nil, err
	}
	parsed, err := r.run(ctx, command{operation: "config", directory: "/", args: []string{"config", "--file", "-", "--no-includes", "--null", "--list"}, input: config, environment: gitEnvironment(false, nil)})
	if err != nil {
		return nil, err
	}
	if parsed.exit != 0 {
		return nil, failure(InvalidRepository, "config")
	}
	settings, err := v.settings(parsed.data)
	if err != nil {
		return nil, err
	}
	if err := v.scanObjects(ctx); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "homelab-git-inspect-")
	if err != nil {
		return nil, failure(CommandFailed, "scratch")
	}
	v.directory = directory
	for _, name := range []string{"refs", "objects", "info"} {
		if os.Mkdir(filepath.Join(directory, name), 0700) != nil {
			return nil, failure(CommandFailed, "scratch")
		}
	}
	minimal := "[core]\nrepositoryformatversion = 0\nbare = false\nlogallrefupdates = false\nfsmonitor = false\nignorestat = false\ncommitgraph = false\nmultipackindex = false\nhooksPath = /dev/null\nattributesFile = /dev/null\nexcludesFile = /dev/null\n" + settings
	if v.format == "sha256" {
		minimal = strings.Replace(minimal, "repositoryformatversion = 0", "repositoryformatversion = 1", 1) + "[extensions]\nobjectFormat = sha256\n"
	}
	if os.WriteFile(filepath.Join(directory, "config"), []byte(minimal), 0600) != nil {
		return nil, failure(CommandFailed, "scratch")
	}
	head, err := v.read(ctx, ".git/HEAD", false)
	if err != nil {
		return nil, err
	}
	headText := strings.TrimSuffix(string(head), "\n")
	if strings.HasPrefix(headText, "ref: ") {
		v.headRef = strings.TrimPrefix(headText, "ref: ")
		if !validBranchRef(v.headRef) {
			return nil, failure(InvalidRepository, "head")
		}
	} else if !validOID(headText, v.format) {
		return nil, failure(InvalidRepository, "head")
	}
	if os.WriteFile(filepath.Join(directory, "HEAD"), head, 0600) != nil {
		return nil, failure(CommandFailed, "scratch")
	}
	for _, name := range []string{"index", "packed-refs", "shallow", "info/exclude"} {
		data, err := v.read(ctx, ".git/"+name, true)
		if err != nil {
			return nil, err
		}
		if data != nil {
			if name == "shallow" {
				v.shallow = len(bytes.TrimSpace(data)) != 0
			}
			if os.WriteFile(filepath.Join(directory, name), data, 0600) != nil {
				return nil, failure(CommandFailed, "scratch")
			}
		}
	}
	if err := fs.WalkDir(root.FS(), ".git/refs", func(name string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return contextFailure(err, "metadata")
		}
		if walkErr != nil {
			return failure(InvalidRepository, "refs")
		}
		if err := v.countEntry(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return failure(UnsafePath, "refs")
		}
		relative := strings.TrimPrefix(name, ".git/")
		if entry.IsDir() {
			if relative == "refs" {
				return nil
			}
			if os.Mkdir(filepath.Join(directory, relative), 0700) != nil {
				return failure(CommandFailed, "scratch")
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return failure(UnsafePath, "refs")
		}
		data, err := v.read(ctx, name, false)
		if err != nil {
			return err
		}
		if os.WriteFile(filepath.Join(directory, relative), data, 0600) != nil {
			return failure(CommandFailed, "scratch")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	ok = true
	return v, nil
}
func (v *view) settings(data []byte) (string, error) {
	settings := "filemode = true\n"
	if len(data) > 0 && data[len(data)-1] != 0 {
		return "", failure(UnexpectedOutput, "config")
	}
	for _, record := range bytes.Split(bytes.TrimSuffix(data, []byte{0}), []byte{0}) {
		if len(record) == 0 {
			continue
		}
		pair := bytes.SplitN(record, []byte{'\n'}, 2)
		key := string(pair[0])
		value := "true"
		if len(pair) == 2 {
			value = string(pair[1])
		}
		switch key {
		case "core.repositoryformatversion":
			if value != "0" && value != "1" {
				return "", failure(UnsupportedRepository, "format")
			}
		case "core.bare":
			if value != "false" {
				return "", failure(UnsupportedRepository, "layout")
			}
		case "extensions.objectformat":
			if value != "sha1" && value != "sha256" {
				return "", failure(UnsupportedRepository, "format")
			}
			v.format = value
		case "extensions.partialclone":
			v.partial = true
		case "core.sparsecheckout", "core.sparsecheckoutcone", "core.splitindex":
			if value != "false" {
				return "", failure(UnsupportedRepository, "index_layout")
			}
		case "core.filemode", "core.ignorecase", "core.precomposeunicode":
			b, ok := gitBoolean(value)
			if !ok {
				return "", failure(InvalidRepository, "config")
			}
			settings += strings.TrimPrefix(key, "core.") + " = " + b + "\n"
		case "core.autocrlf":
			if value != "input" {
				b, ok := gitBoolean(value)
				if !ok {
					return "", failure(InvalidRepository, "config")
				}
				value = b
			}
			settings += "autocrlf = " + value + "\n"
		case "core.eol":
			if value != "lf" && value != "crlf" && value != "native" {
				return "", failure(InvalidRepository, "config")
			}
			settings += "eol = " + value + "\n"
		default:
			if strings.HasPrefix(key, "include.") || strings.HasPrefix(key, "includeif.") || strings.HasPrefix(key, "extensions.") || strings.HasPrefix(key, "filter.") {
				return "", failure(UnsupportedRepository, "config")
			}
			if strings.HasPrefix(key, "remote.") && strings.HasSuffix(key, ".promisor") && value == "true" {
				v.partial = true
			}
		}
	}
	return settings, nil
}
func gitBoolean(s string) (string, bool) {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "1":
		return "true", true
	case "false", "no", "off", "0", "":
		return "false", true
	}
	return "", false
}
func (v *view) countEntry() error {
	v.entries++
	if v.entries > v.options.MaxEntries {
		return failure(MetadataLimit, "metadata")
	}
	return nil
}
func (v *view) scanObjects(ctx context.Context) error {
	return fs.WalkDir(v.root.FS(), ".git/objects", func(name string, entry fs.DirEntry, err error) error {
		if e := ctx.Err(); e != nil {
			return contextFailure(e, "objects")
		}
		if err != nil {
			return failure(InvalidRepository, "objects")
		}
		if err := v.countEntry(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() && !entry.Type().IsRegular() {
			return failure(UnsafePath, "objects")
		}
		if name == ".git/objects/info/alternates" || name == ".git/objects/info/http-alternates" {
			return failure(UnsupportedRepository, "alternates")
		}
		if strings.HasSuffix(name, ".promisor") {
			v.partial = true
		}
		return nil
	})
}
func (v *view) metadataInfo(name string) (os.FileInfo, error) {
	current := "."
	var info os.FileInfo
	for _, part := range strings.Split(name, "/") {
		current = filepath.Join(current, part)
		var err error
		info, err = v.root.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, failure(MissingRepository, "metadata")
			}
			return nil, failure(InvalidRepository, "metadata")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, failure(UnsafePath, "metadata")
		}
	}
	return info, nil
}
func (v *view) read(ctx context.Context, name string, optional bool) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, contextFailure(err, "metadata")
	}
	info, err := v.metadataInfo(name)
	if err != nil {
		var e *Error
		if optional && errors.As(err, &e) && e.Code == MissingRepository {
			return nil, nil
		}
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, failure(UnsafePath, "metadata")
	}
	if info.Size() > int64(v.options.MaxMetadata-v.budget) {
		return nil, failure(MetadataLimit, "metadata")
	}
	f, err := v.root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, failure(InvalidRepository, "metadata")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, failure(UnsafePath, "metadata")
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(v.options.MaxMetadata-v.budget)+1))
	if err != nil {
		return nil, failure(InvalidRepository, "metadata")
	}
	v.budget += len(data)
	if v.budget > v.options.MaxMetadata {
		return nil, failure(MetadataLimit, "metadata")
	}
	return data, nil
}
func checkAbsoluteDirectory(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return failure(UnsafePath, "repository")
	}
	current := "/"
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return failure(MissingRepository, "repository")
			}
			return failure(InvalidRepository, "repository")
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return failure(UnsafePath, "repository")
		}
	}
	return nil
}
func (v *view) command(operation string, args ...string) command {
	env := append(gitEnvironment(false, nil), "GIT_DIR="+v.directory, "GIT_WORK_TREE="+v.worktree, "GIT_OBJECT_DIRECTORY="+v.objects, "GIT_INDEX_FILE="+filepath.Join(v.directory, "index"))
	return command{operation: operation, directory: v.worktree, args: args, environment: env}
}
