package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"git.duti.dev/secure-package-registry/pkg/npm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBaseURL = "https://git.duti.dev"

type stubMetadataFetcher struct {
	packages map[string]*npm.NPMPackage
}

func (s stubMetadataFetcher) FetchPackageMetadata(_ context.Context, pkgName string) (*npm.NPMPackage, error) {
	pkg, ok := s.packages[pkgName]
	if !ok {
		return nil, assert.AnError
	}
	return pkg, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMirrorerMirror(t *testing.T) {
	t.Parallel()

	graph := &npm.DependencyGraph{
		Root: "supports-color@7.2.0",
		Nodes: map[string]*npm.ResolvedPackage{
			"supports-color@7.2.0": {
				Name:       "supports-color",
				Version:    "7.2.0",
				TarballURL: "https://registry.npmjs.org/supports-color/-/supports-color-7.2.0.tgz",
			},
			"has-flag@4.0.0": {
				Name:       "has-flag",
				Version:    "4.0.0",
				TarballURL: "https://registry.npmjs.org/has-flag/-/has-flag-4.0.0.tgz",
			},
		},
		Edges: map[string][]string{
			"supports-color@7.2.0": {"has-flag@4.0.0"},
		},
	}

	metadata := stubMetadataFetcher{
		packages: map[string]*npm.NPMPackage{
			"supports-color": {
				Versions: map[string]npm.NPMVersion{
					"7.2.0": {
						Version:      "7.2.0",
						Dependencies: map[string]string{"has-flag": "^4.0.0"},
					},
				},
			},
			"has-flag": {
				Versions: map[string]npm.NPMVersion{
					"4.0.0": {
						Version: "4.0.0",
					},
				},
			},
		},
	}

	var uploadedPaths []string
	client := &Client{
		baseURL: testBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Host == "git.duti.dev":
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
						Header:     make(http.Header),
					}, nil
				case req.Method == http.MethodGet && req.URL.Host == "registry.npmjs.org":
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("tarball")),
						Header:     make(http.Header),
					}, nil
				case req.Method == http.MethodPut && req.URL.Host == "git.duti.dev":
					uploadedPaths = append(uploadedPaths, req.URL.Path)
					payload, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					var decoded map[string]any
					require.NoError(t, json.Unmarshal(payload, &decoded))
					versions, ok := decoded["versions"].(map[string]any)
					require.True(t, ok)
					assert.NotEmpty(t, versions)
					return &http.Response{
						StatusCode: http.StatusCreated,
						Body:       io.NopCloser(bytes.NewBufferString("created")),
						Header:     make(http.Header),
					}, nil
				default:
					return nil, assert.AnError
				}
			}),
		},
	}

	registry := &NpmRegistry{client: client, owner: "spr-sandbox", token: "token"}
	mirrorer := NewMirrorer(registry, metadata, MirrorOptions{Concurrency: 2})

	result, err := mirrorer.Mirror(context.Background(), graph)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"supports-color@7.2.0", "has-flag@4.0.0"}, result.Uploaded)
	assert.Empty(t, result.Skipped)
	assert.ElementsMatch(t, []string{
		"/api/packages/spr-sandbox/npm/has-flag",
		"/api/packages/spr-sandbox/npm/supports-color",
	}, uploadedPaths)
}

func TestMirrorerMirrorSkipsExistingPackages(t *testing.T) {
	t.Parallel()

	graph := &npm.DependencyGraph{
		Root: "has-flag@4.0.0",
		Nodes: map[string]*npm.ResolvedPackage{
			"has-flag@4.0.0": {
				Name:       "has-flag",
				Version:    "4.0.0",
				TarballURL: "https://registry.npmjs.org/has-flag/-/has-flag-4.0.0.tgz",
			},
		},
		Edges: map[string][]string{},
	}

	client := &Client{
		baseURL: testBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && req.URL.Host == "git.duti.dev" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(bytes.NewBufferString(`{
							"versions": {"4.0.0": {"name": "has-flag", "version": "4.0.0"}}
						}`)),
						Header: make(http.Header),
					}, nil
				}
				return nil, assert.AnError
			}),
		},
	}

	registry := &NpmRegistry{client: client, owner: "spr-sandbox", token: "token"}
	mirrorer := NewMirrorer(registry, stubMetadataFetcher{}, MirrorOptions{})

	result, err := mirrorer.Mirror(context.Background(), graph)
	require.NoError(t, err)
	assert.Empty(t, result.Uploaded)
	assert.Equal(t, []string{"has-flag@4.0.0"}, result.Skipped)
}
