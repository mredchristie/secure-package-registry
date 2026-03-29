# Reproducible Build Tool

CLI tool for building packages from source and registering them with the secure package registry's internal API.

## Overview

`rep-build` reads build configurations from `cmd/rep-build/<ecosystem>/<package>/<version>/build.yaml`, renders the appropriate Containerfile template, builds the package using Podman, and registers the resulting tarball via the internal API.

## Prerequisites

- Go 1.25+
- Podman (rootless)
- Internal API running (see `compose.yml`)

## Usage

```bash
go run ./cmd/rep-build [flags] <ecosystem>/<identifier>/<version>
```

### Examples

Build TypeScript 5.9.3:

```bash
go run ./cmd/rep-build npm/typescript/5.9.3
```

Build with custom API endpoint:

```bash
go run ./cmd/rep-build --api http://localhost:20001 npm/typescript/5.9.3
```

Keep containers/images for debugging:

```bash
go run ./cmd/rep-build --keep npm/typescript/5.9.3
```

### Flags

- `--api`: Internal API base URL (default: `http://localhost:20001`)
- `--config-dir`: Directory containing build configs and templates (default: `./cmd/rep-build`)
- `--keep`: Skip cleanup of containers, images, and temp files (for debugging)

## Build Configuration

Each package version has a `build.yaml` in `cmd/rep-build/<ecosystem>/<identifier>/<version>/`:

```yaml
template: npm # Required: npm, yarn, pnpm, or custom
repo_url: https://github.com/... # Required: Git repository URL
commit_sha: abc123 # Required: Specific commit to build

# Optional template variables:
node_image: node:23-alpine # Base image (default: node:23-alpine)
githead_injection: true # Inject gitHead into package.json
```

### Custom Templates

For non-standard builds, use `template: custom` and include a `Containerfile` alongside `build.yaml`:

```yaml
template: custom
repo_url: https://github.com/prettier/prettier.git
commit_sha: 7584432401a47a26943dd7a9ca9a8e032ead7285
```

The custom `Containerfile` can use Go template syntax with variables from `build.yaml`:

```dockerfile
FROM {{ .node_image | default "node:23-alpine" }}
RUN git clone {{ .repo_url }} /app
WORKDIR /app
RUN git checkout {{ .commit_sha }}
RUN npm ci && npm run build
RUN mkdir -p /out && npm pack --pack-destination /out
CMD ["sh", "-c", "ls /out/*.tgz"]
```

## Available Templates

### npm

Standard npm-based builds. Supports:

- `node_image`: Base image (default: node:23-alpine)
- `repo_url`, `commit_sha`: Git source
- `githead_injection`: Inject gitHead field

### yarn

Standard yarn-based builds. Supports:

- `node_image`: Base image (default: node:23-alpine)
- `repo_url`, `commit_sha`: Git source

### pnpm

Standard pnpm-based builds. Similar to npm template.

### custom

Read `Containerfile` from the package's build directory. Full Go template support.

## Testing

### Manual End-to-End Test

1. Start the services:

   ```bash
   podman compose up -d
   ```

2. Verify internal API is reachable:

   ```bash
   curl http://localhost:20001/api/v1/internal/reproducible-builds
   # Should return 405 Method Not Allowed (not 404 or connection refused)
   ```

3. Run a build:

   ```bash
   go run ./cmd/rep-build npm/typescript/5.9.3
   ```

4. Verify the build was registered (check the API or database).

### Testing Custom Builds

For packages with non-standard build processes:

1. Create the directory structure:

   ```bash
   mkdir -p cmd/rep-build/npm/<package>/<version>
   ```

2. Write `build.yaml` with `template: custom`

3. Write `Containerfile` with the custom build steps

4. Test with `--keep` flag to inspect containers if needed:

   ```bash
   go run ./cmd/rep-build --keep npm/<package>/<version>
   ```

## Architecture

The tool follows this flow:

1. **Parse CLI**: Extract ecosystem/identifier/version from argument
2. **Load Config**: Read `build.yaml` from the package directory
3. **Render Containerfile**: Apply template (built-in or custom) with config variables
4. **Build Image**: Run `podman build` with the rendered Containerfile
5. **Create Container**: `podman create` to set up the container
6. **Extract Artifact**: `podman cp` to copy `/out/*.tgz` from container
7. **Register**: Base64-encode the tarball and POST to `/api/v1/internal/reproducible-builds`
8. **Cleanup**: Remove temp files, container, and image (unless `--keep`)

## Notes

- The tool intentionally does **not** include diffoscope comparison — that's handled by a separate layer per the RFC
- All line ending normalization (unix2dos) is also handled elsewhere, not in this tool
- The internal API is expected to be running on port 20001 (or as specified by `--api`)
