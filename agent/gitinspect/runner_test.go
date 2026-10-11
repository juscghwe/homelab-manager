package gitinspect

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testError(t *testing.T, err error, code Code) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
	if strings.Contains(err.Error(), "CANARY") {
		t.Fatal("raw diagnostics escaped")
	}
}
func script(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "program")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestRunnerBoundsAndRedaction(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			body := "yes CANARY"
			if stream == "stderr" {
				body += " >&2"
			}
			r := gitRunner{path: script(t, body), timeout: time.Second, maxOutput: 128}
			_, err := r.run(context.Background(), command{operation: "test", directory: "/", environment: gitEnvironment(false, nil)})
			testError(t, err, OutputLimit)
		})
	}
	r := gitRunner{path: script(t, "printf CANARY >&2; exit 17"), timeout: time.Second, maxOutput: 128}
	o, err := r.run(context.Background(), command{operation: "test", directory: "/", environment: gitEnvironment(false, nil)})
	if err != nil || o.exit != 17 || len(o.data) != 0 {
		t.Fatalf("diagnostics retained or exit lost: %+v %v", o, err)
	}
}
func TestRunnerCancellation(t *testing.T) {
	r := gitRunner{path: script(t, "sleep 60 & wait"), timeout: 40 * time.Millisecond, maxOutput: 128}
	start := time.Now()
	_, err := r.run(context.Background(), command{operation: "test", directory: "/", environment: gitEnvironment(false, nil)})
	testError(t, err, Timeout)
	if time.Since(start) > time.Second {
		t.Fatal("child kept pipe open")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = r.run(ctx, command{operation: "test"})
	testError(t, err, Canceled)
}
func TestEnvironmentPolicy(t *testing.T) {
	for _, name := range []string{"GIT_SSH_COMMAND", "GIT_ASKPASS", "GIT_TRACE", "GIT_CONFIG_COUNT", "LD_PRELOAD", "GIT_EXEC_PATH"} {
		t.Setenv(name, "CANARY")
	}
	t.Setenv("SSH_AUTH_SOCK", "/trusted/agent.sock")
	local := strings.Join(gitEnvironment(false, nil), "\n")
	if strings.Contains(local, "CANARY") || strings.Contains(local, "agent.sock") {
		t.Fatal("untrusted/local transport environment inherited")
	}
	remote := strings.Join(gitEnvironment(true, []string{"/trusted/helper", "a'b"}), "\n")
	if strings.Contains(remote, "CANARY") || !strings.Contains(remote, "agent.sock") || !strings.Contains(remote, "StrictHostKeyChecking=yes") || !strings.Contains(remote, "credential.helper") {
		t.Fatal("transport policy broken")
	}
	if shellQuote("a'b") != "'a'\\''b'" {
		t.Fatal("helper argv quoting broken")
	}
}
func TestRunnerMissingExecutable(t *testing.T) {
	r := gitRunner{path: "/not/present/git", timeout: time.Second, maxOutput: 128}
	_, err := r.run(context.Background(), command{operation: "test", directory: "/", environment: gitEnvironment(false, nil)})
	testError(t, err, GitUnavailable)
}
