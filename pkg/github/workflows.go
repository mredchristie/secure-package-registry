package github

import (
	"context"
	"fmt"
	"time"
)

// TriggerWorkflow dispatches a workflow_dispatch event and returns the created run.
//
// The GitHub API returns a 200 with run metadata when return_run_details is true.
// This was added in Feb 2026: https://github.blog/changelog/2026-02-19-workflow-dispatch-api-now-returns-run-ids
// Prior to this change the endpoint returned 204 No Content with no body.
func (c *Client) TriggerWorkflow(ctx context.Context, workflowFile string, inputs map[string]string) (*WorkflowRunCreated, error) {
	payload := map[string]any{
		"ref":                "main",
		"inputs":             inputs,
		"return_run_details": true,
	}
	path := fmt.Sprintf("actions/workflows/%s/dispatches", workflowFile)
	var run WorkflowRunCreated
	if err := c.postJSON(ctx, path, payload, &run); err != nil {
		return nil, fmt.Errorf("triggering workflow: %w", err)
	}
	if run.RunID == 0 {
		return nil, fmt.Errorf("API returned run ID 0: workflow may not have triggered")
	}
	return &run, nil
}

// GetWorkflowRun fetches the current status of a workflow run.
func (c *Client) GetWorkflowRun(ctx context.Context, runID int64) (result *WorkflowRun, err error) {
	path := fmt.Sprintf("actions/runs/%d", runID)
	var run WorkflowRun
	if err := c.doJSON(ctx, "GET", path, nil, &run); err != nil {
		return nil, fmt.Errorf("getting workflow run: %w", err)
	}
	return &run, nil
}

// PollWorkflowRun polls GetWorkflowRun at the given interval until the run
// reaches a terminal state (Status == "completed") or the context is cancelled.
// The caller controls timeout via context.WithTimeout.
func (c *Client) PollWorkflowRun(ctx context.Context, runID int64, interval time.Duration) (*WorkflowRun, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		run, err := c.GetWorkflowRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if run.Status == "completed" {
			return run, nil
		}
		c.log.Debug().Int64("run_id", runID).Str("status", run.Status).Msg("workflow still running")

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("polling workflow run %d: %w", runID, ctx.Err())
		case <-ticker.C:
		}
	}
}
