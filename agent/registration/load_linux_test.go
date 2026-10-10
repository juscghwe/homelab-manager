//go:build linux

package registration

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestLoad(t *testing.T) {
	c := fixture(t)
	p := filepath.Join(t.TempDir(), "agent.json")
	write(t, p, string(encode(t, c)))
	r, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Host() != c.Host {
		t.Fatal("wrong host")
	}
}
func TestLoadRejectsUnsafeFiles(t *testing.T) {
	for _, kind := range []string{"relative", "missing", "directory", "symlink", "ancestor symlink", "fifo", "secret name", "secret directory", "oversize", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			base := t.TempDir()
			p := filepath.Join(base, "agent.json")
			code := InvalidPath
			switch kind {
			case "relative":
				p = "agent.json"
			case "missing":
				code = MissingPath
			case "directory":
				mkdir(t, p)
			case "symlink":
				target := filepath.Join(base, ".env")
				write(t, target, "SECRET")
				if err := os.Symlink(target, p); err != nil {
					t.Fatal(err)
				}
			case "ancestor symlink":
				outside := t.TempDir()
				write(t, filepath.Join(outside, "agent.json"), "SECRET")
				link := filepath.Join(base, "link")
				if err := os.Symlink(outside, link); err != nil {
					t.Fatal(err)
				}
				p = filepath.Join(link, "agent.json")
			case "fifo":
				if err := syscall.Mkfifo(p, 0600); err != nil {
					t.Fatal(err)
				}
			case "secret name":
				p = filepath.Join(base, ".env.json")
				write(t, p, "SECRET")
			case "secret directory":
				mkdir(t, filepath.Join(base, "secrets"))
				p = filepath.Join(base, "secrets", "agent.json")
				write(t, p, "SECRET")
			case "oversize":
				write(t, p, strings.Repeat(" ", MaxBytes+1))
				code = TooLarge
			case "malformed":
				write(t, p, `{"CANARY_SECRET":`)
				code = InvalidJSON
			}
			_, err := Load(p)
			wantError(t, err, code, "")
			if strings.Contains(err.Error(), base) || strings.Contains(err.Error(), "SECRET") {
				t.Fatal("loader leaked path/content")
			}
		})
	}
}
func TestInputsNeverRead(t *testing.T) {
	c := fixture(t)
	root := c.Repositories["repo-1"].Root
	// A FIFO would hang if treated as content; validation rejects its metadata.
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	c.Stacks[0].TrackedInputs = []string{"pipe"}
	_, err := Validate(c)
	wantError(t, err, InvalidPath, "stacks[0].tracked_inputs[0]")
	// Unreadable regular contents do not prevent metadata validation.
	p := filepath.Join(root, "opaque.json")
	write(t, p, "SYNTHETIC_SECRET")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	c.Stacks[0].TrackedInputs = []string{"opaque.json"}
	if _, err := Validate(c); err != nil {
		t.Fatal(err)
	}
}
