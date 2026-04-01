package lockfile

import (
	"encoding/json"
	"fmt"
)

// packageJSON is the minimal structure we need from a package.json file.
type packageJSON struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// ParsePackageJSON extracts direct dependencies (including devDependencies) from
// a package.json file. The returned ParseResult contains only direct deps with
// semver range constraints — the caller must resolve them to exact versions and
// build the transitive tree separately.
func ParsePackageJSON(data []byte) (*ParseResult, error) {
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package.json: %w", err)
	}

	seen := make(map[string]bool)
	var direct []Dep

	// dependencies
	for name, constraint := range pkg.Dependencies {
		if seen[name] {
			continue
		}
		seen[name] = true
		direct = append(direct, Dep{
			Name:       name,
			Constraint: constraint,
			Direct:     true,
		})
	}

	// devDependencies are treated as direct (they execute during install/test)
	for name, constraint := range pkg.DevDependencies {
		if seen[name] {
			continue
		}
		seen[name] = true
		direct = append(direct, Dep{
			Name:       name,
			Constraint: constraint,
			Direct:     true,
		})
	}

	return &ParseResult{
		Name:       pkg.Name,
		SourceType: SourcePackageJSON,
		Direct:     direct,
		All:        direct, // only direct deps known; transitive resolved later
	}, nil
}
