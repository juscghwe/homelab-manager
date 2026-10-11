package gitinspect

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRevisionRelationships(t *testing.T) {
	root := newRepository(t)
	a := commitFixture(t, root, "file", "base")
	b := commitFixture(t, root, "file", "left")
	fixtureGit(t, root, "checkout", "-b", "other", a)
	c := commitFixture(t, root, "file", "right")
	fixtureGit(t, root, "checkout", "--orphan", "unrelated")
	d := commitFixture(t, root, "file", "other history")
	e, _ := New(Options{})
	reg := registered(t, root)
	for _, tc := range []struct {
		left, right string
		relation    Relation
	}{{a, a, Equal}, {a, b, Behind}, {b, a, Ahead}, {b, c, Diverged}, {b, d, Diverged}, {b, strings.Repeat("a", 40), Unknown}} {
		result, err := e.Compare(context.Background(), reg, "repo", tc.left, tc.right)
		if err != nil || result.Relation != tc.relation {
			t.Fatalf("want %s, got %+v %v", tc.relation, result, err)
		}
	}
	_, err := e.Compare(context.Background(), reg, "repo", "--help", a)
	testError(t, err, InvalidArgument)
}
func TestShallowAndMissingObjectsStayUnknown(t *testing.T) {
	root := newRepository(t)
	a := commitFixture(t, root, "file", "base")
	b := commitFixture(t, root, "file", "next")
	e, _ := New(Options{})
	reg := registered(t, root)
	writeFixture(t, root, ".git/shallow", b+"\n")
	result, err := e.Compare(context.Background(), reg, "repo", b, a)
	if err != nil || result.Relation != Unknown || result.Reason != ShallowHistory {
		t.Fatalf("shallow: %+v %v", result, err)
	}
	if result, err = e.Compare(context.Background(), reg, "repo", b, b); err != nil || result.Relation != Equal {
		t.Fatal("shallow equality needs no ancestry")
	}
	if err := os.Remove(filepath.Join(root, ".git", "shallow")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".git", "objects", a[:2], a[2:])); err != nil {
		t.Fatal(err)
	}
	result, err = e.Compare(context.Background(), reg, "repo", b, a)
	if err != nil || result.Relation != Unknown || result.Reason != MissingObjects {
		t.Fatalf("missing: %+v %v", result, err)
	}
}
func TestPartialCloneCannotAcquireMissingObjects(t *testing.T) {
	root := newRepository(t)
	a := commitFixture(t, root, "file", "base")
	b := commitFixture(t, root, "file", "next")
	fixtureGit(t, root, "config", "extensions.partialClone", "origin")
	fixtureGit(t, root, "config", "remote.origin.promisor", "true")
	marker := filepath.Join(t.TempDir(), "executed")
	helper := script(t, "printf CANARY > '"+marker+"'")
	fixtureGit(t, root, "config", "remote.origin.url", "ext::"+helper)
	if err := os.Remove(filepath.Join(root, ".git", "objects", a[:2], a[2:])); err != nil {
		t.Fatal(err)
	}
	e, _ := New(Options{})
	result, err := e.Compare(context.Background(), registered(t, root), "repo", b, a)
	if err != nil || result.Relation != Unknown {
		t.Fatalf("partial: %+v %v", result, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("lazy fetch ran")
	}
}
