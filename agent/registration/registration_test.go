package registration

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func fixture(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "compose", "web"))
	mkdir(t, filepath.Join(root, "shared"))
	write(t, filepath.Join(root, "compose", "web", "compose.yaml"), "services: {}")
	write(t, filepath.Join(root, "shared", "defaults.json"), "{}")
	return Config{SchemaVersion: 1, Host: "host-1", Repositories: map[string]Repository{"repo-1": {Root: root}}, Stacks: []Stack{{ID: "web", Repository: "repo-1", Path: "compose/web", ComposeProject: "web", TrackedInputs: []string{"compose/web", "shared/defaults.json"}}}, Capabilities: Capabilities{GitStatus: true}}
}
func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0700); err != nil {
		t.Fatal(err)
	}
}
func write(t *testing.T, p, value string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}
func encode(t *testing.T, c Config) []byte {
	t.Helper()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func wantError(t *testing.T, err error, code Code, field string) {
	t.Helper()
	var got *Error
	if !errors.As(err, &got) {
		t.Fatalf("expected typed %s, got %v", code, err)
	}
	if got.Code != code || (field != "" && got.Field != field) {
		t.Fatalf("got %v, want %s at %s", err, code, field)
	}
}

func TestValidRegistrationAndCopies(t *testing.T) {
	c := fixture(t)
	c.Stacks[0].Path = "./compose//web/"
	c.Stacks[0].TrackedInputs = []string{"shared/./defaults.json", "./compose/web"}
	r, err := Validate(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Host() != c.Host || !r.Capabilities().GitStatus || r.Capabilities().DockerDeploy {
		t.Fatal("wrong host or defaults")
	}
	if !reflect.DeepEqual(r.RepositoryIDs(), []string{"repo-1"}) || !reflect.DeepEqual(r.StackIDs(), []string{"web"}) {
		t.Fatal("wrong IDs")
	}
	s, ok := r.Stack("web")
	if !ok || s.Path != "compose/web" || !reflect.DeepEqual(s.TrackedInputs, []string{"compose/web", "shared/defaults.json"}) {
		t.Fatal("not normalized")
	}
	c.Stacks[0].TrackedInputs[0] = "changed"
	c.Repositories["repo-1"] = Repository{}
	s.TrackedInputs[0] = "changed"
	copy, _ := r.Stack("web")
	repo, _ := r.Repository("repo-1")
	if copy.TrackedInputs[0] != "compose/web" || repo.Root == "" {
		t.Fatal("validated snapshot was mutated")
	}
	if _, ok := r.Stack("missing"); ok {
		t.Fatal("unknown stack exists")
	}
	if _, ok := r.Repository("missing"); ok {
		t.Fatal("unknown repo exists")
	}
}

func TestMultipleRepositoriesStacksAndSharedInputs(t *testing.T) {
	c := fixture(t)
	root := c.Repositories["repo-1"].Root
	mkdir(t, filepath.Join(root, "compose", "api"))
	c.Stacks = append(c.Stacks, Stack{ID: "api", Repository: "repo-1", Path: "compose/api", ComposeProject: "api", TrackedInputs: []string{"compose/api", "shared/defaults.json"}})
	other := fixture(t)
	c.Repositories["another"] = other.Repositories["repo-1"]
	s := other.Stacks[0]
	s.ID = "other"
	s.Repository = "another"
	s.ComposeProject = "other"
	c.Stacks = append(c.Stacks, s)
	r, err := Validate(c)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.RepositoryIDs(), []string{"another", "repo-1"}) || !reflect.DeepEqual(r.StackIDs(), []string{"api", "other", "web"}) {
		t.Fatal("IDs are not sorted")
	}
}

func TestDecodeStrictJSON(t *testing.T) {
	c := fixture(t)
	good := string(encode(t, c))
	cases := []struct {
		name, input string
		code        Code
		field       string
	}{
		{"malformed", `{`, InvalidJSON, "document"},
		{"nonobject", `[]`, InvalidJSON, "document"},
		{"missing version", strings.Replace(good, `"schema_version":1,`, "", 1), InvalidSchema, "schema_version"},
		{"string version", strings.Replace(good, `"schema_version":1`, `"schema_version":"1"`, 1), InvalidSchema, "schema_version"},
		{"float version", strings.Replace(good, `"schema_version":1`, `"schema_version":1.0`, 1), InvalidSchema, "schema_version"},
		{"unsupported version", strings.Replace(good, `"schema_version":1`, `"schema_version":2`, 1), UnsupportedSchema, "schema_version"},
		{"duplicate", strings.Replace(good, `"host":"host-1"`, `"host":"host-1","host":"other"`, 1), DuplicateField, "document"},
		{"escaped duplicate", strings.Replace(good, `"host":"host-1"`, `"host":"host-1","\u0068ost":"other"`, 1), DuplicateField, "document"},
		{"unknown top", strings.Replace(good, `"host":"host-1"`, `"host":"host-1","password":"CANARY"`, 1), UnknownField, "document"},
		{"case folded", strings.Replace(good, `"host"`, `"Host"`, 1), UnknownField, "document"},
		{"duplicate capability", strings.Replace(good, `"git_status":true`, `"git_status":true,"git_status":false`, 1), DuplicateField, "document"},
		{"unknown capability", strings.Replace(good, `"git_status":true`, `"execute_shell":true`, 1), UnsupportedCapability, "capabilities"},
		{"string capability", strings.Replace(good, `"git_status":true`, `"git_status":"true"`, 1), InvalidCapability, "capabilities.git_status"},
		{"null capability", strings.Replace(good, `"git_status":true`, `"git_status":null`, 1), InvalidJSON, "document"},
		{"unknown repository field", strings.Replace(good, `"root":`, `"helper":"CANARY","root":`, 1), UnknownField, "repositories[0]"},
		{"unknown stack field", strings.Replace(good, `"id":"web"`, `"id":"web","secret":"CANARY"`, 1), UnknownField, "stacks[0]"},
		{"empty optional branch", strings.Replace(good, `"root":`, `"branch":"","root":`, 1), InvalidJSON, "repositories[0].branch"},
		{"null object", `null`, InvalidJSON, "document"},
		{"trailing object", good + `{}`, InvalidJSON, "document"},
		{"trailing junk", good + `CANARY`, InvalidJSON, "document"},
		{"invalid utf8", good + string([]byte{0xff}), InvalidJSON, "document"},
		{"deep", strings.Repeat("[", 34) + "1" + strings.Repeat("]", 34), InvalidJSON, "document"},
		{"size", strings.Repeat(" ", MaxBytes+1), TooLarge, "document"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(tc.input))
			wantError(t, err, tc.code, tc.field)
			if strings.Contains(err.Error(), "CANARY") || strings.Contains(err.Error(), c.Repositories["repo-1"].Root) {
				t.Fatal("error leaks input")
			}
		})
	}
	// Repositories are a map: duplicate JSON keys must not be silently merged.
	data := `{"schema_version":1,"host":"host","repositories":{"r":{},"r":{}},"stacks":[]}`
	_, err := Decode(strings.NewReader(data))
	wantError(t, err, DuplicateField, "document")
}

func TestCapabilityDefaultsAndOptIns(t *testing.T) {
	data := `{"schema_version":1,"host":"host","repositories":{},"stacks":[]%s}`
	for _, tc := range []struct {
		suffix string
		want   Capabilities
	}{
		{"", Capabilities{GitStatus: true}},
		{`,"capabilities":{}`, Capabilities{GitStatus: true}},
		{`,"capabilities":{"git_status":false}`, Capabilities{}},
		{`,"capabilities":{"docker_deploy":true}`, Capabilities{GitStatus: true, DockerDeploy: true}},
	} {
		c, err := Decode(strings.NewReader(strings.Replace(data, "%s", tc.suffix, 1)))
		if err != nil {
			t.Fatal(err)
		}
		if c.Capabilities != tc.want {
			t.Fatalf("capabilities: %+v", c.Capabilities)
		}
	}
}

func TestValidationFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		code   Code
		field  string
	}{
		{"host empty", func(c *Config) { c.Host = "" }, InvalidID, "host"},
		{"option ID", func(c *Config) { c.Host = "--help" }, InvalidID, "host"},
		{"unsafe ID", func(c *Config) { c.Host = "x;echo CANARY" }, InvalidID, "host"},
		{"uppercase ID", func(c *Config) { c.Host = "Host" }, InvalidID, "host"},
		{"version", func(c *Config) { c.SchemaVersion = 0 }, UnsupportedSchema, "schema_version"},
		{"no repos", func(c *Config) { c.Repositories = nil }, InvalidID, "repositories"},
		{"no stacks", func(c *Config) { c.Stacks = nil }, InvalidID, "stacks"},
		{"empty repo ID", func(c *Config) { c.Repositories[""] = c.Repositories["repo-1"] }, InvalidID, "repositories[0]"},
		{"duplicate stack", func(c *Config) { c.Stacks = append(c.Stacks, c.Stacks[0]) }, DuplicateID, "stacks[1].id"},
		{"duplicate project", func(c *Config) { s := c.Stacks[0]; s.ID = "other"; c.Stacks = append(c.Stacks, s) }, DuplicateID, "stacks[1].compose_project"},
		{"missing stack ID", func(c *Config) { c.Stacks[0].ID = "" }, InvalidID, "stacks[0].id"},
		{"unsafe project", func(c *Config) { c.Stacks[0].ComposeProject = "--help" }, InvalidID, "stacks[0].compose_project"},
		{"unknown repo", func(c *Config) { c.Stacks[0].Repository = "missing" }, UnknownRepository, "stacks[0].repository"},
		{"missing root", func(c *Config) {
			r := c.Repositories["repo-1"]
			r.Root = filepath.Join(r.Root, "missing")
			c.Repositories["repo-1"] = r
		}, MissingPath, "repositories[0].root"},
		{"relative root", func(c *Config) { c.Repositories["repo-1"] = Repository{Root: "relative"} }, InvalidPath, "repositories[0].root"},
		{"root is file", func(c *Config) {
			r := c.Repositories["repo-1"]
			r.Root = filepath.Join(r.Root, "shared", "defaults.json")
			c.Repositories["repo-1"] = r
		}, InvalidPath, "repositories[0].root"},
		{"same roots", func(c *Config) { c.Repositories["repo-2"] = c.Repositories["repo-1"] }, OverlappingPath, "repositories[1].root"},
		{"nested roots", func(c *Config) {
			r := c.Repositories["repo-1"]
			r.Root = filepath.Join(r.Root, "shared")
			c.Repositories["repo-2"] = r
		}, OverlappingPath, "repositories[1].root"},
		{"missing stack path", func(c *Config) { c.Stacks[0].Path = "missing" }, MissingPath, "stacks[0].path"},
		{"stack is file", func(c *Config) { c.Stacks[0].Path = "shared/defaults.json" }, InvalidPath, "stacks[0].path"},
		{"overlapping stacks", func(c *Config) {
			s := c.Stacks[0]
			s.ID = "parent"
			s.ComposeProject = "parent"
			s.Path = "compose"
			c.Stacks = append(c.Stacks, s)
		}, OverlappingPath, "stacks[1].path"},
		{"empty inputs", func(c *Config) { c.Stacks[0].TrackedInputs = nil }, InvalidPath, "stacks[0].tracked_inputs"},
		{"missing input", func(c *Config) { c.Stacks[0].TrackedInputs = []string{"missing"} }, MissingPath, "stacks[0].tracked_inputs[0]"},
		{"duplicate normalized inputs", func(c *Config) { c.Stacks[0].TrackedInputs = []string{"compose/web", "./compose//web"} }, OverlappingPath, "stacks[0].tracked_inputs[1]"},
		{"nested inputs", func(c *Config) { c.Stacks[0].TrackedInputs = []string{"compose/web", "compose/web/compose.yaml"} }, OverlappingPath, "stacks[0].tracked_inputs[1]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := fixture(t)
			tc.mutate(&c)
			_, err := Validate(c)
			wantError(t, err, tc.code, tc.field)
		})
	}
}

func TestUnsafePaths(t *testing.T) {
	for _, path := range []string{"", "../outside", "compose/../web", "/tmp/injected", `C:\injected`, `compose\web`, "--help", ".git/config", ".env", ".env.production", "secrets/token", "data/db", "volumes/db", "key.pem", "shared/*.json", "bad\x00path"} {
		t.Run(path, func(t *testing.T) {
			c := fixture(t)
			c.Stacks[0].Path = path
			_, err := Validate(c)
			wantError(t, err, InvalidPath, "stacks[0].path")
			c = fixture(t)
			c.Stacks[0].TrackedInputs = []string{path}
			_, err = Validate(c)
			wantError(t, err, InvalidPath, "stacks[0].tracked_inputs[0]")
		})
	}
}

func TestSymlinkRejection(t *testing.T) {
	for _, kind := range []string{"root", "root ancestor", "stack", "input", "input ancestor", "internal input"} {
		t.Run(kind, func(t *testing.T) {
			c := fixture(t)
			root := c.Repositories["repo-1"].Root
			outside := t.TempDir()
			write(t, filepath.Join(outside, "canary.json"), "SECRET")
			link := filepath.Join(root, "link")
			target := outside
			if kind == "internal input" {
				target = filepath.Join(root, "shared")
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			field := "stacks[0].tracked_inputs[0]"
			switch kind {
			case "root":
				c.Repositories["repo-1"] = Repository{Root: link}
				field = "repositories[0].root"
			case "root ancestor":
				mkdir(t, filepath.Join(outside, "sub"))
				c.Repositories["repo-1"] = Repository{Root: filepath.Join(link, "sub")}
				field = "repositories[0].root"
			case "stack":
				c.Stacks[0].Path = "link"
				field = "stacks[0].path"
			case "input":
				c.Stacks[0].TrackedInputs = []string{"link"}
			case "input ancestor":
				c.Stacks[0].TrackedInputs = []string{"link/canary.json"}
			case "internal input":
				c.Stacks[0].TrackedInputs = []string{"link/defaults.json"}
			}
			_, err := Validate(c)
			wantError(t, err, InvalidPath, field)
		})
	}
}

func TestRemoteAndBranchSyntax(t *testing.T) {
	for _, remote := range []string{"", "https://example.invalid/repo.git", "ssh://git@example.invalid/repo.git", "ssh://git@example.invalid:2222/repo.git", "ssh://git@[::1]/repo.git", "git@example.invalid:repo.git"} {
		c := fixture(t)
		r := c.Repositories["repo-1"]
		r.URL = remote
		r.Branch = "feature/topic"
		c.Repositories["repo-1"] = r
		if _, err := Validate(c); err != nil {
			t.Fatalf("valid remote rejected: %v", err)
		}
	}
	for _, remote := range []string{"--upload-pack=CANARY", "ext::CANARY", "file:///tmp/repo", "http://example.invalid/repo", "https://user:CANARY@example.invalid/repo", "https://CANARY@example.invalid/repo", "ssh://git:CANARY@example.invalid/repo", "https://example.invalid/repo?token=CANARY", "https://example.invalid/repo#CANARY", "/tmp/repo", "ssh://-option/repo", "git@host:-option", "git@host:$(CANARY)", "ssh://host/repo%09CANARY", "https://host/repo%0aCANARY", "https://host/repo?", "https://host/repo#"} {
		c := fixture(t)
		r := c.Repositories["repo-1"]
		r.URL = remote
		c.Repositories["repo-1"] = r
		_, err := Validate(c)
		wantError(t, err, InvalidRemote, "repositories[0].url")
		if strings.Contains(err.Error(), "CANARY") {
			t.Fatal("credential leak")
		}
	}
	for _, branch := range []string{"--help", "HEAD", "refs/heads/main", "a..b", "a@{b", "a.lock", "a//b", "a/", ".hidden", "a b", "a\n", "a?", "a\\b"} {
		c := fixture(t)
		r := c.Repositories["repo-1"]
		r.Branch = branch
		c.Repositories["repo-1"] = r
		_, err := Validate(c)
		wantError(t, err, InvalidBranch, "repositories[0].branch")
	}
}

func TestDeterministicValidation(t *testing.T) {
	c := fixture(t)
	repo := c.Repositories["repo-1"]
	for i := 0; i < 50; i++ {
		c.Repositories = map[string]Repository{}
		if i%2 == 0 {
			c.Repositories["z"] = Repository{Root: "bad"}
			c.Repositories["a"] = repo
		} else {
			c.Repositories["a"] = repo
			c.Repositories["z"] = Repository{Root: "bad"}
		}
		_, err := Validate(c)
		wantError(t, err, InvalidPath, "repositories[1].root")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("SECRET reader details") }
func TestReaderErrorRedaction(t *testing.T) {
	_, err := Decode(failingReader{})
	wantError(t, err, ReadFailure, "document")
	if strings.Contains(err.Error(), "SECRET") {
		t.Fatal("reader leak")
	}
}

func TestConcurrentIndependentLoads(t *testing.T) {
	c := fixture(t)
	data := encode(t, c)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, err := Decode(bytes.NewReader(data))
			if err != nil {
				t.Error(err)
				return
			}
			r, err := Validate(cfg)
			if err != nil {
				t.Error(err)
				return
			}
			_ = r.StackIDs()
			_, _ = r.Stack("web")
		}()
	}
	wg.Wait()
}

func FuzzDecode(f *testing.F) {
	f.Add([]byte(`{"schema_version":1,"host":"h","repositories":{},"stacks":[]}`))
	f.Add([]byte(`{"host":"SECRET","host":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxBytes+1 {
			t.Skip()
		}
		_, err := Decode(bytes.NewReader(data))
		if err != nil {
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("untyped error: %T", err)
			}
			if len(e.Error()) > 160 {
				t.Fatal("unbounded error")
			}
		}
	})
}

var _ io.Reader = failingReader{}

func TestRootStackAndDirectoryInput(t *testing.T) {
	c := fixture(t)
	c.Stacks[0].Path = "."
	c.Stacks[0].TrackedInputs = []string{"."}
	if _, err := Validate(c); err != nil {
		t.Fatal(err)
	}
}

func TestValidationDoesNotModifyFiles(t *testing.T) {
	c := fixture(t)
	root := c.Repositories["repo-1"].Root
	before := map[string][]byte{}
	if err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err == nil {
			before[p] = data
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(c); err != nil {
		t.Fatal(err)
	}
	for p, want := range before {
		got, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("fixture modified")
		}
	}
}

func TestDocumentedRuntimeExample(t *testing.T) {
	data, err := os.ReadFile("../../examples/host-registration.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "compose", "caddy"))
	mkdir(t, filepath.Join(root, "shared"))
	write(t, filepath.Join(root, "shared", "non-secret-defaults.yaml"), "example: true")
	repo := c.Repositories["homelab"]
	repo.Root = root
	c.Repositories["homelab"] = repo
	if _, err := Validate(c); err != nil {
		t.Fatal(err)
	}
}
