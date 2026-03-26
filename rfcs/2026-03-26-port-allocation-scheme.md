+++
title = "Port allocation"
author = ["cheongyx@cardiff.ac.uk"]
date = "2026-03-26"
+++

## Summary

This RFC defines a structured port allocation scheme for the Secure Package Registry deployment to improve clarity, maintainability, and avoid port conflicts with host services.

## Motivation

Currently, port assignments in `compose.yml` are ad-hoc and inconsistent:

- Public services use ports like 7001, 7002, 7003, 7004
- Internal services may clash with host services (e.g., port 8081 was occupied by hydroxide)
- No clear distinction between public, internal 1st party, and internal 3rd party services

## Proposal

### Port Ranges

| Range       | Category           | Description                                               |
| ----------- | ------------------ | --------------------------------------------------------- |
| 7000-7999   | Public Services    | External-facing APIs, UIs, registries accessible to users |
| 20000-29999 | Internal 1st Party | Our own services' internal APIs (not exposed to host)     |
| 10000-19999 | Internal 3rd Party | External dependencies (databases, message queues, caches) |

### Port Assignments

#### Public Services (7000-7999)

| Port | Service        | Description                                             |
| ---- | -------------- | ------------------------------------------------------- |
| 7001 | Caddy          | Main public gateway (external API, registry, dashboard) |
| 7002 | Registry Proxy | NPM registry directly accessible (bypass Caddy)         |
| 7003 | Home UI        | Public landing page                                     |

#### Internal 1st Party (20000-29999)

| Port  | Service          | Description                          |
| ----- | ---------------- | ------------------------------------ |
| 20001 | SPR Internal API | Build registration, admin operations |

#### Internal 3rd Party (10000-19999)

| Port  | Service             | Description                 |
| ----- | ------------------- | --------------------------- |
| 10001 | RabbitMQ Management | Web UI for queue management |
| 10002 | RabbitMQ AMQP       | Message queue protocol      |
| 10003 | MinIO S3            | Object storage API          |
| 10004 | MinIO Console       | Web management interface    |
| 10005 | Gitea Web UI        | Git repository management   |

### Internal Container Ports

Services communicate internally via container network (not exposed to host):

- SPR External API: 8080 (internal only, accessed via Caddy at 7001)
- SPR Internal API: 8081 (mapped to host 20001 for CLI tools)
- Registry Proxy: 7002 (mapped to host 7002)

## Implementation

### compose.yml Changes

```yaml
services:
  rabbitmq:
    ports:
      - '127.0.0.1:10001:15672' # Management (was 7001)
      - '127.0.0.1:10002:5672' # AMQP (was 5672)

  minio:
    ports:
      - '127.0.0.1:10004:9001' # Console (was 7003)
      - '127.0.0.1:10003:9000' # S3 API (was 9000)

  gitea:
    ports:
      - '127.0.0.1:10005:3000' # Web UI (was 7004)

  home-ui:
    ports:
      - '127.0.0.1:7003:3000' # Public (was 3001)

  caddy:
    ports:
      - '127.0.0.1:7001:8000' # Public gateway (was 8000)

  spr:
    ports:
      - '127.0.0.1:7002:7002' # Registry proxy
      - '127.0.0.1:20001:8081' # Internal API (was 18081)
```

### Application Changes

- `cmd/rep-build/main.go`: Update default API URL from `http://localhost:18081` to `http://localhost:20001`

## Backwards Compatibility

This is a **breaking change** for:

- Local development workflows using hardcoded ports
- Documentation referencing old ports
- Scripts accessing RabbitMQ management (7001 → 10001)

## Migration Guide

When this RFC is implemented, update:

1. Browser bookmarks: `localhost:7001` → `localhost:10001` (RabbitMQ)
2. MinIO console: `localhost:7003` → `localhost:10004`
3. Gitea access: `localhost:7004` → `localhost:10005`
4. Home page: `localhost:3001` → `localhost:7003`
5. Main gateway: `localhost:8000` → `localhost:7001`
