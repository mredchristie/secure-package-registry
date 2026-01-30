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
  bunx markdownlint-cli2 "**/*.md" --fix


check:
  #!/usr/bin/env bash
  set -e

  # Check Go formatting
  go tool gofumpt -l .

  # Check Go linting
  go tool golangci-lint run

  # Check JS/TS
  bunx @biomejs/biome check

  # Check markdown
  bunx markdownlint-cli2 "**/*.md"

test:
  go test ./...

help:
  @just --help
