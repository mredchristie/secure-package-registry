package lockfile_test

import (
	"sort"
	"testing"

	"git.duti.dev/secure-package-registry/pkg/lockfile"
)

// ---------------------------------------------------------------------------
// ParseResult helper methods
// ---------------------------------------------------------------------------

func TestParseResult_DirectDeps(t *testing.T) {
	r := &lockfile.ParseResult{
		All: []lockfile.Dep{
			{Name: "a", Version: "1.0.0", Direct: true},
			{Name: "b", Version: "2.0.0", Direct: false},
			{Name: "c", Version: "3.0.0", Direct: true},
		},
	}

	got := r.DirectDeps()
	if len(got) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(got))
	}

	names := []string{got[0].Name, got[1].Name}
	sort.Strings(names)
	if names[0] != "a" || names[1] != "c" {
		t.Errorf("unexpected direct deps: %v", names)
	}
}

func TestParseResult_TransitiveDeps(t *testing.T) {
	r := &lockfile.ParseResult{
		All: []lockfile.Dep{
			{Name: "a", Version: "1.0.0", Direct: true},
			{Name: "b", Version: "2.0.0", Direct: false},
			{Name: "c", Version: "3.0.0", Direct: false},
		},
	}

	got := r.TransitiveDeps()
	if len(got) != 2 {
		t.Fatalf("expected 2 transitive deps, got %d", len(got))
	}

	names := []string{got[0].Name, got[1].Name}
	sort.Strings(names)
	if names[0] != "b" || names[1] != "c" {
		t.Errorf("unexpected transitive deps: %v", names)
	}
}

func TestParseResult_EmptyAll(t *testing.T) {
	r := &lockfile.ParseResult{}

	if got := r.DirectDeps(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	if got := r.TransitiveDeps(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Parse (auto-detection)
// ---------------------------------------------------------------------------

func TestParse_DetectsPackageLockJSON(t *testing.T) {
	data := []byte(`{
		"name": "my-app",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"dependencies": { "express": "^4.0.0" }
			},
			"node_modules/express": { "version": "4.18.2" }
		}
	}`)

	r, err := lockfile.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.SourceType != lockfile.SourcePackageLockJSON {
		t.Errorf("source type = %q, want %q", r.SourceType, lockfile.SourcePackageLockJSON)
	}
	if len(r.All) != 1 {
		t.Errorf("expected 1 dep, got %d", len(r.All))
	}
}

func TestParse_DetectsPackageJSON(t *testing.T) {
	data := []byte(`{
		"name": "my-app",
		"dependencies": {
			"express": "^4.18.0"
		}
	}`)

	r, err := lockfile.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.SourceType != lockfile.SourcePackageJSON {
		t.Errorf("source type = %q, want %q", r.SourceType, lockfile.SourcePackageJSON)
	}
	if len(r.Direct) != 1 {
		t.Errorf("expected 1 dep, got %d", len(r.Direct))
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := lockfile.Parse([]byte(`not json at all`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
