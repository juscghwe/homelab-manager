package registration

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSuspiciousPathsAreWarnings(t *testing.T) {
	for _, name := range []string{"secret", "secrets", "data", "volume", "volumes", "SeCrEtS"} {
		t.Run(name, func(t *testing.T) {
			cfg := fixture(t)
			base := cfg.Repositories["repo-1"].Root
			root := filepath.Join(base, name)
			mkdir(t, filepath.Join(root, name))
			input := filepath.Join(root, name, "opaque.json")
			write(t, input, "SYNTHETIC_SECRET")
			if err := os.Chmod(input, 0); err != nil {
				t.Fatal(err)
			}
			cfg.Repositories["repo-1"] = Repository{Root: root}
			cfg.Stacks[0].Path = name
			cfg.Stacks[0].TrackedInputs = []string{name + "/opaque.json"}
			r, err := Validate(cfg)
			if err != nil {
				t.Fatal(err)
			}
			want := []Warning{
				{Code: SuspiciousPath, Field: "repositories[0].root"},
				{Code: SuspiciousPath, Field: "stacks[0].path"},
				{Code: SuspiciousPath, Field: "stacks[0].tracked_inputs[0]"},
			}
			if !reflect.DeepEqual(r.Warnings(), want) {
				t.Fatalf("warnings: %+v", r.Warnings())
			}
			output := fmt.Sprint(r.Warnings())
			if strings.Contains(output, root) || strings.Contains(output, "SYNTHETIC_SECRET") {
				t.Fatal("warning leaks configuration")
			}
			warnings := r.Warnings()
			warnings[0].Field = "changed"
			if !reflect.DeepEqual(r.Warnings(), want) {
				t.Fatal("warning snapshot was mutated")
			}
		})
	}
}

func TestSuspiciousWarningsDoNotWeakenBoundaries(t *testing.T) {
	for _, input := range []string{"secrets/../outside", "/data/injected", "secrets/.env", "data/key.pem", "volumes/.git/config", "secrets/.ssh/config", "secrets/link"} {
		t.Run(input, func(t *testing.T) {
			cfg := fixture(t)
			root := cfg.Repositories["repo-1"].Root
			mkdir(t, filepath.Join(root, "secrets"))
			// Point outside the registered root; even a suspicious name cannot
			// downgrade a real symlink boundary violation to a warning.
			if err := os.Symlink(t.TempDir(), filepath.Join(root, "secrets", "link")); err != nil {
				t.Fatal(err)
			}
			cfg.Stacks[0].TrackedInputs = []string{input}
			_, err := Validate(cfg)
			wantError(t, err, InvalidPath, "stacks[0].tracked_inputs[0]")
		})
	}
	cfg := fixture(t)
	data := strings.Replace(string(encode(t, cfg)), `"schema_version":1`, `"schema_version":1,"allow_suspicious_paths":true`, 1)
	_, err := Decode(strings.NewReader(data))
	wantError(t, err, UnknownField, "document") // No global bypass/consent field.
}

func TestWarningOrderIsDeterministic(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"data", "secrets"} {
		mkdir(t, filepath.Join(base, name))
	}
	want := []Warning{{Code: SuspiciousPath, Field: "repositories[0].root"}, {Code: SuspiciousPath, Field: "repositories[1].root"}}
	for n := 0; n < 20; n++ {
		repos := map[string]Repository{}
		ids := []string{"a", "z"}
		if n%2 == 0 {
			ids = []string{"z", "a"}
		}
		for _, id := range ids {
			name := "data"
			if id == "z" {
				name = "secrets"
			}
			repos[id] = Repository{Root: filepath.Join(base, name)}
		}
		r, err := Validate(Config{SchemaVersion: SchemaVersion, Host: "host", Repositories: repos, Stacks: []Stack{}})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(r.Warnings(), want) {
			t.Fatalf("unstable warnings: %+v", r.Warnings())
		}
	}
	cfg := fixture(t)
	r, err := Validate(cfg)
	if err != nil || len(r.Warnings()) != 0 {
		t.Fatalf("unexpected warning on ordinary paths: %v", err)
	}
	if got := pathWarnings("secret-service/metadata/volumes-backup", "test"); len(got) != 0 {
		t.Fatal("partial-name false positive")
	}
	if got := pathWarnings("secrets/data/volumes", "test"); len(got) != 1 {
		t.Fatal("duplicate warning for one field")
	}
}
