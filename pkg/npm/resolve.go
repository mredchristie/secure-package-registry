package npm

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/rs/zerolog"

	"git.duti.dev/secure-package-registry/pkg/logger"
)

// ResolvedPackage is a single node in the resolved dependency graph.
type ResolvedPackage struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	TarballURL string `json:"tarball_url"`
	Integrity  string `json:"integrity,omitempty"`
	Shasum     string `json:"shasum,omitempty"`
}

// DependencyGraph is the result of resolving a package's full transitive
// dependency tree. Nodes are keyed by "name@version".
type DependencyGraph struct {
	Root  string                      `json:"root"`
	Nodes map[string]*ResolvedPackage `json:"nodes"`
	Edges map[string][]string         `json:"edges"`
}

// Resolver resolves the full transitive dependency graph for an NPM package.
type Resolver struct {
	client      *Client
	concurrency int
	log         zerolog.Logger
}

// ResolverOption configures a Resolver.
type ResolverOption func(*Resolver)

// WithConcurrency sets the maximum number of concurrent registry fetches.
func WithConcurrency(n int) ResolverOption {
	return func(r *Resolver) {
		if n > 0 {
			r.concurrency = n
		}
	}
}

// NewResolver creates a Resolver that uses the given Client to fetch package metadata.
func NewResolver(client *Client, opts ...ResolverOption) *Resolver {
	r := &Resolver{
		client:      client,
		concurrency: 16,
		log:         logger.WithComponent("npm-resolver"),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Resolve builds the full transitive dependency graph for the given package and
// exact version. The version must be an exact semver (not a range).
func (r *Resolver) Resolve(ctx context.Context, name, version string) (*DependencyGraph, error) {
	if _, err := semver.NewVersion(version); err != nil {
		return nil, fmt.Errorf("version %q is not valid semver: %w", version, err)
	}

	graph := &DependencyGraph{
		Root:  nodeKey(name, version),
		Nodes: make(map[string]*ResolvedPackage),
		Edges: make(map[string][]string),
	}

	state := &resolverState{
		resolver:      r,
		ctx:           ctx,
		graph:         graph,
		mu:            sync.Mutex{},
		sem:           make(chan struct{}, r.concurrency),
		wg:            sync.WaitGroup{},
		metadataCache: make(map[string]*NPMPackage),
		cacheMu:       sync.Mutex{},
	}

	state.wg.Add(1)
	go state.resolve(name, version)
	state.wg.Wait()

	if state.err != nil {
		return nil, state.err
	}

	return graph, nil
}

type resolverState struct {
	resolver      *Resolver
	ctx           context.Context
	graph         *DependencyGraph
	mu            sync.Mutex
	sem           chan struct{}
	wg            sync.WaitGroup
	metadataCache map[string]*NPMPackage
	cacheMu       sync.Mutex
	err           error
	errOnce       sync.Once
}

func (s *resolverState) setError(err error) {
	s.errOnce.Do(func() {
		s.err = err
	})
}

func (s *resolverState) resolve(name, version string) {
	defer s.wg.Done()

	if s.err != nil {
		return
	}

	key := nodeKey(name, version)

	s.mu.Lock()
	if _, exists := s.graph.Nodes[key]; exists {
		s.mu.Unlock()
		return
	}
	s.graph.Nodes[key] = nil // placeholder to mark in-progress
	s.mu.Unlock()

	select {
	case s.sem <- struct{}{}:
	case <-s.ctx.Done():
		s.setError(s.ctx.Err())
		return
	}

	pkgMeta, err := s.fetchMetadata(name)
	<-s.sem
	if err != nil {
		s.setError(fmt.Errorf("fetching metadata for %s: %w", name, err))
		return
	}

	versionData, ok := pkgMeta.Versions[version]
	if !ok {
		s.setError(fmt.Errorf("version %s not found in metadata for %s", version, name))
		return
	}

	node := &ResolvedPackage{
		Name:       name,
		Version:    version,
		TarballURL: versionData.Dist.Tarball,
		Integrity:  versionData.Dist.Integrity,
		Shasum:     versionData.Dist.Shasum,
	}

	deps := versionData.Dependencies

	var childKeys []string
	for depName, depRange := range deps {
		resolved, err := s.resolveDepVersion(depName, depRange)
		if err != nil {
			s.setError(fmt.Errorf("resolving %s@%s (dependency of %s@%s): %w", depName, depRange, name, version, err))
			return
		}
		childKey := nodeKey(depName, resolved)
		childKeys = append(childKeys, childKey)
	}
	sort.Strings(childKeys)

	s.mu.Lock()
	s.graph.Nodes[key] = node
	s.graph.Edges[key] = childKeys
	s.mu.Unlock()
}

func (s *resolverState) fetchMetadata(name string) (*NPMPackage, error) {
	s.cacheMu.Lock()
	if pkg, ok := s.metadataCache[name]; ok {
		s.cacheMu.Unlock()
		return pkg, nil
	}
	s.cacheMu.Unlock()

	pkg, err := s.resolver.client.FetchPackageMetadata(s.ctx, name)
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.metadataCache[name] = pkg
	s.cacheMu.Unlock()

	return pkg, nil
}

func (s *resolverState) resolveDepVersion(depName, depRange string) (string, error) {
	s.cacheMu.Lock()
	depMeta, ok := s.metadataCache[depName]
	s.cacheMu.Unlock()

	if !ok {
		select {
		case s.sem <- struct{}{}:
		case <-s.ctx.Done():
			return "", s.ctx.Err()
		}
		var err error
		depMeta, err = s.resolver.client.FetchPackageMetadata(s.ctx, depName)
		<-s.sem
		if err != nil {
			return "", err
		}

		s.cacheMu.Lock()
		s.metadataCache[depName] = depMeta
		s.cacheMu.Unlock()
	}

	resolved, err := matchVersion(depMeta, depRange)
	if err != nil {
		return "", fmt.Errorf("no version of %s satisfies %q: %w", depName, depRange, err)
	}

	s.wg.Add(1)
	go s.resolve(depName, resolved)

	return resolved, nil
}

// matchVersion finds the highest version in the package metadata that satisfies
// the given constraint string. Handles exact versions, ranges, and dist-tags.
func matchVersion(pkg *NPMPackage, constraint string) (string, error) {
	if v, ok := pkg.DistTags[constraint]; ok {
		return v, nil
	}

	if _, ok := pkg.Versions[constraint]; ok {
		if _, err := semver.NewVersion(constraint); err == nil {
			return constraint, nil
		}
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", fmt.Errorf("parsing constraint %q: %w", constraint, err)
	}

	var matched []*semver.Version
	for vStr := range pkg.Versions {
		v, err := semver.NewVersion(vStr)
		if err != nil {
			continue
		}
		if c.Check(v) {
			matched = append(matched, v)
		}
	}

	if len(matched) == 0 {
		return "", fmt.Errorf("no versions match constraint %q", constraint)
	}

	sort.Sort(semver.Collection(matched))
	return matched[len(matched)-1].Original(), nil
}

func nodeKey(name, version string) string {
	return name + "@" + version
}

// ResolveConstraint fetches the package metadata from the npm registry and
// returns the highest version that satisfies the given semver constraint.
// This is used to resolve constraints from package.json (e.g. "^5.48.2")
// into exact versions (e.g. "5.50.1").
func (c *Client) ResolveConstraint(ctx context.Context, pkgName, constraint string) (string, error) {
	meta, err := c.FetchPackageMetadata(ctx, pkgName)
	if err != nil {
		return "", fmt.Errorf("fetching metadata for %s: %w", pkgName, err)
	}
	resolved, err := matchVersion(meta, constraint)
	if err != nil {
		return "", fmt.Errorf("resolving %s@%s: %w", pkgName, constraint, err)
	}
	return resolved, nil
}
