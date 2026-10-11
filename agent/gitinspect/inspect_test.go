package gitinspect

import (
	"context"
	"crypto/sha256"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/juscghwe/homelab-manager/agent/registration"
)

func fixtureGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	base := []string{"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "core.hooksPath=/dev/null"}
	cmd := exec.Command("/usr/bin/git", append(base, args...)...)
	cmd.Dir = root
	cmd.Env = append(gitEnvironment(false, nil), "GIT_ALLOW_PROTOCOL=file") // Fixture setup only, never engine policy.
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture Git %v failed: %v %s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}
func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func newRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	fixtureGit(t, root, "init", "--initial-branch=main")
	return root
}
func commitFixture(t *testing.T, root, name, content string) string {
	t.Helper()
	writeFixture(t, root, name, content)
	fixtureGit(t, root, "add", "--", name)
	fixtureGit(t, root, "commit", "-m", "fixture")
	return fixtureGit(t, root, "rev-parse", "HEAD")
}
func registered(t *testing.T, root string) *registration.Registration {
	t.Helper()
	r, err := registration.Validate(registration.Config{SchemaVersion: 1, Host: "fixture", Repositories: map[string]registration.Repository{"repo": {Root: root}}, Stacks: []registration.Stack{}, Capabilities: registration.Capabilities{GitStatus: true}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func inspectFixture(t *testing.T, root string) Result {
	t.Helper()
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func snapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out[rel] = sha256.Sum256(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestLocalStatesAndPreservation(t *testing.T) {
	root := newRepository(t)
	oid := commitFixture(t, root, "tracked.txt", "first")
	commitFixture(t, root, "other.txt", "second")
	commitFixture(t, root, ".gitignore", "ignored\n.env\n")
	head := fixtureGit(t, root, "rev-parse", "HEAD")
	writeFixture(t, root, ".env", "SYNTHETIC_SECRET")
	writeFixture(t, root, "ignored", "SYNTHETIC_SECRET")
	writeFixture(t, root, "tracked.txt", "staged")
	fixtureGit(t, root, "add", "--", "tracked.txt")
	writeFixture(t, root, "tracked.txt", "unstaged again")
	writeFixture(t, root, "other.txt", "changed")
	for _, name := range []string{"space name", "line\nname", "tab\tname", "-option", string([]byte{0xff})} {
		writeFixture(t, root, name, "untracked")
	}
	before := snapshot(t, root)
	result := inspectFixture(t, root)
	if result.Head.State != HeadAvailable || result.Head.OID != head || result.Head.Branch != "main" || result.Head.Detached || result.Head.OID == oid {
		t.Fatalf("head: %+v", result.Head)
	}
	if result.Dirty != (Dirty{Staged: 1, Unstaged: 2, Untracked: 5}) || !result.Dirty.IsDirty() {
		t.Fatalf("dirty: %+v", result.Dirty)
	}
	if result.Remote.State != RemoteNotRequested || result.Relation != NotApplicable {
		t.Fatal("local request implied remote observation")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("source index/refs/config/worktree/ignored files changed")
	}
}
func TestHeadStates(t *testing.T) {
	root := newRepository(t)
	result := inspectFixture(t, root)
	if result.Head.State != HeadUnborn || result.Head.Branch != "main" || result.Head.OID != "" || result.Dirty.IsDirty() {
		t.Fatalf("unborn: %+v", result)
	}
	writeFixture(t, root, "first.txt", "first")
	fixtureGit(t, root, "add", "--", "first.txt")
	if result := inspectFixture(t, root); result.Head.State != HeadUnborn || result.Dirty.Staged != 1 {
		t.Fatalf("staged unborn: %+v", result)
	}
	fixtureGit(t, root, "commit", "-m", "first")
	head := fixtureGit(t, root, "rev-parse", "HEAD")
	fixtureGit(t, root, "checkout", "--detach", head)
	result = inspectFixture(t, root)
	if !result.Head.Detached || result.Head.Branch != "" || result.Head.OID != head || result.Dirty.IsDirty() {
		t.Fatalf("detached: %+v", result)
	}
}
func TestProtectedPathsBeforeStatus(t *testing.T) {
	for _, name := range []string{".env", "nested/.env.production", "key.pem"} {
		t.Run(name, func(t *testing.T) {
			root := newRepository(t)
			commitFixture(t, root, name, "SYNTHETIC_SECRET")
			e, _ := New(Options{})
			_, err := e.Inspect(context.Background(), registered(t, root), "repo", Request{})
			testError(t, err, UnsafePath)
		})
	}
	root := newRepository(t)
	commitFixture(t, root, "secrets/service.txt", "public configuration")
	if r := inspectFixture(t, root); r.Dirty.IsDirty() {
		t.Fatal("suspicious name mistaken for actual secret")
	}
}
func TestInvalidRegistrationAndRepositories(t *testing.T) {
	e, _ := New(Options{})
	root := newRepository(t)
	r := registered(t, root)
	_, err := e.Inspect(context.Background(), nil, "repo", Request{})
	testError(t, err, InvalidArgument)
	_, err = e.Inspect(context.Background(), r, "--upload-pack=CANARY", Request{})
	testError(t, err, UnknownRepository)
	writeFixture(t, root, ".git/HEAD", "invalid CANARY")
	_, err = e.Inspect(context.Background(), r, "repo", Request{})
	testError(t, err, InvalidRepository)
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	_, err = e.Inspect(context.Background(), r, "repo", Request{})
	testError(t, err, MissingRepository)
}
func TestStatusAndPathParserRejections(t *testing.T) {
	for _, data := range [][]byte{[]byte("? name"), []byte("2 renamed\x00"), []byte("1 XY\x00"), []byte("? ../escape\x00"), []byte("# header\x00"), []byte("\x00")} {
		_, err := parseStatus(data)
		testError(t, err, UnexpectedOutput)
	}
	for _, data := range [][]byte{[]byte("bad"), []byte("100644 invalid 0\tname\x00"), []byte("100644 " + strings.Repeat("a", 40) + " 0\t.env\x00")} {
		_, err := parsePaths(data, false, "sha1")
		if err == nil {
			t.Fatal("invalid path output accepted")
		}
	}
	if _, err := parseStatus([]byte("? odd\nname\x00")); err != nil {
		t.Fatal(err)
	}
}
func TestMaliciousConfigurationIsNotExecuted(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, "plain.txt", "text")
	marker := filepath.Join(t.TempDir(), "executed")
	malicious := script(t, "printf CANARY > '"+marker+"'")
	for _, key := range []string{"core.fsmonitor", "core.hooksPath", "core.sshCommand", "diff.external", "diff.test.textconv", "credential.helper"} {
		fixtureGit(t, root, "config", key, malicious)
	}
	fixtureGit(t, root, "config", "remote.origin.url", "ext::"+malicious)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.fsmonitor")
	t.Setenv("GIT_CONFIG_VALUE_0", malicious)
	t.Setenv("GIT_TRACE", marker)
	before := snapshot(t, root)
	result := inspectFixture(t, root)
	if result.Dirty.IsDirty() {
		t.Fatal("unexpected dirty result")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("repository/injected program or trace ran")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("inspection changed source")
	}
}
func TestMetadataAndLocalOutputLimits(t *testing.T) {
	root := newRepository(t)
	commitFixture(t, root, "plain.txt", "text")
	e, _ := New(Options{MaxMetadata: 1})
	_, err := e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	testError(t, err, MetadataLimit)
	e, _ = New(Options{MaxEntries: 1})
	_, err = e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	testError(t, err, MetadataLimit)
	e, _ = New(Options{MaxOutput: 2})
	_, err = e.Inspect(context.Background(), registered(t, root), "repo", Request{})
	testError(t, err, OutputLimit)
}
