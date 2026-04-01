//nolint:forbidigo // CLI tool uses fmt.Print* for user-facing output
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

const usageText = `spr-cli: project dependency management for the Secure Package Registry

Usage:
  spr-cli [global flags] <command> [command flags]

Commands:
  upload       Upload a package.json or package-lock.json to create/update a project
  list         List all your projects
  show         Show a single project
  deps         List dependencies for a project
  summary      Show security summary for a project
  delete       Delete a project

Global Flags:
`

func main() {
	var (
		apiURL = flag.String("api", envOr("SPR_API_URL", "http://localhost:7001"), "API base URL (or set SPR_API_URL)")
		apiKey = flag.String("key", envOr("SPR_API_KEY", ""), "API key (or set SPR_API_KEY)")
	)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageText)
		flag.PrintDefaults()
	}

	// Parse global flags only up to the first non-flag argument.
	flag.Parse()

	if *apiKey == "" {
		fatal("API key is required. Set SPR_API_KEY or use -key flag.")
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	c := &client{
		baseURL: strings.TrimRight(*apiURL, "/"),
		apiKey:  *apiKey,
		http:    &http.Client{Timeout: 60 * time.Second},
	}

	cmd, cmdArgs := args[0], args[1:]

	var err error
	switch cmd {
	case "upload":
		err = cmdUpload(c, cmdArgs)
	case "list":
		err = cmdList(c)
	case "show":
		err = cmdShow(c, cmdArgs)
	case "deps":
		err = cmdDeps(c, cmdArgs)
	case "summary":
		err = cmdSummary(c, cmdArgs)
	case "delete":
		err = cmdDelete(c, cmdArgs)
	default:
		fatal("Unknown command: %s", cmd)
	}

	if err != nil {
		fatal("%v", err)
	}
}

// ---------- Commands ----------

func cmdUpload(c *client, args []string) error {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	name := fs.String("name", "", "Project name (required)")
	filePath := fs.String("file", "", "Path to package.json or package-lock.json (required)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: spr-cli upload -name <project> -file <path>")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *name == "" || *filePath == "" {
		fs.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*filePath) //nolint:gosec // user-provided CLI argument
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	body := map[string]string{
		"name": *name,
		"file": string(data),
	}

	var result map[string]any
	if err := c.do("POST", "/api/v1/projects", body, &result); err != nil {
		return err
	}

	fmt.Printf("Project created/updated successfully\n")
	fmt.Printf("  ID:          %.0f\n", result["id"])
	fmt.Printf("  Name:        %s\n", result["name"])
	fmt.Printf("  Source Type: %s\n", result["source_type"])
	fmt.Printf("  Total Deps:  %.0f\n", result["total_deps"])
	fmt.Printf("  Direct Deps: %.0f\n", result["direct_deps"])

	return nil
}

func cmdList(c *client) error {
	var result struct {
		Items []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			SourceType string `json:"source_type"`
			CreatedAt  string `json:"created_at"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"items"`
	}

	if err := c.do("GET", "/api/v1/projects", nil, &result); err != nil {
		return err
	}

	if len(result.Items) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	twPrintln(tw, "ID\tNAME\tSOURCE TYPE\tCREATED\tUPDATED")
	for _, p := range result.Items {
		twPrintf(tw, "%d\t%s\t%s\t%s\t%s\n",
			p.ID, p.Name, p.SourceType, truncTime(p.CreatedAt), truncTime(p.UpdatedAt))
	}
	return tw.Flush()
}

func cmdShow(c *client, args []string) error {
	if len(args) != 1 {
		fatal("Usage: spr-cli show <project-id>")
	}
	projectID := args[0]

	var result map[string]any
	if err := c.do("GET", "/api/v1/projects/"+projectID, nil, &result); err != nil {
		return err
	}

	fmt.Printf("ID:          %.0f\n", result["id"])
	fmt.Printf("Name:        %s\n", result["name"])
	fmt.Printf("Source Type: %s\n", result["source_type"])
	if v, ok := result["created_at"]; ok && v != nil {
		fmt.Printf("Created At:  %s\n", v)
	}
	if v, ok := result["updated_at"]; ok && v != nil {
		fmt.Printf("Updated At:  %s\n", v)
	}

	return nil
}

func cmdDeps(c *client, args []string) error {
	fs := flag.NewFlagSet("deps", flag.ExitOnError)
	depType := fs.String("type", "", "Filter by dependency type: direct or transitive")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: spr-cli deps [-type direct|transitive] <project-id>")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}
	projectID := fs.Arg(0)

	path := "/api/v1/projects/" + projectID + "/dependencies"
	if *depType != "" {
		path += "?type=" + *depType
	}

	var result struct {
		Items []struct {
			ID                int    `json:"id"`
			Identifier        string `json:"identifier"`
			Ecosystem         string `json:"ecosystem"`
			Version           string `json:"version"`
			DependencyType    string `json:"dependency_type"`
			VersionConstraint string `json:"version_constraint"`
		} `json:"items"`
	}

	if err := c.do("GET", path, nil, &result); err != nil {
		return err
	}

	if len(result.Items) == 0 {
		fmt.Println("No dependencies found.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	twPrintln(tw, "ID\tPACKAGE\tVERSION\tTYPE\tCONSTRAINT")
	for _, d := range result.Items {
		twPrintf(tw, "%d\t%s\t%s\t%s\t%s\n",
			d.ID, d.Identifier, d.Version, d.DependencyType, d.VersionConstraint)
	}
	return tw.Flush()
}

func cmdSummary(c *client, args []string) error {
	if len(args) != 1 {
		fatal("Usage: spr-cli summary <project-id>")
	}
	projectID := args[0]

	var result struct {
		ProjectID int `json:"project_id"`
		Summary   []struct {
			DependencyType string `json:"dependency_type"`
			Total          int    `json:"total"`
			HasAttestation int    `json:"has_attestation"`
			HasOssRebuild  int    `json:"has_oss_rebuild"`
			BehaviorPassed int    `json:"behavior_passed"`
		} `json:"summary"`
	}

	if err := c.do("GET", "/api/v1/projects/"+projectID+"/summary", nil, &result); err != nil {
		return err
	}

	if len(result.Summary) == 0 {
		fmt.Println("No dependency data found.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	twPrintln(tw, "TYPE\tTOTAL\tATTESTATION\tOSS REBUILD\tBEHAVIOR OK")
	for _, s := range result.Summary {
		twPrintf(tw, "%s\t%d\t%d\t%d\t%d\n",
			s.DependencyType, s.Total, s.HasAttestation, s.HasOssRebuild, s.BehaviorPassed)
	}
	return tw.Flush()
}

func cmdDelete(c *client, args []string) error {
	if len(args) != 1 {
		fatal("Usage: spr-cli delete <project-id>")
	}
	projectID := args[0]

	if err := c.doRaw("DELETE", "/api/v1/projects/"+projectID, nil, http.StatusNoContent); err != nil {
		return err
	}

	fmt.Println("Project deleted successfully.")
	return nil
}

// ---------- HTTP Client ----------

type client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// do makes a JSON request and decodes the JSON response into out.
func (c *client) do(method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader) //nolint:noctx // CLI tool, no long-lived context needed
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() //nolint:errcheck // best-effort close
	}()

	if resp.StatusCode >= 400 {
		return decodeError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// doRaw makes a request and checks for the expected status code. No JSON decoding.
func (c *client) doRaw(method, path string, body any, expectStatus int) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader) //nolint:noctx // CLI tool
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() //nolint:errcheck // best-effort close
	}()

	if resp.StatusCode != expectStatus {
		return decodeError(resp)
	}

	return nil
}

func decodeError(resp *http.Response) error {
	var errBody map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
		if msg, ok := errBody["error"].(string); ok {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
		}
		return fmt.Errorf("API error %d: %v", resp.StatusCode, errBody)
	}
	return fmt.Errorf("API error %d", resp.StatusCode)
}

// ---------- Helpers ----------

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func truncTime(s string) string {
	if len(s) > 19 {
		return s[:19]
	}
	return s
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// twPrintln writes a line to a tabwriter, ignoring the error (stdout write failures are fatal anyway).
func twPrintln(w *tabwriter.Writer, s string) {
	_, _ = fmt.Fprintln(w, s)
}

// twPrintf writes formatted output to a tabwriter, ignoring the error.
func twPrintf(w *tabwriter.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
