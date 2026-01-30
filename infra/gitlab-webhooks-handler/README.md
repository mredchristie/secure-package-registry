# GitLab → Discord Webhook Router

Cloudflare Worker that filters and routes GitLab webhooks to two Discord channels based on event importance.

## Environment Variables

| Variable                      | Description                                                        |
| ----------------------------- | ------------------------------------------------------------------ |
| `GITLAB_WEBHOOK_SECRET`       | Secret token to validate requests (set in GitLab webhook settings) |
| `DISCORD_WEBHOOK_IMPORTANT`   | Discord webhook URL for critical events                            |
| `DISCORD_WEBHOOK_UNIMPORTANT` | Discord webhook URL for routine notifications                      |

## Event Routing

| GitLab Event           | Action     | Route   | Discord Channel |
| ---------------------- | ---------- | ------- | --------------- |
| **Merge Request**      | `open`     | ✅ Send | Important 🔴    |
|                        | `merge`    | ✅ Send | Important 🔴    |
|                        | `close`    | ✅ Send | Important 🔴    |
|                        | `reopen`   | ✅ Send | Important 🔴    |
|                        | `update`   | ❌ Drop | —               |
|                        | `approved` | ❌ Drop | —               |
| **Issue**              | `open`     | ✅ Send | Important 🔴    |
|                        | `close`    | ✅ Send | Important 🔴    |
|                        | `reopen`   | ✅ Send | Important 🔴    |
|                        | `update`   | ❌ Drop | —               |
|                        | `label`    | ❌ Drop | —               |
| **Note (Comment)**     | Any        | ✅ Send | Unimportant 🔵  |
| **Push/Wiki/Pipeline** | Any        | ❌ Drop | —               |

## Setup

1. **Create Worker**: Paste `worker.js` into Cloudflare Workers
2. **Set Variables**: Add the three environment variables in Worker Settings → Variables
3. **Configure GitLab**:
   - URL: `https://your-worker.your-subdomain.workers.dev`
   - Secret Token: Match `GITLAB_WEBHOOK_SECRET`
   - Triggers: Enable **Merge requests**, **Comments**, **Issues**
   - Disable: Push events (ignored by design)

## Security

- Returns HTTP 401 if `X-Gitlab-Token` header doesn't match secret
- Secrets never logged or exposed to client
- Rotate secrets via Cloudflare dashboard if compromised
