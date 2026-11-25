#!/usr/bin/env bash
#
# Convert the old Go/SQLite wishlist database into INSERT statements for D1.
#
# Usage:
#   ./scripts/convert-db.sh [source.db] [output.sql]
#
# Defaults:
#   source.db  wishlist.db (repo root)
#   output.sql seed.sql
#
# The script only prints counts, never row contents.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

SOURCE="${1:-$ROOT_DIR/wishlist.db}"
OUTPUT="${2:-$ROOT_DIR/seed.sql}"
MIGRATION="$ROOT_DIR/migrations/0001_init.sql"

if [ ! -f "$SOURCE" ]; then
  echo "Source database not found: $SOURCE" >&2
  exit 1
fi

TMP="$(mktemp -t wishpage-convert.XXXXXX.db)"
trap 'rm -f "$TMP"' EXIT

sqlite3 "$TMP" < "$MIGRATION"

sqlite3 "$TMP" <<SQL
ATTACH DATABASE '$SOURCE' AS old;

INSERT INTO members (id, name)
  SELECT id, name FROM old.family_members;

INSERT INTO items (id, member_id, name, link, price_cents, reserved)
  SELECT id,
         family_member_id,
         name,
         NULLIF(link, ''),
         CAST(ROUND(price * 100) AS INTEGER),
         CASE WHEN reserved THEN 1 ELSE 0 END
  FROM old.items;

SELECT 'members=' || COUNT(*) FROM members;
SELECT 'items=' || COUNT(*) FROM items;
SELECT 'price_mismatches=' || COUNT(*)
  FROM items i JOIN old.items o ON o.id = i.id
  WHERE COALESCE(i.price_cents, -1)
        <> COALESCE(CAST(ROUND(o.price * 100) AS INTEGER), -1);
SELECT 'link_mismatches=' || COUNT(*)
  FROM items i JOIN old.items o ON o.id = i.id
  WHERE (i.link IS NULL) <> (o.link IS NULL OR o.link = '');
SQL

sqlite3 "$TMP" <<'SQL' > "$OUTPUT"
.mode insert members
SELECT * FROM members;
.mode insert items
SELECT * FROM items;
SQL

echo "Wrote $OUTPUT"
