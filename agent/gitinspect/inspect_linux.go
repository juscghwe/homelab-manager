//go:build linux

package gitinspect

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juscghwe/homelab-manager/agent/registration"
)

type Engine struct {
	options Options
	runner  runner
}

// New accepts only trusted executable/authentication/limit settings.
func New(options Options) (*Engine, error) {
	o, err := normalizedOptions(options)
	if err != nil {
		return nil, err
	}
	return &Engine{options: o, runner: gitRunner{path: o.GitPath, timeout: o.Timeout, maxOutput: o.MaxOutput}}, nil
}

// Inspect resolves a registered ID, revalidates its root and observes facts.
// On failure, a partial Result may contain independently available observations.
func (e *Engine) Inspect(parent context.Context, reg *registration.Registration, id string, request Request) (Result, error) {
	result := Result{Head: Head{State: HeadUnavailable}, Remote: Remote{State: RemoteNotRequested}, Relation: NotApplicable}
	if request.Remote {
		result.Remote.State = RemoteNotObserved
		result.Relation = Unknown
	}
	if reg == nil {
		return result, failure(InvalidArgument, "registration")
	}
	if !reg.Capabilities().GitStatus {
		return result, failure(CapabilityDisabled, "registration")
	}
	repo, ok := reg.Repository(id)
	if !ok {
		result.RepositoryID = ""
		return result, failure(UnknownRepository, "registration")
	}
	result.RepositoryID = id
	ctx, cancel := context.WithTimeout(parent, e.options.Timeout)
	defer cancel()
	if err := requireGit(ctx, e.runner); err != nil {
		return result, err
	}
	v, err := openView(ctx, e.runner, repo.Root, e.options)
	if err != nil {
		return result, err
	}
	defer v.close()
	result.Evidence = Evidence{Shallow: v.shallow, Partial: v.partial, ObjectFormat: v.format}
	if v.headRef != "" {
		result.Head.Branch = strings.TrimPrefix(v.headRef, "refs/heads/")
	} else {
		result.Head.Detached = true
	}
	head, err := e.runner.run(ctx, v.command("head", "rev-parse", "--verify", "HEAD^{commit}"))
	if err != nil {
		return result, err
	}
	if head.exit != 0 {
		if v.headRef == "" {
			return result, failure(IncompleteRepository, "head")
		}
		ref, err := e.runner.run(ctx, v.command("head", "show-ref", "--verify", "--quiet", v.headRef))
		if err != nil {
			return result, err
		}
		if ref.exit != 1 {
			return result, failure(IncompleteRepository, "head")
		}
		result.Head.State = HeadUnborn
	} else {
		oid := strings.TrimSuffix(string(head.data), "\n")
		if !validOID(oid, v.format) {
			return result, failure(UnexpectedOutput, "head")
		}
		result.Head = Head{State: HeadAvailable, OID: oid, Branch: result.Head.Branch, Detached: result.Head.Detached}
	}
	if err := e.screenPaths(ctx, v, result.Head.State == HeadAvailable); err != nil {
		return result, err
	}
	status, err := e.runner.run(ctx, v.command("status", "status", "--porcelain=v2", "-z", "--untracked-files=all", "--ignore-submodules=all", "--no-renames"))
	if err != nil {
		return result, err
	}
	if status.exit != 0 {
		return result, failure(IncompleteRepository, "status")
	}
	result.Dirty, err = parseStatus(status.data)
	if err != nil {
		return result, err
	}
	result.DirtyAvailable = true
	result.ObservedAt = time.Now().UTC()
	if request.Remote {
		return result, failure(UnsupportedRepository, "remote_pending")
	}
	return result, nil
}
func (e *Engine) screenPaths(ctx context.Context, v *view, hasHead bool) error {
	index, err := e.runner.run(ctx, v.command("paths", "ls-files", "--stage", "--sparse", "-z"))
	if err != nil {
		return err
	}
	if index.exit != 0 {
		return failure(IncompleteRepository, "index")
	}
	names, err := parsePaths(index.data, false, v.format)
	if err != nil {
		return err
	}
	if hasHead {
		tree, err := e.runner.run(ctx, v.command("paths", "ls-tree", "--full-tree", "-r", "-z", "HEAD"))
		if err != nil {
			return err
		}
		if tree.exit != 0 {
			return failure(IncompleteRepository, "tree")
		}
		headNames, err := parsePaths(tree.data, true, v.format)
		if err != nil {
			return err
		}
		names = append(names, headNames...)
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return contextFailure(err, "tracked_paths")
		}
		current := "."
		for _, part := range strings.Split(name, "/") {
			current = filepath.Join(current, part)
			info, err := v.root.Lstat(current)
			if errors.Is(err, os.ErrNotExist) {
				break
			} // A tracked deletion does not read any content.
			if err != nil {
				return failure(UnsafePath, "tracked_paths")
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
				return failure(UnsafePath, "tracked_paths")
			}
		}
	}
	return nil
}
