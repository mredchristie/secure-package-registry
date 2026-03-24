#!/bin/sh
su-exec git /usr/local/bin/gitea migrate

# Create admin account
su-exec git /usr/local/bin/gitea admin user create \
  --username 'spr-admin' \
  --password 'spr-admin' \
  --email admin@local \
  --admin \
  --must-change-password=false

# Create registry account (owns curated/approved packages)
su-exec git /usr/local/bin/gitea admin user create \
  --username 'spr-registry' \
  --password 'spr-registry' \
  --email registry@local \
  --must-change-password=false

# Create sandbox account (temporary packages during behavioral analysis)
su-exec git /usr/local/bin/gitea admin user create \
  --username 'spr-sandbox' \
  --password 'spr-sandbox' \
  --email sandbox@local \
  --must-change-password=false

# Generate access tokens for each account
su-exec git /usr/local/bin/gitea admin user generate-access-token \
  --username 'spr-admin' \
  --token-name "spr-token" \
  --scopes "all" | sed "s/^.*: //" >/tmp/token-admin.txt

su-exec git /usr/local/bin/gitea admin user generate-access-token \
  --username 'spr-registry' \
  --token-name "spr-token" \
  --scopes "all" | sed "s/^.*: //" >/tmp/token-registry.txt

su-exec git /usr/local/bin/gitea admin user generate-access-token \
  --username 'spr-sandbox' \
  --token-name "spr-token" \
  --scopes "all" | sed "s/^.*: //" >/tmp/token-sandbox.txt

# Build and store gitea:config as a single JSON entry in valkey
GITEA_CONFIG=$(jq -nc \
  --arg admin_user "spr-admin" \
  --arg admin_token "$(cat /tmp/token-admin.txt)" \
  --arg registry_user "spr-registry" \
  --arg registry_token "$(cat /tmp/token-registry.txt)" \
  --arg sandbox_user "spr-sandbox" \
  --arg sandbox_token "$(cat /tmp/token-sandbox.txt)" \
  '{
    "base_url": "http://gitea:3000",
    "accounts": {
      "admin":    { "username": $admin_user,    "token": $admin_token },
      "registry": { "username": $registry_user, "token": $registry_token },
      "sandbox":  { "username": $sandbox_user,  "token": $sandbox_token }
    }
  }')
valkey-cli -h valkey -p 6379 SET gitea:config "$GITEA_CONFIG" NX
