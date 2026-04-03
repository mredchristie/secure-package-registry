package private

// --- Package handler types ---

// CreatePackageRequest is the request body for CreatePackage.
type CreatePackageRequest struct {
	Identifier string `json:"identifier"`
	Ecosystem  string `json:"ecosystem"`
}

// CreatePackageResponse is the response body for CreatePackage.
type CreatePackageResponse struct {
	ID            int32  `json:"id"`
	Identifier    string `json:"identifier"`
	Ecosystem     string `json:"ecosystem"`
	AlreadyExists bool   `json:"already_exists"`
}

// --- Reproducible build handler types ---

// RegisterReproducibleBuildRequest is the payload for registering a reproducible build.
type RegisterReproducibleBuildRequest struct {
	Ecosystem  string `json:"ecosystem"`
	Identifier string `json:"identifier"`
	Version    string `json:"version"`
	// Tarball is the base64-encoded .tgz artifact.
	Tarball string `json:"tarball"`
}

// RegisterReproducibleBuildResponse is returned on success.
type RegisterReproducibleBuildResponse struct {
	PackageVersionID int32 `json:"package_version_id"`
}
