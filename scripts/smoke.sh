#!/usr/bin/env bash
#
# Minimal post-deploy check. Detects a broken deploy, nothing more.
#
# Usage:
#   ./scripts/smoke.sh https://wishlist.example.com

set -euo pipefail

BASE_URL="${1:-http://localhost:8787}"
BASE_URL="${BASE_URL%/}"

check() {
  local expected="$1"
  local actual="$2"
  local label="$3"

  if [ "$actual" != "$expected" ]; then
    echo "FAIL: $label returned $actual (expected $expected)" >&2
    exit 1
  fi
}

status=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/")
check 200 "$status" "GET /"

status=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/admin/members")
check 401 "$status" "unauthenticated POST /admin/members"

status=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/robots.txt")
check 200 "$status" "GET /robots.txt"

echo "Smoke check passed for $BASE_URL"
