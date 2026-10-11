package gitinspect

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/juscghwe/homelab-manager/agent/registration"
)

type remoteFixtureRunner struct {
	local    runner
	calls    int
	response output
	err      error
	last     command
}

func (r *remoteFixtureRunner) run(ctx context.Context, c command) (output, error) {
	if c.operation != "remote" {
		return r.local.run(ctx, c)
	}
	r.calls++
	r.last = c
	return r.response, r.err
}
func registrationRemote(t *testing.T, root, url, branch string) *registration.Registration {
	t.Helper()
	reg, err := registration.Validate(registration.Config{SchemaVersion: 1, Host: "fixture", Repositories: map[string]registration.Repository{"repo": {Root: root, URL: url, Branch: branch}}, Stacks: []registration.Stack{}, Capabilities: registration.Capabilities{GitStatus: true}})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}
func TestRemoteStatesAndStrictInvocation(t *testing.T) {
	root := newRepository(t)
	head := commitFixture(t, root, "file", "text")
	for _, tc := range []struct {
		name, url, branch string
		response          output
		err               error
		state             RemoteState
		code              Code
		calls             int
	}{
		{"absent", "", "", output{}, nil, RemoteNotConfigured, "", 0},
		{"branch absent", "https://example.invalid/repo.git", "", output{}, nil, RemoteUnsupported, UnsafeConfiguration, 0},
		{"available", "https://example.invalid/repo.git", "main", output{data: []byte(head + "\trefs/heads/main\n")}, nil, RemoteAvailable, "", 1},
		{"missing", "https://example.invalid/repo.git", "main", output{exit: 2}, nil, RemoteRefMissing, "", 1},
		{"unavailable", "https://example.invalid/repo.git", "main", output{exit: 128}, nil, RemoteUnavailable, RemoteFailure, 1},
		{"timeout", "https://example.invalid/repo.git", "main", output{}, failure(Timeout, "remote"), RemoteUnavailable, Timeout, 1},
		{"canceled", "https://example.invalid/repo.git", "main", output{}, failure(Canceled, "remote"), RemoteUnavailable, Canceled, 1},
		{"malformed", "https://example.invalid/repo.git", "main", output{data: []byte("CANARY")}, nil, RemoteUnavailable, UnexpectedOutput, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := New(Options{})
			mock := &remoteFixtureRunner{local: e.runner, response: tc.response, err: tc.err}
			e.runner = mock
			reg := registrationRemote(t, root, tc.url, tc.branch)
			before := snapshot(t, root)
			result, err := e.Inspect(context.Background(), reg, "repo", Request{Remote: true})
			if err != nil || !result.DirtyAvailable || result.Remote.State != tc.state || result.Remote.Failure != tc.code || mock.calls != tc.calls {
				t.Fatalf("result %+v error %v calls %d", result, err, mock.calls)
			}
			if tc.state == RemoteAvailable && (result.Relation != Equal || result.Remote.OID != head) {
				t.Fatal("remote observation/equality lost")
			}
			if tc.calls > 0 {
				want := []string{"-c", "http.followRedirects=false", "ls-remote", "--exit-code", "--refs", "--", tc.url, "refs/heads/main"}
				if mock.last.directory != "/" || !reflect.DeepEqual(mock.last.args, want) {
					t.Fatal("uncontrolled remote invocation")
				}
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("remote inspection changed source refs/index/state")
			}
		})
	}
	e, _ := New(Options{})
	mock := &remoteFixtureRunner{local: e.runner}
	e.runner = mock
	_, err := e.Inspect(context.Background(), registrationRemote(t, root, "https://example.invalid/repo.git", "main"), "repo", Request{})
	if err != nil || mock.calls != 0 {
		t.Fatal("local request contacted remote")
	}
}
func TestRemoteParserRejectsExtraOrUnexpectedRefs(t *testing.T) {
	oid := strings.Repeat("a", 40)
	for _, data := range []string{"", oid + "\tHEAD\n", oid + "\trefs/heads/main", oid + "\trefs/heads/main\n" + oid + "\trefs/heads/main\n", strings.Repeat("b", 64) + "\trefs/heads/main\n", oid + "\trefs/heads/main\nCANARY"} {
		_, err := parseRemote([]byte(data), "refs/heads/main", "sha1")
		testError(t, err, UnexpectedOutput)
	}
}
func TestRemoteUsesFreshAdvertisedOIDNotCachedRef(t *testing.T) {
	root := newRepository(t)
	old := commitFixture(t, root, "file", "old")
	head := commitFixture(t, root, "file", "new")
	fixtureGit(t, root, "update-ref", "refs/remotes/origin/main", head)
	e, _ := New(Options{})
	e.runner = &remoteFixtureRunner{local: e.runner, response: output{data: []byte(old + "\trefs/heads/main\n")}}
	result, err := e.Inspect(context.Background(), registrationRemote(t, root, "https://example.invalid/repo.git", "main"), "repo", Request{Remote: true})
	if err != nil || result.Remote.OID != old || result.Relation != Ahead {
		t.Fatalf("cached ref mistaken for fresh observation: %+v %v", result, err)
	}
}
