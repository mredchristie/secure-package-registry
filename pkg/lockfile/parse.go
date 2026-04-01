// Package lockfile parses package.json and lock files (package-lock.json)
// into a common representation of direct and transitive dependencies.
package lockfile

import (
	"encoding/json"
	"fmt"
)

// SourceType identifies the kind of file that was parsed.
type SourceType string

const (
	SourcePackageJSON     SourceType = "package.json"
	SourcePackageLockJSON SourceType = "package-lock.json"
)

// Dep represents a single dependency extracted from a manifest or lock file.
type Dep struct {
	Name       string // npm package name (e.g. "express", "@babel/core")
	Version    string // resolved exact version (e.g. "4.18.2")
	Constraint string // original semver range from package.json (empty for lock files)
	Direct     bool   // true = direct dependency, false = transitive
}

// ParseResult is the output of parsing a manifest or lock file.
type ParseResult struct {
	Name       string     // project name from the file (may be empty)
	SourceType SourceType // which file format was parsed
	Direct     []Dep      // direct dependencies only
	All        []Dep      // all dependencies (direct + transitive), deduplicated
}

// DirectDeps returns only the direct dependencies from All.
func (r *ParseResult) DirectDeps() []Dep {
	var out []Dep
	for _, d := range r.All {
		if d.Direct {
			out = append(out, d)
		}
	}
	return out
}

// TransitiveDeps returns only the transitive dependencies from All.
func (r *ParseResult) TransitiveDeps() []Dep {
	var out []Dep
	for _, d := range r.All {
		if !d.Direct {
			out = append(out, d)
		}
	}
	return out
}

// Parse auto-detects whether data is a package.json or package-lock.json and
// dispatches to the appropriate parser.
//
// Detection heuristic: if the JSON contains a "lockfileVersion" key it's a
// lock file; otherwise it's a package.json.
func Parse(data []byte) (*ParseResult, error) {
	var probe struct {
		LockfileVersion *int `json:"lockfileVersion"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("detecting file type: %w", err)
	}

	if probe.LockfileVersion != nil {
		return ParsePackageLockJSON(data)
	}
	return ParsePackageJSON(data)
}
