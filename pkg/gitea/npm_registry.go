package gitea

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type NpmRegistry struct {
	client *Client
	owner  string
	token  string
}

func (r *NpmRegistry) baseURL() string {
	return fmt.Sprintf("%s/api/packages/%s/npm", r.client.baseURL, r.owner)
}

var ErrPackageNotFound = fmt.Errorf("package not found")

func (r *NpmRegistry) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := fmt.Sprintf("%s/%s", r.baseURL(), path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return r.client.httpClient.Do(req)
}

func (r *NpmRegistry) GetPackageMetadata(ctx context.Context, packageName string) (result map[string]any, err error) {
	resp, err := r.doRequest(ctx, http.MethodGet, url.QueryEscape(packageName), nil)
	if err != nil {
		return nil, fmt.Errorf("fetching package: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPackageNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

func (r *NpmRegistry) PackageExists(ctx context.Context, packageName string) (bool, error) {
	_, err := r.GetPackageMetadata(ctx, packageName)
	if errors.Is(err, ErrPackageNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (r *NpmRegistry) PackageVersionExists(ctx context.Context, packageName, version string) (bool, error) {
	metadata, err := r.GetPackageMetadata(ctx, packageName)
	if errors.Is(err, ErrPackageNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	versions, _ := metadata["versions"].(map[string]any)
	_, exists := versions[version]
	return exists, nil
}

func (r *NpmRegistry) UploadPackage(ctx context.Context, name, version string, tarball []byte, metadata map[string]any) (err error) {
	body, err := buildNpmUploadBody(r.client.baseURL, r.owner, name, version, tarball, metadata)
	if err != nil {
		return fmt.Errorf("building upload body: %w", err)
	}

	pkgPath := encodeScopedName(name)
	resp, err := r.doRequest(ctx, http.MethodPut, pkgPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("uploading package: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", closeErr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusConflict:
		return nil
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, respBody)
	}
}

func (r *NpmRegistry) DeletePackage(ctx context.Context, packageName, version string) (err error) {
	packagePath := url.PathEscape(packageName)
	deleteURL := fmt.Sprintf("%s/api/v1/packages/%s/npm/%s/%s", r.client.baseURL, r.owner, packagePath, version)
	deleteReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}
	deleteReq.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.client.httpClient.Do(deleteReq)
	if err != nil {
		return fmt.Errorf("deleting package: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func encodeScopedName(name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[0] + "%2f" + parts[1]
		}
	}
	return name
}

func unscopedName(name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return name
}

func buildNpmUploadBody(baseURL, owner, name, version string, tarball []byte, metadata map[string]any) ([]byte, error) {
	hash512 := sha512.Sum512(tarball)
	hash1 := sha1.Sum(tarball)
	integrity := fmt.Sprintf("sha512-%s", base64.StdEncoding.EncodeToString(hash512[:]))
	shasum := fmt.Sprintf("%x", hash1[:])

	tarballName := unscopedName(name)
	tarballFileName := fmt.Sprintf("%s-%s.tgz", tarballName, version)
	tarballURL := fmt.Sprintf("%s/api/packages/%s/npm/%s/-/%s",
		baseURL, owner, encodeScopedName(name), tarballFileName)

	manifest := map[string]any{
		"_id":     fmt.Sprintf("%s@%s", name, version),
		"name":    name,
		"version": version,
		"dist": map[string]any{
			"integrity": integrity,
			"shasum":    shasum,
			"tarball":   tarballURL,
		},
	}

	if metadata != nil {
		fields := []string{
			"scripts", "main", "module", "type", "bin", "repository",
			"description", "author", "license", "keywords", "homepage",
			"bugs", "engines", "os", "cpu",
			"dependencies", "peerDependencies", "devDependencies",
			"bundledDependencies", "optionalDependencies",
		}
		for _, field := range fields {
			if val, ok := metadata[field]; ok {
				manifest[field] = val
			}
		}
	}

	root := map[string]any{
		"_id":       name,
		"name":      name,
		"dist-tags": map[string]string{"latest": version},
		"versions":  map[string]any{version: manifest},
		"_attachments": map[string]any{
			tarballFileName: map[string]any{
				"content_type": "application/octet-stream",
				"data":         base64.StdEncoding.EncodeToString(tarball),
				"length":       len(tarball),
			},
		},
	}

	return json.Marshal(root)
}
