//go:build linux

package gitinspect

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type command struct {
	operation, directory string
	args, environment    []string
	input                []byte
}
type output struct {
	data []byte
	exit int
}
type runner interface {
	run(context.Context, command) (output, error)
}
type gitRunner struct {
	path      string
	timeout   time.Duration
	maxOutput int
}

func (r gitRunner) run(parent context.Context, c command) (output, error) {
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return output{}, contextFailure(err, c.operation)
	}
	cmd := exec.CommandContext(ctx, r.path, c.args...)
	cmd.Dir = c.directory
	cmd.Env = c.environment
	cmd.Stdin = bytes.NewReader(c.input)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				return os.ErrProcessDone
			}
			return err
		}
		return nil
	}
	cmd.WaitDelay = 200 * time.Millisecond
	capture := &capture{maximum: r.maxOutput, cancel: cancel}
	cmd.Stdout = captureWriter{capture: capture, retain: true}
	cmd.Stderr = captureWriter{capture: capture} // Count and discard; never retain diagnostics.
	err := cmd.Run()
	if capture.exceeded {
		return output{}, failure(OutputLimit, c.operation)
	}
	if ctx.Err() != nil {
		return output{}, contextFailure(ctx.Err(), c.operation)
	}
	result := output{data: capture.stdout.Bytes()}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.exit = exit.ExitCode()
			return result, nil
		}
		return output{}, failure(GitUnavailable, c.operation)
	}
	return result, nil
}
func contextFailure(err error, operation string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return failure(Timeout, operation)
	}
	return failure(Canceled, operation)
}

type capture struct {
	mu             sync.Mutex
	stdout         bytes.Buffer
	maximum, count int
	exceeded       bool
	cancel         context.CancelFunc
}
type captureWriter struct {
	capture *capture
	retain  bool
}

func (w captureWriter) Write(p []byte) (int, error) {
	c := w.capture
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(p) > c.maximum-c.count {
		c.exceeded = true
		c.cancel()
		return 0, io.ErrShortWrite
	}
	c.count += len(p)
	if w.retain {
		return c.stdout.Write(p)
	}
	return len(p), nil
}

// Only trusted transport necessities survive. Incoming GIT_*, askpass, config,
// tracing, pager, executable paths and dynamic-loader injection are not copied.
func gitEnvironment(remote bool, helper []string) []string {
	env := []string{"PATH=/usr/bin:/bin", "HOME=/nonexistent", "XDG_CONFIG_HOME=/nonexistent", "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0", "GIT_ATTR_NOSYSTEM=1", "GIT_ALLOW_PROTOCOL=", "GIT_PROTOCOL_FROM_USER=0"}
	if !remote {
		return env
	}
	env = append(env, "GIT_ALLOW_PROTOCOL=https:ssh", "GIT_SSH_COMMAND=/usr/bin/ssh -F /dev/null -oBatchMode=yes -oStrictHostKeyChecking=yes", "GIT_SSH_VARIANT=ssh")
	for _, name := range []string{"HOME", "SSH_AUTH_SOCK", "SSL_CERT_FILE", "SSL_CERT_DIR", "GIT_SSL_CAINFO", "GIT_SSL_CAPATH", "HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "NO_PROXY", "https_proxy", "http_proxy", "all_proxy", "no_proxy"} {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	if len(helper) > 0 {
		parts := make([]string, len(helper))
		for i, arg := range helper {
			parts[i] = shellQuote(arg)
		}
		env = append(env, "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=credential.helper", "GIT_CONFIG_VALUE_0=!"+strings.Join(parts, " "))
	}
	return env
}

// Git's credential-helper ABI uses a shell. Quote only trusted constructor argv;
// no repository or remote data is interpolated into this command.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func normalizedOptions(o Options) (Options, error) {
	if o.GitPath == "" {
		o.GitPath = "/usr/bin/git"
	}
	if !filepath.IsAbs(o.GitPath) {
		return o, failure(InvalidArgument, "options")
	}
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.MaxOutput == 0 {
		o.MaxOutput = 4 << 20
	}
	if o.MaxMetadata == 0 {
		o.MaxMetadata = 8 << 20
	}
	if o.MaxEntries == 0 {
		o.MaxEntries = 50000
	}
	if o.Timeout < 0 || o.Timeout > time.Minute || o.MaxOutput < 1 || o.MaxOutput > 32<<20 || o.MaxMetadata < 1 || o.MaxMetadata > 64<<20 || o.MaxEntries < 1 || o.MaxEntries > 1000000 {
		return o, failure(InvalidArgument, "options")
	}
	if len(o.TrustedCredentialHelper) > 0 {
		if !filepath.IsAbs(o.TrustedCredentialHelper[0]) {
			return o, failure(InvalidArgument, "authentication")
		}
		for _, arg := range o.TrustedCredentialHelper {
			if strings.ContainsRune(arg, 0) {
				return o, failure(InvalidArgument, "authentication")
			}
		}
		o.TrustedCredentialHelper = append([]string(nil), o.TrustedCredentialHelper...)
	}
	return o, nil
}
func validOID(s string, format string) bool {
	n := 40
	if format == "sha256" {
		n = 64
	}
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return s != strings.Repeat("0", n)
}
func requireGit(ctx context.Context, r runner) error {
	o, err := r.run(ctx, command{operation: "version", directory: "/", args: []string{"--version"}, environment: gitEnvironment(false, nil)})
	if err != nil {
		return err
	}
	version := strings.TrimPrefix(strings.TrimSpace(string(o.data)), "git version ")
	parts := strings.Split(version, ".")
	if o.exit != 0 || len(parts) < 2 {
		return failure(GitUnavailable, "version")
	}
	major, e1 := strconv.Atoi(parts[0])
	minor, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || major < 2 || major == 2 && minor < 43 {
		return failure(UnsupportedRepository, "git_version")
	}
	return nil
}
