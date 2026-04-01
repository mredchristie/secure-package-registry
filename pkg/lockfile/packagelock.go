package lockfile

import (
	"encoding/json"
	"fmt"
	"strings"
)

// packageLockJSON is the structure of a package-lock.json (v2/v3).
// v2 has both "dependencies" (flat, legacy) and "packages" (nested).
// v3 has only "packages". We prefer "packages" when available.
type packageLockJSON struct {
	Name            string                      `json:"name"`
	LockfileVersion int                         `json:"lockfileVersion"`
	Packages        map[string]packageLockEntry `json:"packages"`
	Dependencies    map[string]legacyLockEntry  `json:"dependencies"`
}

// packageLockEntry is a single entry in the v2/v3 "packages" map.
// The key is the node_modules path (e.g. "node_modules/express").
// The root entry has key "".
type packageLockEntry struct {
	Version         string            `json:"version"`
	Resolved        string            `json:"resolved"`
	Integrity       string            `json:"integrity"`
	Dev             bool              `json:"dev"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// legacyLockEntry is a single entry in the v1 "dependencies" map (fallback).
type legacyLockEntry struct {
	Version      string                     `json:"version"`
	Resolved     string                     `json:"resolved"`
	Integrity    string                     `json:"integrity"`
	Dev          bool                       `json:"dev"`
	Dependencies map[string]legacyLockEntry `json:"dependencies"`
}

// ParsePackageLockJSON parses a package-lock.json file (v1, v2, or v3) and
// returns all dependencies classified as direct or transitive.
//
// Direct dependencies are identified from the root entry's dependencies and
// devDependencies in the "packages" map. If the "packages" map is absent (v1),
// we fall back to "dependencies" and identify direct deps as top-level entries
// that are not marked "dev" or that appear in the root.
func ParsePackageLockJSON(data []byte) (*ParseResult, error) {
	var lock packageLockJSON
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing package-lock.json: %w", err)
	}

	// Prefer v2/v3 "packages" format.
	if len(lock.Packages) > 0 {
		return parsePackagesFormat(&lock)
	}

	// Fall back to v1 "dependencies" format.
	if len(lock.Dependencies) > 0 {
		return parseLegacyFormat(&lock)
	}

	return &ParseResult{
		Name:       lock.Name,
		SourceType: SourcePackageLockJSON,
	}, nil
}

// parsePackagesFormat handles v2/v3 lock files using the "packages" map.
func parsePackagesFormat(lock *packageLockJSON) (*ParseResult, error) {
	// Build the set of direct dependency names from the root entry.
	directNames := make(map[string]bool)
	root, hasRoot := lock.Packages[""]
	if hasRoot {
		for name := range root.Dependencies {
			directNames[name] = true
		}
		for name := range root.DevDependencies {
			directNames[name] = true
		}
	}

	seen := make(map[string]bool) // "name@version"
	var all []Dep

	for path, entry := range lock.Packages {
		if path == "" {
			continue // skip root entry
		}

		name := packageNameFromPath(path)
		if name == "" || entry.Version == "" {
			continue
		}

		key := name + "@" + entry.Version
		if seen[key] {
			continue
		}
		seen[key] = true

		isDirect := directNames[name]

		all = append(all, Dep{
			Name:    name,
			Version: entry.Version,
			Direct:  isDirect,
		})
	}

	var direct []Dep
	for _, d := range all {
		if d.Direct {
			direct = append(direct, d)
		}
	}

	return &ParseResult{
		Name:       lock.Name,
		SourceType: SourcePackageLockJSON,
		Direct:     direct,
		All:        all,
	}, nil
}

// parseLegacyFormat handles v1 lock files using the flat "dependencies" map.
func parseLegacyFormat(lock *packageLockJSON) (*ParseResult, error) {
	seen := make(map[string]bool)
	var all []Dep

	// In v1, top-level keys in "dependencies" are direct deps.
	// Nested "dependencies" within each entry are transitive.
	for name, entry := range lock.Dependencies {
		collectLegacyDeps(name, &entry, true, seen, &all)
	}

	var direct []Dep
	for _, d := range all {
		if d.Direct {
			direct = append(direct, d)
		}
	}

	return &ParseResult{
		Name:       lock.Name,
		SourceType: SourcePackageLockJSON,
		Direct:     direct,
		All:        all,
	}, nil
}

// collectLegacyDeps recursively collects deps from a v1 legacy entry.
func collectLegacyDeps(name string, entry *legacyLockEntry, isDirect bool, seen map[string]bool, out *[]Dep) {
	key := name + "@" + entry.Version
	if seen[key] {
		return
	}
	seen[key] = true

	*out = append(*out, Dep{
		Name:    name,
		Version: entry.Version,
		Direct:  isDirect,
	})

	for childName, childEntry := range entry.Dependencies {
		childCopy := childEntry
		collectLegacyDeps(childName, &childCopy, false, seen, out)
	}
}

// packageNameFromPath extracts the npm package name from a node_modules path.
//
// Examples:
//
//	"node_modules/express"                          → "express"
//	"node_modules/@babel/core"                      → "@babel/core"
//	"node_modules/express/node_modules/debug"       → "debug"
//	"node_modules/@scope/pkg/node_modules/@s/other" → "@s/other"
func packageNameFromPath(path string) string {
	// Find the last "node_modules/" segment.
	const prefix = "node_modules/"
	idx := strings.LastIndex(path, prefix)
	if idx < 0 {
		return ""
	}
	return path[idx+len(prefix):]
}
