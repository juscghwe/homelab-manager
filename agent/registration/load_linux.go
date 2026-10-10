//go:build linux

package registration

import (
	"os"
	"path/filepath"
	"syscall"
)

// Load reads a local regular JSON file and returns a validated snapshot. An
// empty path selects DefaultPath. Alternate paths must be absolute. This local
// API is not a remote selector or a privileged registration override.
func Load(path string) (*Registration, error) {
	if path == "" {
		path = DefaultPath
	}
	if filepath.Ext(path) != ".json" {
		return nil, fail(InvalidPath, "config")
	}
	clean, info, err := absoluteMetadata(path, "config")
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fail(InvalidPath, "config")
	}
	parent, err := os.OpenRoot(filepath.Dir(clean))
	if err != nil {
		return nil, fail(ReadFailure, "config")
	}
	defer parent.Close()
	// Do not follow a substituted final symlink or block on a substituted FIFO.
	f, err := parent.OpenFile(filepath.Base(clean), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fail(ReadFailure, "config")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, fail(ReadFailure, "config")
	}
	cfg, err := Decode(f)
	if err != nil {
		return nil, err
	}
	registration, err := Validate(cfg)
	if err != nil {
		return nil, err
	}
	// Config-file warnings precede declaration warnings deterministically.
	registration.warnings = append(pathWarnings(clean, "config"), registration.warnings...)
	return registration, nil
}
