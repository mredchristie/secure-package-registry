package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"

	"git.duti.dev/secure-package-registry/pkg/npm"
)

type MetadataFetcher interface {
	FetchPackageMetadata(ctx context.Context, pkgName string) (*npm.NPMPackage, error)
}

type MirrorOptions struct {
	Concurrency int
}

type MirrorResult struct {
	Uploaded []string
	Skipped  []string
}

type Mirrorer struct {
	registry    *NpmRegistry
	metadata    MetadataFetcher
	httpClient  *http.Client
	concurrency int
}

func NewMirrorer(registry *NpmRegistry, metadata MetadataFetcher, opts MirrorOptions) *Mirrorer {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 8
	}

	return &Mirrorer{
		registry:    registry,
		metadata:    metadata,
		httpClient:  &http.Client{Timeout: registry.client.httpClient.Timeout},
		concurrency: concurrency,
	}
}

func (m *Mirrorer) Mirror(ctx context.Context, graph *npm.DependencyGraph) (*MirrorResult, error) {
	if graph == nil {
		return nil, fmt.Errorf("dependency graph is nil")
	}

	keys := make([]string, 0, len(graph.Nodes))
	for key := range graph.Nodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := &MirrorResult{}
	resultMu := sync.Mutex{}
	sem := make(chan struct{}, m.concurrency)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for _, key := range keys {
		node := graph.Nodes[key]
		if node == nil {
			return nil, fmt.Errorf("dependency node %s is nil", key)
		}

		wg.Add(1)
		go func(nodeKey string, pkg *npm.ResolvedPackage) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				select {
				case errCh <- ctx.Err():
				default:
				}
				return
			}
			defer func() { <-sem }()

			skipped, err := m.mirrorPackage(ctx, pkg)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("mirroring %s: %w", nodeKey, err):
				default:
				}
				return
			}

			resultMu.Lock()
			if skipped {
				result.Skipped = append(result.Skipped, nodeKey)
			} else {
				result.Uploaded = append(result.Uploaded, nodeKey)
			}
			resultMu.Unlock()
		}(key, node)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		return nil, err
	case <-done:
	}

	sort.Strings(result.Uploaded)
	sort.Strings(result.Skipped)
	return result, nil
}

func (m *Mirrorer) mirrorPackage(ctx context.Context, pkg *npm.ResolvedPackage) (bool, error) {
	exists, err := m.registry.PackageVersionExists(ctx, pkg.Name, pkg.Version)
	if err != nil {
		return false, fmt.Errorf("checking package existence: %w", err)
	}
	if exists {
		return true, nil
	}

	packageMetadata, err := m.metadata.FetchPackageMetadata(ctx, pkg.Name)
	if err != nil {
		return false, fmt.Errorf("fetching package metadata: %w", err)
	}

	versionMetadata, ok := packageMetadata.Versions[pkg.Version]
	if !ok {
		return false, fmt.Errorf("version %s missing from package metadata", pkg.Version)
	}

	normalizedMetadata, err := marshalVersionMetadata(versionMetadata)
	if err != nil {
		return false, fmt.Errorf("normalizing package metadata: %w", err)
	}

	tarball, err := m.downloadTarball(ctx, pkg.TarballURL)
	if err != nil {
		return false, fmt.Errorf("downloading tarball: %w", err)
	}

	if err := m.registry.UploadPackage(ctx, pkg.Name, pkg.Version, tarball, normalizedMetadata); err != nil {
		return false, fmt.Errorf("uploading package: %w", err)
	}

	return false, nil
}

func (m *Mirrorer) downloadTarball(ctx context.Context, tarballURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating tarball request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing tarball request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected tarball status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tarball response: %w", err)
	}
	return data, nil
}

func marshalVersionMetadata(version npm.NPMVersion) (map[string]any, error) {
	data, err := json.Marshal(version)
	if err != nil {
		return nil, fmt.Errorf("marshaling version metadata: %w", err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshaling version metadata: %w", err)
	}

	delete(metadata, "dist")
	return metadata, nil
}
