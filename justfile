set dotenv-load

_default:
  @just --list

fix:
  #!/usr/bin/env bash
  # Fix markdown formatting with Prettier
  bunx prettier --write "**/*.md" --config ./.prettierrc

  go fix ./...

  go tool gofumpt -l -w .

  go tool golangci-lint run --fix

  cd ./dashboard-ui/ && bun ci && bun run format && cd ..
  cd ./home-ui/ && bun ci && bun run format && cd ..

  git ls-files "*.md" | xargs -r bunx markdownlint-cli2 --fix


check:
  #!/usr/bin/env bash
  set -e

  go tool gofumpt -e -l .

  go tool golangci-lint run

  cd ./dashboard-ui/ && bun ci && bun run check && cd ..
  cd ./home-ui/ && bun ci && bun run check && cd ..

  # Check markdown
  bunx prettier -c "**/*.md" --config ./.prettierrc
  git ls-files "*.md" | xargs -r bunx markdownlint-cli2

test:
  go test ./...

integration:
  go test -tags integration -v -count=1 -timeout 15m ./...

generate:
  #!/usr/bin/env bash
  go generate ./...
  # Generate API documentation
  go run ./cmd/apigen
  # TODO: Generate Typescript bindings from API docs

help:
  @just --help
