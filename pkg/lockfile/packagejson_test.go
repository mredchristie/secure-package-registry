package lockfile_test

import (
	"sort"
	"testing"

	"git.duti.dev/secure-package-registry/pkg/lockfile"
)

// ---------------------------------------------------------------------------
// ParsePackageJSON
// ---------------------------------------------------------------------------

func TestParsePackageJSON_Basic(t *testing.T) {
	data := []byte(`{
		"name": "my-app",
		"dependencies": {
			"express": "^4.18.0",
			"lodash": "~4.17.21"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`)

	r, err := lockfile.ParsePackageJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "my-app" {
		t.Errorf("name = %q, want %q", r.Name, "my-app")
	}
	if r.SourceType != lockfile.SourcePackageJSON {
		t.Errorf("source type = %q, want %q", r.SourceType, lockfile.SourcePackageJSON)
	}

	// 3 direct deps: express, lodash, jest (devDep counts as direct)
	if len(r.Direct) != 3 {
		t.Fatalf("expected 3 direct deps, got %d", len(r.Direct))
	}

	// All should equal Direct for package.json (no transitive info)
	if len(r.All) != len(r.Direct) {
		t.Errorf("All (%d) != Direct (%d)", len(r.All), len(r.Direct))
	}

	// All should be marked direct
	for _, d := range r.All {
		if !d.Direct {
			t.Errorf("dep %q should be direct", d.Name)
		}
		if d.Version != "" {
			t.Errorf("dep %q should have empty version (only constraint), got %q", d.Name, d.Version)
		}
	}

	// Check constraint values
	byName := depsByName(r.Direct)
	if byName["express"].Constraint != "^4.18.0" {
		t.Errorf("express constraint = %q, want %q", byName["express"].Constraint, "^4.18.0")
	}
	if byName["jest"].Constraint != "^29.0.0" {
		t.Errorf("jest constraint = %q, want %q", byName["jest"].Constraint, "^29.0.0")
	}
}

func TestParsePackageJSON_ScopedPackages(t *testing.T) {
	data := []byte(`{
		"name": "@myorg/my-lib",
		"dependencies": {
			"@babel/core": "^7.24.0",
			"@types/node": "^20.0.0"
		},
		"devDependencies": {
			"@eslint/js": "^9.0.0"
		}
	}`)

	r, err := lockfile.ParsePackageJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "@myorg/my-lib" {
		t.Errorf("name = %q, want %q", r.Name, "@myorg/my-lib")
	}

	if len(r.Direct) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(r.Direct))
	}

	names := depNames(r.Direct)
	sort.Strings(names)
	expected := []string{"@babel/core", "@eslint/js", "@types/node"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("dep[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestParsePackageJSON_DevDependenciesAreDirect(t *testing.T) {
	data := []byte(`{
		"name": "test-proj",
		"devDependencies": {
			"vitest": "^1.0.0",
			"prettier": "^3.0.0"
		}
	}`)

	r, err := lockfile.ParsePackageJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d", len(r.Direct))
	}

	for _, d := range r.Direct {
		if !d.Direct {
			t.Errorf("dep %q should be direct", d.Name)
		}
	}
}

func TestParsePackageJSON_DeduplicateOverlap(t *testing.T) {
	// If the same package appears in both dependencies and devDependencies,
	// it should only appear once.
	data := []byte(`{
		"name": "overlap-proj",
		"dependencies": {
			"shared-pkg": "^1.0.0"
		},
		"devDependencies": {
			"shared-pkg": "^1.0.0"
		}
	}`)

	r, err := lockfile.ParsePackageJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.Direct) != 1 {
		t.Fatalf("expected 1 dep (deduplicated), got %d", len(r.Direct))
	}

	if r.Direct[0].Name != "shared-pkg" {
		t.Errorf("dep name = %q, want %q", r.Direct[0].Name, "shared-pkg")
	}
}

func TestParsePackageJSON_NoDeps(t *testing.T) {
	data := []byte(`{"name": "empty-proj"}`)

	r, err := lockfile.ParsePackageJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "empty-proj" {
		t.Errorf("name = %q, want %q", r.Name, "empty-proj")
	}
	if len(r.Direct) != 0 {
		t.Errorf("expected 0 deps, got %d", len(r.Direct))
	}
}

func TestParsePackageJSON_InvalidJSON(t *testing.T) {
	_, err := lockfile.ParsePackageJSON([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParsePackageJSON_EmptyObject(t *testing.T) {
	r, err := lockfile.ParsePackageJSON([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "" {
		t.Errorf("name = %q, want empty", r.Name)
	}
	if len(r.Direct) != 0 {
		t.Errorf("expected 0 deps, got %d", len(r.Direct))
	}
}

// --- helpers ---

func depsByName(deps []lockfile.Dep) map[string]lockfile.Dep {
	m := make(map[string]lockfile.Dep, len(deps))
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func depNames(deps []lockfile.Dep) []string {
	names := make([]string, len(deps))
	for i, d := range deps {
		names[i] = d.Name
	}
	return names
}
