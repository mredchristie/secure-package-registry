package lockfile_test

import (
	"sort"
	"testing"

	"git.duti.dev/secure-package-registry/pkg/lockfile"
)

// ---------------------------------------------------------------------------
// ParsePackageLockJSON — v2/v3 format (packages map)
// ---------------------------------------------------------------------------

func TestParsePackageLockJSON_V3Basic(t *testing.T) {
	data := []byte(`{
		"name": "my-app",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"name": "my-app",
				"dependencies": {
					"express": "^4.18.0"
				},
				"devDependencies": {
					"jest": "^29.0.0"
				}
			},
			"node_modules/express": {
				"version": "4.18.2",
				"resolved": "https://registry.npmjs.org/express/-/express-4.18.2.tgz"
			},
			"node_modules/jest": {
				"version": "29.7.0",
				"resolved": "https://registry.npmjs.org/jest/-/jest-29.7.0.tgz"
			},
			"node_modules/body-parser": {
				"version": "1.20.1",
				"resolved": "https://registry.npmjs.org/body-parser/-/body-parser-1.20.1.tgz"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "my-app" {
		t.Errorf("name = %q, want %q", r.Name, "my-app")
	}
	if r.SourceType != lockfile.SourcePackageLockJSON {
		t.Errorf("source type = %q, want %q", r.SourceType, lockfile.SourcePackageLockJSON)
	}

	// 2 direct deps: express + jest (devDep)
	if len(r.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d: %v", len(r.Direct), depNames(r.Direct))
	}

	directNames := depNames(r.Direct)
	sort.Strings(directNames)
	if directNames[0] != "express" || directNames[1] != "jest" {
		t.Errorf("direct deps = %v, want [express jest]", directNames)
	}

	// 3 total deps: express, jest, body-parser
	if len(r.All) != 3 {
		t.Fatalf("expected 3 total deps, got %d", len(r.All))
	}

	// body-parser is transitive
	byName := depsByName(r.All)
	if byName["body-parser"].Direct {
		t.Error("body-parser should be transitive")
	}
	if byName["body-parser"].Version != "1.20.1" {
		t.Errorf("body-parser version = %q, want %q", byName["body-parser"].Version, "1.20.1")
	}
}

func TestParsePackageLockJSON_V2Format(t *testing.T) {
	// v2 has both "packages" and "dependencies". We prefer "packages".
	data := []byte(`{
		"name": "v2-app",
		"lockfileVersion": 2,
		"packages": {
			"": {
				"name": "v2-app",
				"dependencies": {
					"lodash": "^4.17.0"
				}
			},
			"node_modules/lodash": {
				"version": "4.17.21"
			},
			"node_modules/lodash/node_modules/sub-dep": {
				"version": "1.0.0"
			}
		},
		"dependencies": {
			"lodash": {
				"version": "4.17.21"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "v2-app" {
		t.Errorf("name = %q, want %q", r.Name, "v2-app")
	}

	if len(r.Direct) != 1 {
		t.Fatalf("expected 1 direct dep, got %d", len(r.Direct))
	}
	if r.Direct[0].Name != "lodash" {
		t.Errorf("direct dep = %q, want %q", r.Direct[0].Name, "lodash")
	}

	// 2 total: lodash (direct) + sub-dep (transitive)
	if len(r.All) != 2 {
		t.Fatalf("expected 2 total deps, got %d", len(r.All))
	}
}

func TestParsePackageLockJSON_ScopedPackages(t *testing.T) {
	data := []byte(`{
		"name": "scoped-app",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"dependencies": {
					"@babel/core": "^7.24.0",
					"@types/node": "^20.0.0"
				}
			},
			"node_modules/@babel/core": {
				"version": "7.24.5"
			},
			"node_modules/@types/node": {
				"version": "20.12.7"
			},
			"node_modules/@babel/core/node_modules/@babel/helper-module-transforms": {
				"version": "7.24.5"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d: %v", len(r.Direct), depNames(r.Direct))
	}

	directNames := depNames(r.Direct)
	sort.Strings(directNames)
	if directNames[0] != "@babel/core" || directNames[1] != "@types/node" {
		t.Errorf("direct deps = %v, want [@babel/core @types/node]", directNames)
	}

	// Nested scoped package should be transitive
	if len(r.All) != 3 {
		t.Fatalf("expected 3 total deps, got %d", len(r.All))
	}

	byName := depsByName(r.All)
	helperMod := byName["@babel/helper-module-transforms"]
	if helperMod.Direct {
		t.Error("@babel/helper-module-transforms should be transitive")
	}
	if helperMod.Version != "7.24.5" {
		t.Errorf("helper version = %q, want %q", helperMod.Version, "7.24.5")
	}
}

func TestParsePackageLockJSON_DevDependenciesAreDirect(t *testing.T) {
	data := []byte(`{
		"name": "dev-app",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"devDependencies": {
					"vitest": "^1.0.0"
				}
			},
			"node_modules/vitest": {
				"version": "1.6.0"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.Direct) != 1 {
		t.Fatalf("expected 1 direct dep, got %d", len(r.Direct))
	}
	if !r.Direct[0].Direct {
		t.Error("vitest should be direct (devDependency)")
	}
}

func TestParsePackageLockJSON_SkipsEntriesWithoutVersion(t *testing.T) {
	data := []byte(`{
		"name": "skip-app",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"dependencies": {
					"real-pkg": "^1.0.0"
				}
			},
			"node_modules/real-pkg": {
				"version": "1.2.3"
			},
			"node_modules/no-version": {
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// no-version entry should be skipped
	if len(r.All) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(r.All))
	}
	if r.All[0].Name != "real-pkg" {
		t.Errorf("dep name = %q, want %q", r.All[0].Name, "real-pkg")
	}
}

func TestParsePackageLockJSON_DeduplicatesSameVersion(t *testing.T) {
	// In v2, the same package may appear at multiple paths if hoisted and nested.
	// They should be deduplicated by name@version.
	data := []byte(`{
		"name": "dedup-app",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"dependencies": {
					"debug": "^4.0.0"
				}
			},
			"node_modules/debug": {
				"version": "4.3.4"
			},
			"node_modules/express/node_modules/debug": {
				"version": "4.3.4"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be deduplicated to just one entry
	if len(r.All) != 1 {
		t.Fatalf("expected 1 dep (deduplicated), got %d", len(r.All))
	}
	if r.All[0].Name != "debug" {
		t.Errorf("dep name = %q, want %q", r.All[0].Name, "debug")
	}
	if r.All[0].Version != "4.3.4" {
		t.Errorf("dep version = %q, want %q", r.All[0].Version, "4.3.4")
	}
}

// ---------------------------------------------------------------------------
// ParsePackageLockJSON — v1 legacy format (dependencies map)
// ---------------------------------------------------------------------------

func TestParsePackageLockJSON_V1Legacy(t *testing.T) {
	data := []byte(`{
		"name": "legacy-app",
		"lockfileVersion": 1,
		"dependencies": {
			"express": {
				"version": "4.18.2",
				"resolved": "https://registry.npmjs.org/express/-/express-4.18.2.tgz",
				"dependencies": {
					"body-parser": {
						"version": "1.20.1",
						"resolved": "https://registry.npmjs.org/body-parser/-/body-parser-1.20.1.tgz"
					}
				}
			},
			"lodash": {
				"version": "4.17.21",
				"resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "legacy-app" {
		t.Errorf("name = %q, want %q", r.Name, "legacy-app")
	}

	// Top-level deps are direct: express, lodash
	if len(r.Direct) != 2 {
		t.Fatalf("expected 2 direct deps, got %d: %v", len(r.Direct), depNames(r.Direct))
	}

	directNames := depNames(r.Direct)
	sort.Strings(directNames)
	if directNames[0] != "express" || directNames[1] != "lodash" {
		t.Errorf("direct deps = %v, want [express lodash]", directNames)
	}

	// Total: express + lodash + body-parser
	if len(r.All) != 3 {
		t.Fatalf("expected 3 total deps, got %d", len(r.All))
	}

	byName := depsByName(r.All)
	if byName["body-parser"].Direct {
		t.Error("body-parser should be transitive in v1")
	}
	if byName["body-parser"].Version != "1.20.1" {
		t.Errorf("body-parser version = %q, want %q", byName["body-parser"].Version, "1.20.1")
	}
}

func TestParsePackageLockJSON_V1DeeplyNested(t *testing.T) {
	data := []byte(`{
		"name": "nested-app",
		"lockfileVersion": 1,
		"dependencies": {
			"a": {
				"version": "1.0.0",
				"dependencies": {
					"b": {
						"version": "2.0.0",
						"dependencies": {
							"c": {
								"version": "3.0.0"
							}
						}
					}
				}
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// a is direct, b and c are transitive
	if len(r.Direct) != 1 {
		t.Fatalf("expected 1 direct dep, got %d", len(r.Direct))
	}
	if r.Direct[0].Name != "a" {
		t.Errorf("direct dep = %q, want %q", r.Direct[0].Name, "a")
	}

	if len(r.All) != 3 {
		t.Fatalf("expected 3 total deps, got %d", len(r.All))
	}

	byName := depsByName(r.All)
	if byName["b"].Direct {
		t.Error("b should be transitive")
	}
	if byName["c"].Direct {
		t.Error("c should be transitive")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestParsePackageLockJSON_EmptyPackagesAndDependencies(t *testing.T) {
	data := []byte(`{
		"name": "empty-lock",
		"lockfileVersion": 3,
		"packages": {}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty packages map (only root or nothing) → still valid, 0 deps
	if len(r.All) != 0 {
		t.Errorf("expected 0 deps, got %d", len(r.All))
	}
}

func TestParsePackageLockJSON_NoPackagesOrDeps(t *testing.T) {
	data := []byte(`{
		"name": "bare-lock",
		"lockfileVersion": 3
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Name != "bare-lock" {
		t.Errorf("name = %q, want %q", r.Name, "bare-lock")
	}
	if len(r.All) != 0 {
		t.Errorf("expected 0 deps, got %d", len(r.All))
	}
}

func TestParsePackageLockJSON_InvalidJSON(t *testing.T) {
	_, err := lockfile.ParsePackageLockJSON([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParsePackageLockJSON_NoRootEntry(t *testing.T) {
	// packages map exists but no "" root entry → can't determine direct deps
	// All deps should be transitive.
	data := []byte(`{
		"name": "no-root",
		"lockfileVersion": 3,
		"packages": {
			"node_modules/express": {
				"version": "4.18.2"
			},
			"node_modules/lodash": {
				"version": "4.17.21"
			}
		}
	}`)

	r, err := lockfile.ParsePackageLockJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without root entry, nothing is in directNames, so all are transitive
	if len(r.Direct) != 0 {
		t.Errorf("expected 0 direct deps (no root entry), got %d", len(r.Direct))
	}
	if len(r.All) != 2 {
		t.Fatalf("expected 2 total deps, got %d", len(r.All))
	}
}
