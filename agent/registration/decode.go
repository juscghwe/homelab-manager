package registration

import (
	"bytes"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"unicode/utf8"
)

// Decode strictly decodes at most MaxBytes of JSON, without filesystem access.
// Omitted capabilities default to git_status=true and all other flags=false.
// Call Validate before using the returned declarations for any operation.
func Decode(r io.Reader) (Config, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Config{}, fail(ReadFailure, "document")
	}
	if len(data) > MaxBytes {
		return Config{}, fail(TooLarge, "document")
	}
	if !utf8.Valid(data) {
		return Config{}, fail(InvalidJSON, "document")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := scanValue(d, 0); err != nil {
		return Config{}, err
	}
	if _, err := d.Token(); err != io.EOF {
		return Config{}, fail(InvalidJSON, "document")
	}
	obj, err := object(data, "document", []string{"schema_version", "host", "repositories", "stacks", "capabilities"}, UnknownField)
	if err != nil {
		return Config{}, err
	}
	var version int
	if err := json.Unmarshal(obj["schema_version"], &version); err != nil {
		return Config{}, fail(InvalidSchema, "schema_version")
	}
	if version != SchemaVersion {
		return Config{}, fail(UnsupportedSchema, "schema_version")
	}
	repos, err := object(obj["repositories"], "repositories", nil, UnknownField)
	if err != nil {
		return Config{}, err
	}
	for i, id := range sortedKeys(repos) {
		field := "repositories[" + strconv.Itoa(i) + "]"
		v, err := object(repos[id], field, []string{"root", "url", "branch"}, UnknownField)
		if err != nil {
			return Config{}, err
		}
		for _, name := range []string{"url", "branch"} {
			if raw, exists := v[name]; exists {
				var s string
				if json.Unmarshal(raw, &s) != nil || s == "" {
					return Config{}, fail(InvalidJSON, field+"."+name)
				}
			}
		}
	}
	var stacks []json.RawMessage
	if json.Unmarshal(obj["stacks"], &stacks) != nil {
		return Config{}, fail(InvalidJSON, "stacks")
	}
	for i, raw := range stacks {
		if _, err := object(raw, "stacks["+strconv.Itoa(i)+"]", []string{"id", "repository", "path", "compose_project", "tracked_inputs"}, UnknownField); err != nil {
			return Config{}, err
		}
	}
	if raw, exists := obj["capabilities"]; exists {
		caps, err := object(raw, "capabilities", []string{"git_status", "docker_status", "docker_deploy", "docker_stop", "nftables_apply"}, UnsupportedCapability)
		if err != nil {
			return Config{}, err
		}
		for _, name := range sortedKeys(caps) {
			var v bool
			if json.Unmarshal(caps[name], &v) != nil {
				return Config{}, fail(InvalidCapability, "capabilities."+name)
			}
		}
	}
	cfg := Config{Capabilities: Capabilities{GitStatus: true}}
	if json.Unmarshal(data, &cfg) != nil {
		return Config{}, fail(InvalidJSON, "document")
	}
	return cfg, nil
}

// scanValue rejects duplicate keys (including escaped spellings), nulls, and
// deep nesting before encoding/json can silently accept/overwrite them.
func scanValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return fail(InvalidJSON, "document")
	}
	t, err := d.Token()
	if err != nil || t == nil {
		return fail(InvalidJSON, "document")
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			t, err := d.Token()
			if err != nil {
				return fail(InvalidJSON, "document")
			}
			key, ok := t.(string)
			if !ok {
				return fail(InvalidJSON, "document")
			}
			if seen[key] {
				return fail(DuplicateField, "document")
			}
			seen[key] = true
			if err := scanValue(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := scanValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return fail(InvalidJSON, "document")
	}
	end, err := d.Token()
	if err != nil || (delim == '{' && end != json.Delim('}')) || (delim == '[' && end != json.Delim(']')) {
		return fail(InvalidJSON, "document")
	}
	return nil
}

func object(data []byte, field string, allowed []string, unknown Code) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(data, &obj) != nil || obj == nil {
		return nil, fail(InvalidJSON, field)
	}
	if allowed != nil {
		for _, key := range sortedKeys(obj) {
			if !slices.Contains(allowed, key) {
				return nil, fail(unknown, field)
			}
		}
	}
	return obj, nil
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
