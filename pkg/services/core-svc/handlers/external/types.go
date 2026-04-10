package external

// ErrorResponse is the standard error envelope returned by all handlers.
type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Admin: requests ---

// AddPackageRequest is the request body for AddPackage.
type AddPackageRequest struct {
	Identifier string `json:"identifier"`
	Ecosystem  string `json:"ecosystem"`
}

// TriggerScanRequest is the optional request body for TriggerScan.
type TriggerScanRequest struct {
	Version string `json:"version"`
}

// --- Admin: responses ---

// PackageListItem is a single item returned by ListPackages.
type PackageListItem struct {
	ID            int32   `json:"id"`
	Identifier    string  `json:"identifier"`
	Ecosystem     string  `json:"ecosystem"`
	LatestVersion *string `json:"latest_version"`
}

// PackageListResponse is the response body for ListPackages.
type PackageListResponse struct {
	Items []PackageListItem `json:"items"`
}

// AddPackageResponse is the response body for AddPackage.
type AddPackageResponse struct {
	ID            int32  `json:"id"`
	Identifier    string `json:"identifier"`
	Ecosystem     string `json:"ecosystem"`
	AlreadyExists bool   `json:"already_exists"`
}

// ListVersionsResponse is the response body for ListVersions.
type ListVersionsResponse struct {
	Identifier    string   `json:"identifier"`
	Ecosystem     string   `json:"ecosystem"`
	LatestVersion *string  `json:"latest_version"`
	Versions      []string `json:"versions"`
}

// TriggerScanResponse is the response body for TriggerScan.
type TriggerScanResponse struct {
	TaskID     int32  `json:"task_id"`
	Identifier string `json:"identifier"`
	Ecosystem  string `json:"ecosystem"`
	Version    string `json:"version"`
	Status     string `json:"status"`
}

// TaskListItem is a single item returned by ListTasks.
type TaskListItem struct {
	ID            int32   `json:"id"`
	Identifier    string  `json:"identifier"`
	Ecosystem     string  `json:"ecosystem"`
	Version       string  `json:"version"`
	Source        string  `json:"source"`
	Status        string  `json:"status"`
	FailureReason *string `json:"failure_reason,omitempty"`
	HasArtifact   bool    `json:"has_artifact"`
	StartedAt     *string `json:"started_at,omitempty"`
	CompletedAt   *string `json:"completed_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// TaskListResponse is the response body for ListTasks.
type TaskListResponse struct {
	Items []TaskListItem `json:"items"`
}

// --- Project: requests ---

// UploadProjectRequest is the request body for uploading/updating a project.
type UploadProjectRequest struct {
	Name string `json:"name"`
	File string `json:"file"` // raw contents of package.json or package-lock.json
}

// --- Project: responses ---

// UploadProjectResponse is the response body for UploadProject.
// Returns immediately with status "pending" — processing happens asynchronously.
type UploadProjectResponse struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ProjectListItem is a single item returned by ListProjects.
type ProjectListItem struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ProjectListResponse is the response body for ListProjects.
type ProjectListResponse struct {
	Items []ProjectListItem `json:"items"`
}

// GetProjectResponse is the response body for GetProject.
type GetProjectResponse struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// DependencyListItem is a single item returned by ListDependencies.
// Check fields are nil when checks haven't been run yet, otherwise true/false.
type DependencyListItem struct {
	ID                int32  `json:"id"`
	Identifier        string `json:"identifier"`
	Ecosystem         string `json:"ecosystem"`
	Version           string `json:"version"`
	DependencyType    string `json:"dependency_type"`
	VersionConstraint string `json:"version_constraint,omitempty"`
	HasAttestation    *bool  `json:"has_attestation"`
	HasOssRebuild     *bool  `json:"has_oss_rebuild"`
	BehaviorPassed    *bool  `json:"behavior_passed"`
}

// DependencyListResponse is the response body for ListDependencies.
type DependencyListResponse struct {
	Items []DependencyListItem `json:"items"`
}

// SummaryRow is a single row in the project summary.
type SummaryRow struct {
	DependencyType string `json:"dependency_type"`
	Total          int32  `json:"total"`
	HasAttestation int32  `json:"has_attestation"`
	HasOssRebuild  int32  `json:"has_oss_rebuild"`
	BehaviorPassed int32  `json:"behavior_passed"`
}

// ProjectSummaryResponse is the response body for GetSummary.
type ProjectSummaryResponse struct {
	ProjectID int          `json:"project_id"`
	Summary   []SummaryRow `json:"summary"`
}
