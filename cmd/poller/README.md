# Package Watcher

Watches requested NPM packages and publishes updates when a newer version appears in the NPM changes feed.

## Overview

`cmd/poller` is a thin entrypoint around `pkg/services/package-watcher`. The service:

1. Subscribes to `spr.package.requested`
2. Starts an NPM changes-feed poller from the current registry sequence
3. Keeps an in-memory watch list of requested packages and their last published version
4. Fetches the latest NPM version for watched packages
5. Publishes `spr.package.updated` when a new version is detected

The current implementation is NPM-only and does not persist watched packages or the last processed changes-feed sequence
across restarts.

## Architecture

```mermaid
flowchart TB
    subgraph External["External Services"]
        NPM_Replicate["replicate.npmjs.com<br/>/_changes"]
        NPM_Registry["registry.npmjs.org"]
    end

    subgraph Internal["Poller Service"]
        Watcher["Package Watcher"]
        WatchList[(In-Memory Watch List)]
    end

    subgraph Infrastructure["Infrastructure"]
        RabbitMQ[RabbitMQ]
        CoreSvc[core-svc]
    end

    CoreSvc -->|"Publish<br/>spr.package.requested"| RabbitMQ
    RabbitMQ -->|"Consume"| Watcher

    Watcher -->|"Track package"| WatchList
    Watcher -->|"Poll changes"| NPM_Replicate
    NPM_Replicate -->|"Package names + sequence"| Watcher
    Watcher -->|"Fetch latest version"| NPM_Registry
    NPM_Registry -->|"Package metadata"| Watcher

    Watcher -->|"Publish<br/>spr.package.updated"| RabbitMQ

    style External fill:#e1f5fe
    style Internal fill:#e8f5e9
    style Infrastructure fill:#fff3e0
```

## Configuration

The watcher uses shared core config from `pkg/config`.

| Environment Variable | Default                                                          | Description                                                  |
| -------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------ |
| `RABBITMQ_URL`       | `amqp://admin:admin@rabbitmq:5672/`                              | Watermill AMQP connection URL for subscribing and publishing |
| `DATABASE_URL`       | `postgres://postgres:postgres@core_db:5432/core?sslmode=disable` | Shared core config value; currently unused by the watcher    |
| `VALKEY_URL`         | `valkey:6379`                                                    | Shared core config value; currently unused by the watcher    |
| `NPM_REGISTRY_URL`   | `https://registry.npmjs.org`                                     | Base URL for package metadata lookups                        |
| `NPM_REPLICATE_URL`  | `https://replicate.npmjs.com`                                    | CouchDB-compatible changes feed endpoint                     |
| `NPM_HTTP_TIMEOUT`   | `10s`                                                            | Timeout for NPM HTTP requests                                |

## Runtime Behavior

### Startup

- Creates RabbitMQ subscriber and publisher clients
- Builds an NPM client and poller
- Resolves the initial changes-feed sequence from `GET /` on the replicate endpoint
- Starts two concurrent loops:
  - a RabbitMQ consumer for package requests
  - a replicate-feed polling loop

Because the poller starts from the current sequence when no sequence is provided, updates that happened before the
service started are not replayed.

### Package request flow

When `core-svc` publishes `spr.package.requested`, the payload is a gob-encoded `messages.PackageRequest`:

```go
type PackageRequest struct {
    Ecosystem  string
    Identifier string
}
```

The watcher:

1. Decodes the message
2. Adds the package to the in-memory watch list with an empty known version
3. Immediately fetches the latest version from the NPM registry
4. Publishes `spr.package.updated` if a version is found
5. Acknowledges the message only after the initial fetch succeeds

Non-NPM requests are rejected as unsupported.

### Update detection flow

The NPM poller fetches batches of up to 100 changes from `_changes` every 30 seconds.

For each change:

- design documents and deleted packages are skipped
- only packages already present in the watch list are considered
- the watcher fetches the latest version from `registry.npmjs.org/<package>`
- if the version differs from the last published version, it publishes an update and refreshes the cached version

If a poll returns a full batch, the next poll is triggered immediately so the watcher can catch up without waiting for
the 30 second interval.

## Changes Feed

```plain
GET https://replicate.npmjs.com/_changes?since=<sequence>&limit=100
```

The code starts from the current `update_seq` returned by the replicate root endpoint unless a sequence is explicitly
supplied to `pkg/npm.Poller.Start`.

Response shape used by the poller:

```json
{
  "results": [
    { "seq": "97816046", "id": "lodash", "changes": [{ "rev": "123-abc" }] },
    { "seq": "97816047", "id": "react", "changes": [{ "rev": "456-def" }] }
  ],
  "last_seq": "97816047"
}
```

## RabbitMQ Topics and Payloads

### Consumed: `spr.package.requested`

```go
type PackageRequest struct {
    Ecosystem  string
    Identifier string
}
```

### Published: `spr.package.updated`

```go
type PackageUpdated struct {
    Ecosystem  string
    Identifier string
    Version    string
}
```

Both messages are gob-encoded before being sent through Watermill AMQP.

## Current Limitations

- Watched packages are stored in memory only
- The NPM changes-feed sequence is not persisted yet
- Only the `npm` ecosystem is supported
- Version checks happen by fetching full package metadata on demand
