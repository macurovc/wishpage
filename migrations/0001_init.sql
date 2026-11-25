CREATE TABLE members (
  id   INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE items (
  id          INTEGER PRIMARY KEY,
  member_id   INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  link        TEXT,
  price_cents INTEGER CHECK (price_cents IS NULL OR price_cents >= 0),
  reserved    INTEGER NOT NULL DEFAULT 0 CHECK (reserved IN (0, 1))
);

CREATE UNIQUE INDEX items_member_name ON items (member_id, name);
CREATE INDEX items_member ON items (member_id);
