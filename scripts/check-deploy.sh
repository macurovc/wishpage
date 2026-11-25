#!/usr/bin/env bash
#
# Fail early unless a local wrangler.jsonc with a real D1 id exists.
# wrangler.jsonc is git-ignored; wrangler.jsonc.example is the committed
# template, so the account's database id never enters the public repo.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG="$ROOT_DIR/wrangler.jsonc"
PLACEHOLDER="00000000-0000-0000-0000-000000000000"

if [ ! -f "$CONFIG" ]; then
  cat >&2 <<'MESSAGE'
wrangler.jsonc is missing.

  cp wrangler.jsonc.example wrangler.jsonc
  1. npx wrangler login
  2. npx wrangler d1 create wishpage
  3. paste the printed id into wrangler.jsonc
  4. npm run db:remote
  5. npx wrangler secret put EDIT_PASSWORD
  6. npx wrangler secret put SESSION_SECRET
  7. npm run deploy
MESSAGE
  exit 1
fi

if ! grep -q "$PLACEHOLDER" "$CONFIG"; then
  echo "Deploy config looks set."
  exit 0
fi

cat >&2 <<'MESSAGE'
wrangler.jsonc still has the placeholder D1 database_id.

  1. npx wrangler d1 create wishpage   (or use the existing database)
  2. paste the printed id into wrangler.jsonc
  3. npm run db:remote
  4. npx wrangler secret put EDIT_PASSWORD
  5. npx wrangler secret put SESSION_SECRET
  6. npm run deploy
MESSAGE

exit 1
