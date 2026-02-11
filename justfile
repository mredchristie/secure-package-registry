set dotenv-load

_default:
  @just --list

fix:
  #!/usr/bin/env bash
  # Fix markdown formatting with Prettier
  bunx prettier --write "**/*.md"

  # Format Go
  go tool gofumpt -l -w .

  # Lint Go
  go tool golangci-lint run --fix

  # Lint JS/TS
  bunx @biomejs/biome check --write

  # Lint and fix markdown with markdownlint-cli2
  git ls-files "*.md" | xargs -r bunx markdownlint-cli2 --fix


check:
  #!/usr/bin/env bash
  set -e

  # Check Go formatting
  go tool gofumpt -l .

  # Check Go linting
  go tool golangci-lint run

  # Check JS/TS
  bunx @biomejs/biome check

  # Run checks in dashboard
  cd ./dashboard-ui/ && bun ci && bun run check && cd ..

  # Check markdown
  bunx prettier -c "**/*.md" --config ./.prettierrc
  git ls-files "*.md" | xargs -r bunx markdownlint-cli2

test:
  go test ./...

generate:
  #!/usr/bin/env bash
  go generate ./...
  # Generate API documentation
  go run ./cmd/apigen
  # TODO: Generate Typescript bindings from API docs

help:
  @just --help
