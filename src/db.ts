export type Member = {
  id: number;
  name: string;
};

export type Item = {
  id: number;
  member_id: number;
  name: string;
  link: string | null;
  price_cents: number | null;
  reserved: number;
  member_name: string;
};

export type ItemInput = {
  name: string;
  link: string | null;
  price_cents: number | null;
};

const ITEM_COLUMNS = `
  i.id, i.member_id, i.name, i.link, i.price_cents, i.reserved,
  m.name AS member_name
`;

export async function loadAll(
  db: D1Database,
): Promise<{ members: Member[]; items: Item[] }> {
  const [membersResult, itemsResult] = await db.batch([
    db.prepare("SELECT id, name FROM members ORDER BY name COLLATE NOCASE"),
    db.prepare(`
      SELECT ${ITEM_COLUMNS}
      FROM items i
      JOIN members m ON m.id = i.member_id
      ORDER BY CASE WHEN i.price_cents IS NULL THEN 0 ELSE 1 END,
               i.price_cents ASC,
               i.name COLLATE NOCASE ASC
    `),
  ]);

  return {
    members: (membersResult.results ?? []) as unknown as Member[],
    items: (itemsResult.results ?? []) as unknown as Item[],
  };
}

export async function getMember(
  db: D1Database,
  id: number,
): Promise<Member | null> {
  return await db
    .prepare("SELECT id, name FROM members WHERE id = ?")
    .bind(id)
    .first<Member>();
}

export async function getItem(
  db: D1Database,
  id: number,
): Promise<Item | null> {
  return await db
    .prepare(`
      SELECT ${ITEM_COLUMNS}
      FROM items i
      JOIN members m ON m.id = i.member_id
      WHERE i.id = ?
    `)
    .bind(id)
    .first<Item>();
}

export async function createMember(
  db: D1Database,
  name: string,
): Promise<number> {
  const result = await db
    .prepare("INSERT INTO members (name) VALUES (?)")
    .bind(name)
    .run();
  return Number(result.meta.last_row_id);
}

export async function deleteMember(db: D1Database, id: number): Promise<void> {
  await db.prepare("DELETE FROM members WHERE id = ?").bind(id).run();
}

export async function createItem(
  db: D1Database,
  memberId: number,
  input: ItemInput,
): Promise<number> {
  const result = await db
    .prepare(
      "INSERT INTO items (member_id, name, link, price_cents) VALUES (?, ?, ?, ?)",
    )
    .bind(memberId, input.name, input.link, input.price_cents)
    .run();
  return Number(result.meta.last_row_id);
}

export async function updateItem(
  db: D1Database,
  id: number,
  input: ItemInput,
): Promise<void> {
  await db
    .prepare("UPDATE items SET name = ?, link = ?, price_cents = ? WHERE id = ?")
    .bind(input.name, input.link, input.price_cents, id)
    .run();
}

export async function deleteItem(db: D1Database, id: number): Promise<void> {
  await db.prepare("DELETE FROM items WHERE id = ?").bind(id).run();
}

export async function setReserved(
  db: D1Database,
  id: number,
  reserved: 0 | 1,
): Promise<void> {
  await db
    .prepare("UPDATE items SET reserved = ? WHERE id = ?")
    .bind(reserved, id)
    .run();
}

export async function memberNameTaken(
  db: D1Database,
  name: string,
): Promise<boolean> {
  const hit = await db
    .prepare("SELECT 1 AS hit FROM members WHERE name = ?")
    .bind(name)
    .first();
  return hit !== null;
}

export async function itemNameTaken(
  db: D1Database,
  memberId: number,
  name: string,
  excludeId = 0,
): Promise<boolean> {
  const hit = await db
    .prepare(
      "SELECT 1 AS hit FROM items WHERE member_id = ? AND name = ? AND id != ?",
    )
    .bind(memberId, name, excludeId)
    .first();
  return hit !== null;
}
