+++
title = "Project Structure"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-03-08"
supersedes = ["2026-02-08-project-structure.md"]
+++

## Summary

Updated project layout reflecting the service refactoring into `pkg/services/`, thin `cmd/` wrappers, a unified binary,
and the `internal/` package hierarchy. Supersedes the original project structure RFC.

## Directory Structure

```plain
├── cmd/                         # Thin entry points (~40 lines each)
│   ├── be-runner/               # Behavioral analysis runner
│   ├── core-svc/                # Central data service
│   ├── poller/                  # Registry polling
│   ├── seed/                    # Database seeder (dev only)
│   ├── spr/                     # Unified binary (all services via errgroup)
│   └── apigen/                  # OpenAPI spec generator
├── pkg/                         # Shared libraries and service logic
│   ├── services/                # Service implementations
│   │   ├── deps.go              # Shared Deps struct (Pool, Valkey, Config)
│   │   ├── core-svc/            # core-svc: HTTP servers + AMQP consumer
│   │   │   ├── service.go       # Start() entry point
│   │   │   ├── server/          # HTTP server setup
│   │   │   └── handlers/        # external/ and private/ HTTP handlers
│   │   ├── be-runner/           # be-runner: collection pipeline
│   │   └── package-watcher/     # Poller: registry version detection
│   ├── config/                  # Configuration loading
│   ├── gitea/                   # Gitea API client + mirroring
│   ├── github/                  # GitHub API client (workflows, artifacts)
│   ├── npm/                     # npm registry client + dependency resolver
│   ├── pkgdb/                   # Higher-level DB helpers on top of sqlc
│   └── logger/                  # Zerolog setup
├── internal/                    # Project-internal packages (not importable)
│   ├── gen/coredb/              # sqlc-generated DB code (schema, queries, models)
│   ├── messages/                # AMQP message types (gob-encoded structs)
│   └── valkey/                  # Valkey client wrapper
├── dashboard-ui/                # SvelteKit frontend
├── home-ui/                     # Landing page
├── infra/                       # Infrastructure (migrations, Cloudflare workers)
│   └── migrations/              # Postgres migrations
├── docs/svc/                    # Generated OpenAPI spec
├── context-references/          # Hackathon prototype code (read-only reference)
├── rfcs/                        # RFC documents
├── compose.yml                  # Service topology
└── justfile                     # Task runner (fix, check, test, generate, integration)
```

## Service Architecture

Each service follows a two-layer pattern:

- **`cmd/{service}/main.go`** — Thin wrapper that builds a `services.Deps`, calls `Start(ctx, deps)`, and handles
  signals. Roughly 40 lines.
- **`pkg/services/{service}/`** — All service logic. Exports a `Start(ctx, deps)` function. Creates its own AMQP
  connections from `deps.Config.RabbitMQURL` to avoid channel contention — AMQP connections are **not** shared in the
  Deps struct.

The **unified binary** (`cmd/spr/main.go`) starts all services as goroutines via `errgroup`, sharing a single `Deps`
instance. This enables testcontainers-based integration testing where the full system runs in-process.

## Data Access

- **core-svc** is the only service that accesses the main Postgres database, through sqlc-generated queries
  (`internal/gen/coredb/`) and higher-level helpers (`pkg/pkgdb/`).
- **package-watcher** uses Valkey for tracking known versions and Gitea configuration.
- **be-runner** uses Valkey for Gitea credentials, plus the GitHub and npm API clients.
- All inter-service communication goes through RabbitMQ (Watermill AMQP).

## Naming Conventions

- Service directories use hyphenated names matching the service identity: `core-svc`, `be-runner`, `package-watcher`
- `cmd/poller/` maps to `pkg/services/package-watcher/` (the cmd name is the short alias)
- `cmd/spr/` is the unified binary, not a service itself

## Tooling

- `just check` — Build all Go + lint (staticcheck) + frontend checks (svelte-check, biome, prettier, markdownlint)
- `just test` — Unit tests only (`go test ./...`)
- `just fix` — Auto-format everything (gofumpt, prettier, biome, markdownlint)
- `just generate` — Regenerate sqlc bindings and OpenAPI spec
- `just integration` — Integration tests (`//go:build integration`)

The reason integration tests are separated is because they require API keys and make calls to external services. See
`.env.sample` for requirements
