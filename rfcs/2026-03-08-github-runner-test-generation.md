+++
title = "GitHub Runner Test Generation"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-03-08"
depends_on = ["2026-03-07-behavioral-analysis-pipeline.md"]
+++

## Summary

`spr-gh-runner/` is a standalone Go module implementing the GitHub Actions side of the behavioral analysis pipeline. It
contains a test scaffolding generator (`test-generator` CLI) and two workflows: `collect-behavior.yml` which
orchestrates test execution under Tracee monitoring, and `build-release.yml` which publishes the tooling as release
assets.

This RFC documents the design decisions made while building this component, refactored from the HackEurope prototype in
`https://github.com/acheong08/hackeurope-spr/`.

## Decisions

### Standalone module with release-based distribution

`spr-gh-runner` is a separate Go module. The binary and templates are published as GitHub Release assets under a rolling
`tools-latest` tag. `build-release.yml` rebuilds on pushes to `main` that touch source files, deletes the existing
release, and creates a new one with the fresh binary and templates archive. The analysis workflow downloads these at
runtime rather than building from source.

This keeps the runner repository lightweight. No Go toolchain needed at analysis time and gives `be-runner` a stable
download URL without version management overhead.

Concerns: Race condition between deletion and recreation?

### Four test categories covering primary attack surfaces

Malicious packages need their code to execute to be effective. An attacker's goal is to ensure their payload runs with
minimal or no user interaction. This constrains the realistic attack surfaces to entry points that trigger
automatically:

1. **Install** — `npm install` triggers lifecycle scripts (`preinstall`, `postinstall`, `prepare`). This is the most
   common real-world attack vector because it runs before the developer ever imports the package.
2. **Import** — `require()` or `import` triggers top-level side effects (IIFEs, module initialization). The second most
   common vector; the payload fires on first use.
3. **Prototype pollution** — Importing the package may silently modify `Object.prototype`. The test records baseline
   prototype properties, imports the package, then detects and invokes any newly added properties. This catches a class
   of attacks that are invisible without explicit detection.
4. **CLI** — Conditional, only generated when the package declares a `bin` field. Runs `npx <package> --version` with a
   30-second timeout.

Together these four categories cover the vast majority of observed malicious npm samples. Exotic vectors (e.g. Patching
Node.js internals, delayed timers, worker threads) exist but require deliberate user action or unusual conditions to
trigger, making them both rarer and lower-priority for automated detection.

### Workflow design

`collect-behavior.yml` is triggered via `workflow_dispatch` with three inputs: `package`, `version`, and `registry`
(choice of `gitea` or `npm`, defaulting to `gitea`). A registry preset system maps the choice to a URL, owner, and
`NPM_CONFIG_REGISTRY` value, so callers pass a preset name rather than raw configuration.

The workflow:

- Downloads pre-built tooling from the `tools-latest` release
- Runs `test-generator` to produce test scaffolding
- Creates a `node:20` Docker container with `--network host`
- Starts Tracee on the host (see below)
- Copies test files in via `docker cp` and executes them sequentially (install, import, prototype, CLI)
- Collects artifacts

Sequential execution is intentional: since Tracee captures all container activity in a single stream, running tests one
at a time makes it possible to attribute events to specific test phases by timestamp.

### Host-side Tracee with container scoping

Tracee runs on the GitHub Actions host, not inside the Docker container. It is scoped via `--scope container` to only
capture events from the analysis container. Output is written to `/tmp/tracee-out/behavior.jsonl` on the host
filesystem.

This is the critical security property: even if a malicious package achieves code execution inside the container, it
cannot tamper with or suppress the behavioral data being recorded outside it.

Captured events: `execve`, `execveat`, `open`, `openat`, `connect`, `net_packet_dns_request`. Covers process execution,
file access, network connections, and DNS queries.

Future consideration: Capturing all syscalls and comparing behavior between reproducible build and NPM package.

### Fail-open artifact collection

All test steps suppress failures with `|| echo "..."`. Tracee stop, container cleanup, and artifact upload use
`if: always()`. A separate debug artifact (7-day retention) is uploaded on failure; the main behavioral artifact gets
30-day retention.

This ensures behavioral data is captured even when malicious packages crash, hang, or intentionally fail — which is
itself a useful signal.

### Package JSON generation via `json.MarshalIndent`

`package.json` files are generated programmatically rather than through text templates. This avoids escaping issues with
scoped package names (`@scope/name` contains characters that are problematic in template-rendered JSON). JS test scripts
are still rendered through Go's `text/template` engine.

### Module type detection heuristic

Detection follows a conservative priority chain:

- `"type": "module"` in package.json → ESM
- `"module"` field present → ESM
- `"exports"` field present → Dual, treated as CommonJS
- Default → CommonJS

Dual-support packages are treated as CommonJS to maximize compatibility. ESM/CJS branching logic lives in the JS
templates via `{{if eq .ModuleType "module"}}`, not in Go code.

### CLI test marker pattern

Rather than generating a test script, the CLI test generator creates a `HAS_CLI` marker file containing the binary name.
The workflow checks for the `cli/HAS_CLI` file's existence and runs `npx` directly. This separates "what to test"
(generator's responsibility) from "how to test" (workflow's responsibility).

## Open Questions

1. Re-adding registry authentication for non-public registries
2. Whether scoped package name normalization should be shared between Go and the workflow shell (currently duplicated)
3. Testing the ESM code path for dual-support packages
