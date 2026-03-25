+++
title = "Registry Proxy Authentication Flow"
authors = ["cheongyx@cardiff.ac.uk"]
tags = ["registry", "authentication", "proxy"]
creation = "2026-03-24"
depends_on = ["2026-01-30-architectural-design.md", "2026-03-08-project-structure.md"]
+++

## Summary

Documents the authentication flow for the registry reverse proxy (`reg-proxy`). Clients authenticate with BetterAuth
API keys sent as Bearer tokens. The proxy validates tokens by hashing them (SHA-256, base64url) and looking up the hash
in the `apikey` table. Valid requests are forwarded to the appropriate Gitea package registry account with the account's
own Gitea token injected.

This replaces the intern's earlier prototype which used Basic auth with hand-rolled tokens stored in a custom
`user_tokens` table.

## Problem

The registry proxy sits between npm clients and Gitea's package registry. It needs to:

1. **Authenticate clients** — only users with valid API keys should access packages.
2. **Own routing** — clients should not control which Gitea account their requests route to (the old implementation
   exposed `/api/packages/{user}/npm/...`, letting clients pick the Gitea user).
3. **Use BetterAuth** — the dashboard already uses BetterAuth for session auth. API keys should come from the same
   system (`@better-auth/api-key` plugin) so users manage keys through the dashboard UI.

## Recommendation

### Authentication scheme

Clients authenticate using Bearer tokens in the `Authorization` header:

```text
Authorization: Bearer <api-key>
```

The proxy validates the token as follows:

1. Extract the raw key from the `Bearer` prefix.
2. Compute `SHA-256(raw_key)` and encode the hash as **base64url with no padding** — this matches BetterAuth's
   `defaultKeyHasher` implementation.
3. Look up the resulting hash in the `apikey` table's `key` column.
4. If found, the request is authenticated. The `referenceId` column links to the owning user.
5. If not found, return `403 Forbidden`.

### Logging levels

| Scenario                          | Level   |
| --------------------------------- | ------- |
| Request accepted (valid token)    | `Debug` |
| Missing or non-Bearer auth header | `Debug` |
| Invalid API key (hash not found)  | `Warn`  |

### URL routing

The proxy exposes a flat namespace to clients:

```text
Client request:  GET /npm/<package-name>
Proxy rewrites:  GET /api/packages/<gitea-account>/npm/<package-name>
```

**Only GET requests are supported.** The proxy blocks PUT, POST, DELETE, and all other HTTP methods with a 405
"Method Not Allowed" response. npm publish/publish/unpublish requires different permissions and routing logic that is
not yet implemented.

Currently all reads route to the **sandbox** Gitea account (`spr-sandbox`). The proxy reads the sandbox account
credentials (username + Gitea token) from the `gitea:config` key in Valkey at startup.

Future work will route based on package metadata (e.g. reproducibility status) — verified packages could be served from
the `spr-registry` account instead of the sandbox.

### Response URL rewriting

JSON responses from Gitea contain internal URLs like `http://gitea:3000/api/packages/spr-sandbox/npm/...`. The proxy
rewrites these to the external URL `http://localhost:7002/npm/...` so npm clients can resolve tarballs correctly.

### Database schema

The `apikey` table is created by migration `000004_add_apikey.up.sql` and matches the `@better-auth/api-key` v1.5.x
schema. The key columns relevant to authentication:

| Column        | Purpose                                                       |
| ------------- | ------------------------------------------------------------- |
| `key`         | SHA-256 hash of the raw API key (base64url, no padding)       |
| `referenceId` | Foreign key to the user who owns the key                      |
| `enabled`     | Whether the key is active (checked by `GetAPIKeyOwner` query) |

### Service architecture

The proxy follows the established `pkg/services` pattern:

- **`pkg/services/reg-proxy/service.go`** — all business logic (auth, routing, URL rewriting)
- **`cmd/reg-proxy/main.go`** — ~40-line boilerplate entry point
- **`cmd/spr/main.go`** — starts `regproxy.Start` in the shared errgroup alongside `core-svc` and `be-runner`

The shared `httpserver.Server` struct (extracted to `pkg/httpserver/server.go`) handles listener lifecycle for all
services.

### Seeding (dev mode)

When `SPR_MOCK=true` is set, `cmd/spr/main.go` calls `seed.Run()` at startup, which:

1. Inserts a dev user (`seed-user-1`) into the `user` table.
2. Generates a random 32-byte API key (hex-encoded, 64 chars).
3. Hashes it with SHA-256 → base64url (no padding) and inserts into the `apikey` table.
4. Logs the raw key to stdout so the developer can use it immediately.

The seed is idempotent — the user insert uses `ON CONFLICT DO NOTHING` and the API key gets a random ID each time.

## Testing

### Prerequisites

Start the full stack:

```sh
podman compose build
podman compose down && podman compose up -d
```

Wait for all services to be healthy:

```sh
podman compose ps
```

The `spr` service must be running and the `gitea` healthcheck must pass (it waits for `gitea:config` to exist in
Valkey).

### Getting an API key

**Option A: Use SPR_MOCK=true (recommended for local dev)**

Add `SPR_MOCK=true` to the `spr` service's environment in `compose.yml`, then restart. The seed logs the raw API key:

```sh
podman compose logs spr 2>&1 | grep "Seeded API key"
```

Look for a line like:

```text
INF Seeded API key (use as Bearer token) apikey_id=... key_start=abcd1234... raw_key=abcd1234...full64chars...
```

**Option B: Create a key via the BetterAuth dashboard**

Use the dashboard UI at `http://localhost:8000` to create an API key through the user settings. The dashboard uses the
`@better-auth/api-key` plugin which writes to the same `apikey` table.

### Configure npm

Create or edit `~/.npmrc` (or a project-local `.npmrc`):

```ini
//localhost:7002/npm/:_authToken=<your-raw-api-key>
registry=http://localhost:7002/npm/
```

Note the differences from the old setup:

| Old (Basic auth)                                | New (Bearer auth)            |
| ----------------------------------------------- | ---------------------------- |
| `_auth=<base64-encoded>`                        | `_authToken=<raw-key>`       |
| `http://localhost:7002/api/packages/admin/npm/` | `http://localhost:7002/npm/` |

### Verify authentication

Test that the proxy accepts a valid key and rejects invalid ones:

```sh
# Should return a valid JSON response (or 404 if the package doesn't exist yet)
curl -H "Authorization: Bearer <your-raw-api-key>" http://localhost:7002/npm/is-number

# Should return 403 Forbidden
curl -H "Authorization: Bearer invalid-key" http://localhost:7002/npm/is-number

# Should return 403 Forbidden (no auth)
curl http://localhost:7002/npm/is-number
```

### Publish a test package (optional)

To test the full flow including package resolution, you can publish a package to the sandbox:

```sh
mkdir /tmp/test-pkg && cd /tmp/test-pkg
npm init -y --scope=@test
npm publish --registry http://localhost:7002/npm/
```

Then verify it resolves:

```sh
npm view @test/test-pkg --registry http://localhost:7002/npm/
```

Note: The seed only creates a user and API key — it does **not** seed any packages. You must publish a package first
before `npm install` or `npm view` will return results.

## Open Questions

1. **Token expiration / `enabled` / `expiresAt` check**: The proxy validates tokens via `GetAPIKeyOwner`, which checks
   `enabled = TRUE` and that `expiresAt` is either NULL or in the future. Revocation is supported by setting `enabled =
FALSE` or `expiresAt` to a past timestamp. A dashboard UI for key management is a separate concern.
2. **Rate limiting**: The `apikey` table has rate-limiting columns (`rateLimitEnabled`, `rateLimitMax`, etc.) from
   BetterAuth. Should the proxy enforce these, or leave rate limiting to a future API gateway?
3. **Intelligent routing**: When should we implement tag-based routing (e.g. verified packages from `spr-registry`,
   unverified from `spr-sandbox`)? This is blocked on the reproducible builds pipeline being functional.
