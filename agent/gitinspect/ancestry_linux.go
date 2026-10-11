//go:build linux

package gitinspect

import (
	"context"
	"strings"

	"github.com/juscghwe/homelab-manager/agent/registration"
)

// Compare classifies explicit full commit OIDs using local objects only. It
// neither contacts a remote nor treats ref reachability as object completeness.
func (e *Engine) Compare(parent context.Context, reg *registration.Registration, id, left, right string) (Comparison, error) {
	unknown := Comparison{Relation: Unknown}
	if e == nil || e.runner == nil || reg == nil {
		return unknown, failure(InvalidArgument, "registration")
	}
	if !reg.Capabilities().GitStatus {
		return unknown, failure(CapabilityDisabled, "registration")
	}
	repo, ok := reg.Repository(id)
	if !ok {
		return unknown, failure(UnknownRepository, "registration")
	}
	ctx, cancel := context.WithTimeout(parent, e.options.Timeout)
	defer cancel()
	if err := requireGit(ctx, e.runner); err != nil {
		return unknown, err
	}
	v, err := openView(ctx, e.runner, repo.Root, e.options)
	if err != nil {
		return unknown, err
	}
	defer v.close()
	return e.compare(ctx, v, left, right)
}
func (e *Engine) compare(ctx context.Context, v *view, left, right string) (Comparison, error) {
	unknown := Comparison{Relation: Unknown, Reason: MissingObjects}
	if !validOID(left, v.format) || !validOID(right, v.format) {
		return unknown, failure(InvalidArgument, "revision")
	}
	for _, oid := range []string{left, right} {
		o, err := e.runner.run(ctx, v.command("ancestry", "cat-file", "-e", oid+"^{commit}"))
		if err != nil {
			return unknown, err
		}
		if o.exit != 0 {
			return unknown, nil
		}
	}
	if left == right {
		return Comparison{Relation: Equal}, nil
	}
	if v.shallow {
		return Comparison{Relation: Unknown, Reason: ShallowHistory}, nil
	}
	// Verify the reachable object closure before interpreting negative ancestry
	// tests. Missing parents/trees/blobs cannot be mistaken for divergence.
	objects, err := e.runner.run(ctx, v.command("ancestry", "rev-list", "--objects", "--no-object-names", "--missing=print", left, right, "--"))
	if err != nil {
		return unknown, err
	}
	if objects.exit != 0 {
		return unknown, nil
	}
	if len(objects.data) == 0 || objects.data[len(objects.data)-1] != '\n' {
		return unknown, failure(UnexpectedOutput, "ancestry")
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(objects.data), "\n"), "\n") {
		missing := strings.HasPrefix(line, "?")
		oid := strings.TrimPrefix(line, "?")
		if !validOID(oid, v.format) {
			return unknown, failure(UnexpectedOutput, "ancestry")
		}
		if missing {
			return unknown, nil
		}
	}
	forward, err := e.runner.run(ctx, v.command("ancestry", "merge-base", "--is-ancestor", left, right))
	if err != nil {
		return unknown, err
	}
	if forward.exit == 0 {
		return Comparison{Relation: Behind}, nil
	}
	if forward.exit != 1 {
		return unknown, nil
	}
	reverse, err := e.runner.run(ctx, v.command("ancestry", "merge-base", "--is-ancestor", right, left))
	if err != nil {
		return unknown, err
	}
	if reverse.exit == 0 {
		return Comparison{Relation: Ahead}, nil
	}
	if reverse.exit != 1 {
		return unknown, nil
	}
	return Comparison{Relation: Diverged}, nil // Includes proven unrelated complete histories.
}
