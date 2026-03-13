package github

import "time"

// WorkflowRunCreated is returned by the dispatch endpoint when return_run_details is true.
// See: https://github.blog/changelog/2026-02-19-workflow-dispatch-api-now-returns-run-ids
type WorkflowRunCreated struct {
	RunID   int64  `json:"workflow_run_id"`
	RunURL  string `json:"run_url"`
	HTMLURL string `json:"html_url"`
}

// WorkflowRun represents the status of a GitHub Actions workflow run.
type WorkflowRun struct {
	ID         int64     `json:"id"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	HTMLURL    string    `json:"html_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Artifact represents a workflow run artifact.
type Artifact struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SizeInBytes int64  `json:"size_in_bytes"`
	Expired     bool   `json:"expired"`
}

type artifactList struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}
