package gitinspect

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/juscghwe/homelab-manager/agent/registration"
)

// Remote subprocesses run outside the checkout: no local Git config, includes,
// URL rewrites, SSH commands or credential helpers can be loaded from it.
func (e *Engine) observeRemote(ctx context.Context, repo registration.Repository, format string) Remote {
	if repo.URL == "" {
		return Remote{State: RemoteNotConfigured}
	}
	if repo.Branch == "" {
		return Remote{State: RemoteUnsupported, Failure: UnsafeConfiguration}
	}
	ref := "refs/heads/" + repo.Branch
	if !validBranchRef(ref) {
		return Remote{State: RemoteUnsupported, Failure: UnsafeConfiguration}
	}
	o, err := e.runner.run(ctx, command{operation: "remote", directory: "/", args: []string{"-c", "http.followRedirects=false", "ls-remote", "--exit-code", "--refs", "--", repo.URL, ref}, environment: gitEnvironment(true, e.options.TrustedCredentialHelper)})
	if err != nil {
		var classified *Error
		code := RemoteFailure
		if errors.As(err, &classified) {
			code = classified.Code
		}
		return Remote{State: RemoteUnavailable, Failure: code}
	}
	if o.exit == 2 {
		return Remote{State: RemoteRefMissing, ObservedAt: time.Now().UTC()}
	}
	if o.exit != 0 {
		return Remote{State: RemoteUnavailable, Failure: RemoteFailure}
	}
	oid, err := parseRemote(o.data, ref, format)
	if err != nil {
		var classified *Error
		errors.As(err, &classified)
		return Remote{State: RemoteUnavailable, Failure: classified.Code}
	}
	return Remote{State: RemoteAvailable, OID: oid, ObservedAt: time.Now().UTC()}
}
func parseRemote(data []byte, ref, format string) (string, error) {
	text := string(data)
	if !strings.HasSuffix(text, "\n") || strings.Count(text, "\n") != 1 {
		return "", failure(UnexpectedOutput, "remote")
	}
	oid, name, ok := strings.Cut(strings.TrimSuffix(text, "\n"), "\t")
	if !ok || name != ref || !validOID(oid, format) {
		return "", failure(UnexpectedOutput, "remote")
	}
	return oid, nil
}
