//nolint:forbidigo // CLI tool uses fmt.Print* for user-facing output
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

const usage = `rep-build: reproducible build tool for the secure package registry

Usage:
  go run ./cmd/rep-build [flags] <ecosystem>/<identifier>/<version>

Example:
  go run ./cmd/rep-build npmjs/typescript/5.9.3

Flags:`

// buildConfig is the parsed build.yaml for a package version.
type buildConfig struct {
	// Template is the build template to use: "npm", "pnpm", "yarn", or "custom".
	// For "custom" a Containerfile must exist alongside build.yaml.
	Template string `yaml:"template"`
	// All other keys are passed verbatim as template variables.
	Vars map[string]interface{} `yaml:",inline"`
}

// registerRequest mirrors the internal API payload.
type registerRequest struct {
	Ecosystem  string `json:"ecosystem"`
	Identifier string `json:"identifier"`
	Version    string `json:"version"`
	Tarball    string `json:"tarball"`
}

func main() {
	var (
		apiURL    = flag.String("api", "http://localhost:8081", "Internal API base URL")
		configDir = flag.String("config-dir", "./cmd/rep-build", "Directory containing build configs and templates")
		keep      = flag.Bool("keep", false, "Skip cleanup (for debugging)")
	)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	parts := strings.SplitN(flag.Arg(0), "/", 3)
	if len(parts) != 3 {
		fmt.Fprintf(os.Stderr, "error: argument must be <ecosystem>/<identifier>/<version>, got %q\n", flag.Arg(0))
		os.Exit(1)
	}
	ecosystem, identifier, version := parts[0], parts[1], parts[2]

	if err := run(context.Background(), *apiURL, *configDir, ecosystem, identifier, version, *keep); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, apiURL, configDir, ecosystem, identifier, version string, keep bool) error {
	// Resolve the build.yaml path.
	// Identifier may contain "@scope/name" — use it directly as sub-path.
	buildDir := filepath.Join(configDir, ecosystem, identifier, version)
	buildYAMLPath := filepath.Join(buildDir, "build.yaml")

	fmt.Printf("==> Loading build config: %s\n", buildYAMLPath)

	cfg, err := loadBuildConfig(buildYAMLPath)
	if err != nil {
		return fmt.Errorf("load build config: %w", err)
	}

	// Render the Containerfile.
	containerfileContent, err := renderContainerfile(configDir, buildDir, cfg)
	if err != nil {
		return fmt.Errorf("render Containerfile: %w", err)
	}

	// Write Containerfile to a temp file.
	tmpCF, err := os.CreateTemp("", "rep-build-containerfile-*")
	if err != nil {
		return fmt.Errorf("create temp Containerfile: %w", err)
	}
	tmpCFPath := tmpCF.Name()
	if _, err := tmpCF.WriteString(containerfileContent); err != nil {
		_ = tmpCF.Close()
		return fmt.Errorf("write temp Containerfile: %w", err)
	}
	if err := tmpCF.Close(); err != nil {
		return fmt.Errorf("close temp Containerfile: %w", err)
	}

	// Derive a stable image name.
	safeID := strings.ReplaceAll(identifier, "/", "-")
	safeID = strings.ReplaceAll(safeID, "@", "")
	imageName := fmt.Sprintf("rep-build-%s-%s", safeID, version)
	containerName := imageName

	defer func() {
		if keep {
			fmt.Printf("==> --keep: skipping cleanup (Containerfile: %s, image: %s)\n", tmpCFPath, imageName)
			return
		}
		fmt.Println("==> Cleaning up...")
		//nolint:errcheck // best-effort cleanup
		_ = runCmd("podman", "rm", "-f", containerName)
		//nolint:errcheck // best-effort cleanup
		_ = runCmd("podman", "rmi", "-f", imageName)
		//nolint:errcheck // best-effort cleanup
		_ = os.Remove(tmpCFPath)
	}()

	// Build the container image.
	fmt.Printf("==> Building image %s\n", imageName)
	if err := runInteractive(ctx, "podman", "build",
		"-f", tmpCFPath,
		"-t", imageName,
		buildDir,
	); err != nil {
		return fmt.Errorf("podman build: %w", err)
	}

	// Create a temp dir to extract the artifact into.
	tmpExtractDir, err := os.MkdirTemp("", "rep-build-extract-*")
	if err != nil {
		return fmt.Errorf("create temp extract dir: %w", err)
	}
	defer func() {
		if !keep {
			//nolint:errcheck // best-effort cleanup
			_ = os.RemoveAll(tmpExtractDir)
		}
	}()

	// Run the container and copy /out to the host.
	fmt.Printf("==> Running container to extract artifact\n")
	if err := runCmd("podman", "create", "--name", containerName, imageName); err != nil {
		return fmt.Errorf("podman create: %w", err)
	}
	if err := runCmd("podman", "cp", containerName+":/out/.", tmpExtractDir); err != nil {
		return fmt.Errorf("podman cp: %w", err)
	}

	// Find the .tgz artifact.
	tgzPath, err := findTGZ(tmpExtractDir)
	if err != nil {
		return fmt.Errorf("find artifact: %w", err)
	}
	fmt.Printf("==> Found artifact: %s\n", tgzPath)

	// Read and base64-encode the tarball.
	tarballBytes, err := os.ReadFile(tgzPath) //nolint:gosec // path comes from temp dir we control
	if err != nil {
		return fmt.Errorf("read artifact: %w", err)
	}
	tarballB64 := base64.StdEncoding.EncodeToString(tarballBytes)

	// Register with internal API.
	fmt.Printf("==> Registering with internal API at %s\n", apiURL)
	if err := registerBuild(ctx, apiURL, ecosystem, identifier, version, tarballB64); err != nil {
		return fmt.Errorf("register build: %w", err)
	}

	fmt.Printf("==> Done! Reproducible build registered for %s/%s@%s\n", ecosystem, identifier, version)
	return nil
}

// loadBuildConfig reads and parses a build.yaml file.
func loadBuildConfig(path string) (*buildConfig, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from user CLI argument
	if err != nil {
		return nil, err
	}

	// First pass: get the template name.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	tplName, _ := raw["template"].(string)
	if tplName == "" {
		return nil, fmt.Errorf("build.yaml missing required field: template")
	}

	delete(raw, "template")
	return &buildConfig{
		Template: tplName,
		Vars:     raw,
	}, nil
}

// renderContainerfile renders the appropriate Containerfile template.
func renderContainerfile(configDir, buildDir string, cfg *buildConfig) (string, error) {
	var tmplText string

	if cfg.Template == "custom" {
		// Custom template: read Containerfile from buildDir.
		data, err := os.ReadFile(filepath.Join(buildDir, "Containerfile")) //nolint:gosec // path comes from user CLI argument
		if err != nil {
			return "", fmt.Errorf("read custom Containerfile: %w", err)
		}
		tmplText = string(data)
	} else {
		// Named template: look in configDir/templates/<name>.Containerfile.
		tmplPath := filepath.Join(configDir, "templates", cfg.Template+".Containerfile")
		data, err := os.ReadFile(tmplPath) //nolint:gosec // path comes from user CLI argument
		if err != nil {
			return "", fmt.Errorf("read template %q: %w", tmplPath, err)
		}
		tmplText = string(data)
	}

	// Use a custom "default" func so templates can write {{ index . "key" | default "fallback" }}.
	funcMap := template.FuncMap{
		"default": func(def, val interface{}) interface{} {
			if val == nil {
				return def
			}
			s, ok := val.(string)
			if ok && s == "" {
				return def
			}
			return val
		},
	}

	t, err := template.New("Containerfile").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, cfg.Vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// runCmd runs a command, streaming stdout/stderr to the terminal.
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runInteractive runs a command with context, streaming stdout/stderr to the terminal.
func runInteractive(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// findTGZ finds the first .tgz file in a directory.
func findTGZ(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tgz") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .tgz file found in %s", dir)
}

// registerBuild calls POST /api/v1/internal/reproducible-builds.
func registerBuild(ctx context.Context, apiURL, ecosystem, identifier, version, tarballB64 string) error {
	payload := registerRequest{
		Ecosystem:  ecosystem,
		Identifier: identifier,
		Version:    version,
		Tarball:    tarballB64,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(apiURL, "/") + "/api/v1/internal/reproducible-builds"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		//nolint:errcheck // best-effort close on response body
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]interface{}
		if decErr := json.NewDecoder(resp.Body).Decode(&errBody); decErr == nil {
			return fmt.Errorf("API returned %d: %v", resp.StatusCode, errBody)
		}
		return fmt.Errorf("API returned %d", resp.StatusCode)
	}

	return nil
}
