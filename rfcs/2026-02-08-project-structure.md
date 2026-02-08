+++
title = "Project layout and directory structure"
authors = ["cheongyx@cardiff.ac.uk"]
reviewers = []
creation = "2026-02-08"
last_updated = "2026-02-08"
+++

## Summary

Directory structure and code organization conventions for the Secure Package Registry.

## Directory Structure

```plain
├── cmd/                   # Service binaries
│   ├── be-analysis/       # Behavioral anomaly detection
│   ├── be-runner/         # Behavioral analysis runner
│   ├── core-svc/          # Central data service
│   ├── differ/            # Differential analysis
│   ├── poller/            # Git & registry polling
│   ├── reg-proxy/         # Registry proxy
│   └── rep-build/         # Reproducible build runner
├── pkg/                   # Shared libraries
├── dashboard-ui/          # SvelteKit frontend
├── infra/                 # DevOps & infrastructure
└── rfcs/                  # RFC documents
```

## Service Naming

Abbreviated names with hyphens separating concepts:

- `be-analysis`, `be-runner` - Behavioral domain services
- `core-svc` - Central data service
- `differ`, `poller` - Standalone concept services
- `reg-proxy`, `rep-build` - Registry-related services

## Code Placement

### cmd/{service}/

- Each service has a `main.go` entrypoint
- Service-specific code only
- Services communicate via RabbitMQ

### pkg/

- Reusable library code
- No service-specific business logic

### dashboard-ui/

- SvelteKit full-stack application
- See dashboard-ui/README.md for details

### infra/

- Infrastructure code (Cloudflare workers, config)
- Not part of core product

### rfcs/

- RFC documents per `2026-01-28-rfc-process.md`
- Naming: `YYYY-MM-DD-{kebab-case-title}.md`

## Data Access

- **core-svc** is the only service that accesses the main database
- Other services may have lightweight dependencies (e.g., key-value stores)
- Example: `poller` may use Valkey for tracking latest package versions

## Tooling

See `2026-01-30-formatting-linting-choices.md` for formatting and linting conventions.

## Open Questions

1. **pkg/ structure:** How should shared libraries be organized?

## Changelog

N/A
