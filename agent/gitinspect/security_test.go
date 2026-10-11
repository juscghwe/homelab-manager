package gitinspect

import (
	"context"
	"fmt"
	"github.com/juscghwe/homelab-manager/agent/registration"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestUnsafeLayoutsAndSourcePaths(t *testing.T) {
	for _, kind := range []string{"config symlink", "objects symlink", "loose object symlink", "ref symlink", "worktree symlink", "alternates", "filter", "include", "sparse", "gitfile", "bad index"} {
		t.Run(kind, func(t *testing.T) {
			root := newRepository(t)
			oid := commitFixture(t, root, "file", "text")
			reg := registered(t, root)
			outside := t.TempDir()
			writeFixture(t, outside, "canary", "SYNTHETIC_SECRET")
			marker := filepath.Join(outside, "executed")
			helper := script(t, "printf CANARY > '"+marker+"'")
			code := UnsafePath
			symlink := func(name, target string) {
				t.Helper()
				if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "config symlink":
				symlink(".git/config", filepath.Join(outside, "canary"))
			case "objects symlink":
				symlink(".git/objects", outside)
			case "loose object symlink":
				symlink(".git/objects/"+oid[:2]+"/"+oid[2:], filepath.Join(outside, "canary"))
			case "ref symlink":
				symlink(".git/refs/heads/main", filepath.Join(outside, "canary"))
			case "worktree symlink":
				symlink("file", filepath.Join(outside, "canary"))
			case "alternates":
				writeFixture(t, root, ".git/objects/info/alternates", outside+"\n")
				code = UnsupportedRepository
			case "filter":
				fixtureGit(t, root, "config", "filter.evil.clean", helper)
				code = UnsupportedRepository
			case "include":
				fixtureGit(t, root, "config", "include.path", filepath.Join(outside, "canary"))
				code = UnsupportedRepository
			case "sparse":
				fixtureGit(t, root, "config", "core.sparseCheckout", "true")
				code = UnsupportedRepository
			case "gitfile":
				if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, ".git", "gitdir: "+outside+"\n")
				code = UnsupportedRepository
			case "bad index":
				writeFixture(t, root, ".git/index", "CANARY")
				code = IncompleteRepository
			}
			e, _ := New(Options{})
			result, err := e.Inspect(context.Background(), reg, "repo", Request{})
			testError(t, err, code)
			if result.DirtyAvailable {
				t.Fatal("unsafe inspection reported known clean state")
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("unapproved executable ran")
			}
		})
	}
}
func TestNoLocalHTTPOrLazyObjectAcquisition(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	root := newRepository(t)
	a := commitFixture(t, root, "file", "base")
	b := commitFixture(t, root, "file", "next")
	fixtureGit(t, root, "config", "remote.origin.url", server.URL+"/repo.git")
	fixtureGit(t, root, "config", "remote.origin.promisor", "true")
	fixtureGit(t, root, "config", "extensions.partialClone", "origin")
	if err := os.Remove(filepath.Join(root, ".git", "objects", a[:2], a[2:])); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	e, _ := New(Options{})
	reg := registered(t, root)
	local, err := e.Inspect(context.Background(), reg, "repo", Request{})
	if err != nil || local.Remote.State != RemoteNotRequested || !local.Evidence.Partial {
		t.Fatalf("local facts: %+v %v", local, err)
	}
	rel, err := e.Compare(context.Background(), reg, "repo", b, a)
	if err != nil || rel.Relation != Unknown {
		t.Fatalf("incomplete relation: %+v %v", rel, err)
	}
	if requests.Load() != 0 {
		t.Fatal("local-only inspection contacted HTTP promisor")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("objects/refs/index changed or missing object acquired")
	}
}
func TestSHA256AndUnusualTrackedNames(t *testing.T) {
	root := t.TempDir()
	fixtureGit(t, root, "init", "--object-format=sha256", "--initial-branch=main")
	for _, name := range []string{"line\nname", "tab\tname", "-option", string([]byte{0xff})} {
		commitFixture(t, root, name, "non-secret")
	}
	r := inspectFixture(t, root)
	if len(r.Head.OID) != 64 || r.Evidence.ObjectFormat != "sha256" || r.Dirty.IsDirty() {
		t.Fatalf("SHA256: %+v", r)
	}
	writeFixture(t, root, "line\nname", "changed")
	r = inspectFixture(t, root)
	if r.Dirty.Unstaged != 1 {
		t.Fatal("tracked unusual name lost")
	}
}
func TestConcurrentInspectionPreservesSourceAndState(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, ".gitignore", "state/\n")
	commitFixture(t, root, "file", "text")
	writeFixture(t, root, "state/last-successful.json", "SYNTHETIC_STATE")
	before := snapshot(t, root)
	e, _ := New(Options{})
	reg := registered(t, root)
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := e.Inspect(context.Background(), reg, "repo", Request{})
			if err != nil || !result.DirtyAvailable || result.Dirty.IsDirty() {
				t.Errorf("concurrent: %+v %v", result, err)
			}
		}()
	}
	wg.Wait()
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("concurrent inspection modified index/refs/state/contents")
	}
}
func TestRealStatusOutputLimitAndCancellation(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, "file", "text")
	for n := 0; n < 100; n++ {
		writeFixture(t, root, fmt.Sprintf("untracked-%03d-%s", n, strings.Repeat("x", 30)), "synthetic")
	}
	before := snapshot(t, root)
	e, _ := New(Options{MaxOutput: 512})
	result, err := e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	testError(t, err, OutputLimit)
	if result.DirtyAvailable {
		t.Fatal("truncated output interpreted as clean")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("limit failure changed source")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = e.Inspect(ctx, registered(t, root), "repo", Request{})
	testError(t, err, Canceled)
}
func TestMissingHeadTreeIsUnavailable(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, "file", "text")
	tree := fixtureGit(t, root, "rev-parse", "HEAD^{tree}")
	if err := os.Remove(filepath.Join(root, ".git", "objects", tree[:2], tree[2:])); err != nil {
		t.Fatal(err)
	}
	e, _ := New(Options{})
	result, err := e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	testError(t, err, IncompleteRepository)
	if result.DirtyAvailable {
		t.Fatal("missing tree treated as clean")
	}
}
func TestStatusParserValidatesModesAndObjectFields(t *testing.T) {
	oid := strings.Repeat("a", 40)
	valid := "1 M. N... 100644 100644 100644 " + oid + " " + oid + " file\x00"
	if result, err := parseStatus([]byte(valid), "sha1"); err != nil || result.Staged != 1 {
		t.Fatal("valid status rejected")
	}
	for _, bad := range []string{strings.Replace(valid, "100644", "CANARY", 1), strings.Replace(valid, oid, "CANARY", 1), strings.Replace(valid, "N...", "S...", 1), strings.Replace(valid, "M.", "Q.", 1)} {
		_, err := parseStatus([]byte(bad), "sha1")
		testError(t, err, UnexpectedOutput)
	}
}
func TestTrustedHelperArgvDoesNotBecomeShellCode(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "args")
	injection := filepath.Join(t.TempDir(), "injected")
	helper := script(t, "printf '%s\n' \"$@\" > '"+marker+"'; printf 'username=fixture\npassword=SYNTHETIC_TOKEN\n'")
	arg := "a'b; touch " + injection
	r := gitRunner{path: "/usr/bin/git", timeout: 10e9, maxOutput: 4096}
	out, err := r.run(context.Background(), command{operation: "authentication_test", directory: "/", args: []string{"credential", "fill"}, environment: gitEnvironment(true, []string{helper, arg}), input: []byte("protocol=https\nhost=example.invalid\n\n")})
	if err != nil || out.exit != 0 {
		t.Fatalf("trusted helper failed: %v", err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != arg+"\nget\n" {
		t.Fatalf("helper argv changed: %v", err)
	}
	if _, err := os.Stat(injection); !os.IsNotExist(err) {
		t.Fatal("trusted helper arg became shell syntax")
	}
}

func TestIndexFlagsCannotConcealDirtyFiles(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			root := newRepository(t)
			commitFixture(t, root, "file", "original")
			fixtureGit(t, root, "update-index", flag, "--", "file") // Fixture setup only.
			writeFixture(t, root, "file", "changed")
			before := snapshot(t, root)
			e, _ := New(Options{})
			result, err := e.Inspect(context.Background(), registered(t, root), "repo", Request{})
			testError(t, err, UnsupportedRepository)
			if result.DirtyAvailable || !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("hidden dirty state or index flag mutation")
			}
		})
	}
}
func TestLocalAttributesPreserveStatusSemantics(t *testing.T) {
	root := newRepository(t)
	writeFixture(t, root, ".git/info/attributes", "file text eol=crlf\n")
	commitFixture(t, root, "file", "line\r\n")
	before := snapshot(t, root)
	if result := inspectFixture(t, root); result.Dirty.IsDirty() {
		t.Fatal("ignored local attributes produced false dirty state")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("attributes inspection modified source")
	}
}
func TestUnmergedFixture(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, "file", "base")
	fixtureGit(t, root, "checkout", "-b", "other")
	commitFixture(t, root, "file", "other")
	fixtureGit(t, root, "checkout", "main")
	commitFixture(t, root, "file", "main")
	cmd := exec.Command("/usr/bin/git", "-c", "core.hooksPath=/dev/null", "merge", "other")
	cmd.Dir = root
	cmd.Env = gitEnvironment(false, nil)
	if err := cmd.Run(); err == nil {
		t.Fatal("fixture merge did not conflict")
	}
	before := snapshot(t, root)
	result := inspectFixture(t, root)
	if result.Dirty.Conflicted != 1 || !result.Dirty.IsDirty() {
		t.Fatalf("conflict lost: %+v", result)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("conflict inspection modified source")
	}
}
func TestDisabledCapabilityAndTrackedSymlink(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, "file", "text")
	reg, err := registration.Validate(registration.Config{SchemaVersion: 1, Host: "fixture", Repositories: map[string]registration.Repository{"repo": {Root: root}}, Stacks: []registration.Stack{}})
	if err != nil {
		t.Fatal(err)
	}
	e, _ := New(Options{})
	_, err = e.Inspect(context.Background(), reg, "repo", Request{})
	testError(t, err, CapabilityDisabled)
	if err := os.Symlink("file", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, root, "add", "--", "link")
	fixtureGit(t, root, "commit", "-m", "link fixture")
	_, err = e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	testError(t, err, UnsupportedRepository)
}
