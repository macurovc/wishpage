import { env, SELF } from "cloudflare:test";
import { beforeEach, describe, expect, it } from "vitest";

const BASE = "https://example.com";
const PASSWORD = "test-password";

// ------------------------------------------------------------------ helpers

async function resetDatabase(): Promise<void> {
  await env.DB.batch([
    env.DB.prepare("DELETE FROM items"),
    env.DB.prepare("DELETE FROM members"),
  ]);
}

async function seedMember(name = "Alice"): Promise<number> {
  const result = await env.DB.prepare(
    "INSERT INTO members (name) VALUES (?)",
  )
    .bind(name)
    .run();
  return Number(result.meta.last_row_id);
}

async function seedItem(
  memberId: number,
  name = "Book",
  priceCents: number | null = 1200,
): Promise<number> {
  const result = await env.DB.prepare(
    "INSERT INTO items (member_id, name, link, price_cents) VALUES (?, ?, ?, ?)",
  )
    .bind(memberId, name, "https://example.com/item", priceCents)
    .run();
  return Number(result.meta.last_row_id);
}

function formPost(
  path: string,
  fields: Record<string, string>,
  cookie?: string,
): Promise<Response> {
  return SELF.fetch(`${BASE}${path}`, {
    method: "POST",
    body: new URLSearchParams(fields),
    headers: {
      "content-type": "application/x-www-form-urlencoded",
      ...(cookie ? { cookie } : {}),
    },
    redirect: "manual",
  });
}

async function login(): Promise<string> {
  const response = await formPost("/login", { password: PASSWORD });
  expect(response.status).toBe(303);
  expect(response.headers.get("location")).toBe("/edit");

  const setCookie = response.headers.get("set-cookie");
  expect(setCookie).not.toBeNull();
  return setCookie!.split(";")[0];
}

async function pageText(path: string, cookie?: string): Promise<string> {
  const response = await SELF.fetch(`${BASE}${path}`, {
    headers: cookie ? { cookie } : {},
  });
  expect(response.status).toBe(200);
  return response.text();
}

function compact(html: string): string {
  return html.replace(/\s+/g, " ");
}

async function countRows(table: "members" | "items"): Promise<number> {
  const row = await env.DB.prepare(
    `SELECT COUNT(*) AS count FROM ${table}`,
  ).first<{ count: number }>();
  return Number(row?.count ?? 0);
}

beforeEach(resetDatabase);

// --------------------------------------------------------------- public view

describe("public view", () => {
  it("renders members, items and prices", async () => {
    const memberId = await seedMember("Alice");
    await seedItem(memberId, "Board game", 2499);

    const html = await pageText("/");

    expect(html).toContain("Alice");
    expect(html).toContain("Board game");
    expect(html).toContain('class="currency">€</span>');
    expect(html).toContain("24.99");
  });

  it("shows the public empty state when there are no members", async () => {
    const html = await pageText("/");
    expect(html).toContain("No wishlist yet");
    expect(html).toContain("Check back soon");
  });

  it("shows the edit empty state on the edit page", async () => {
    const cookie = await login();
    const html = await pageText("/edit", cookie);
    expect(html).toContain("No family members yet");
    expect(html).toContain("Add the first one below");
  });

  it("shows the only member's list by default", async () => {
    const memberId = await seedMember("Alice");
    await seedItem(memberId, "Book");

    const html = await pageText("/");
    expect(html).toContain("Alice");
    expect(html).toContain("Book");

    const cookie = await login();
    const edit = await pageText("/edit", cookie);
    expect(edit).toContain("Alice");
    expect(edit).toContain("Book");
  });

  it("exposes stable identifiers for people and items", async () => {
    const first = await seedMember("Alice");
    await seedMember("Bob");
    const itemId = await seedItem(first);

    const html = await pageText("/");
    expect(html).toContain(`data-person="${first}"`);
    expect(html).toContain(`id="item-${itemId}"`);
  });

  it("orders items without a price first, then by price", async () => {
    const memberId = await seedMember();
    await seedItem(memberId, "PricedTen", 1000);
    await seedItem(memberId, "NoPrice", null);
    await seedItem(memberId, "PricedFive", 500);

    const html = await pageText("/");
    const noPrice = html.indexOf("NoPrice");
    const five = html.indexOf("PricedFive");
    const ten = html.indexOf("PricedTen");

    expect(noPrice).toBeGreaterThan(-1);
    expect(noPrice).toBeLessThan(five);
    expect(five).toBeLessThan(ten);
  });

  it("keeps a reserved item in place instead of moving it to the end", async () => {
    const memberId = await seedMember();
    await seedItem(memberId, "Cheap", 500);
    const middleId = await seedItem(memberId, "Middle", 1000);
    await seedItem(memberId, "Pricey", 2000);

    await env.DB.prepare("UPDATE items SET reserved = 1 WHERE id = ?")
      .bind(middleId)
      .run();

    const html = await pageText("/");
    const cheap = html.indexOf("Cheap");
    const middle = html.indexOf("Middle");
    const pricey = html.indexOf("Pricey");

    expect(cheap).toBeLessThan(middle);
    expect(middle).toBeLessThan(pricey);
  });

  it("escapes item names and never emits unsafe link schemes", async () => {
    const memberId = await seedMember();
    await env.DB.prepare(
      "INSERT INTO items (member_id, name, link) VALUES (?, ?, ?)",
    )
      .bind(memberId, "<script>alert(1)</script>", "javascript:alert(1)")
      .run();

    const html = await pageText("/");

    expect(html).toContain("&lt;script&gt;alert(1)&lt;/script&gt;");
    expect(html).not.toContain("<script>alert(1)</script>");
    expect(html).not.toContain('href="javascript:');
  });

  it("quotes attribute values that contain spaces", async () => {
    const memberId = await seedMember();
    await seedItem(memberId, "Blue sweater", 1000);
    const cookie = await login();

    const html = await pageText("/edit", cookie);

    expect(html).toContain('value="Blue sweater"');
    expect(html).toContain('href="/favicon.svg"');
  });

  it("opens a modal confirmation for available items", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId, "Book");

    const html = compact(await pageText("/"));

    expect(html).toContain(`data-dialog="get-${itemId}"`);
    expect(html).toContain(`id="get-${itemId}"`);
    expect(html).toContain('method="dialog"');
    expect(html).toContain("Get “Book” for Alice?");
  });

  it("renders the reserved state in the same footer slot", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId);
    await env.DB.prepare("UPDATE items SET reserved = 1 WHERE id = ?")
      .bind(itemId)
      .run();

    const html = compact(await pageText("/"));

    expect(html).toContain('class="reserved-status"');
    expect(html).not.toContain(`data-dialog="get-${itemId}"`);
  });

  it("emits reserved as a class token, not an attribute", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId);
    await env.DB.prepare("UPDATE items SET reserved = 1 WHERE id = ?")
      .bind(itemId)
      .run();

    const html = await pageText("/");

    expect(html).toContain('class="gift-row reserved"');
  });

  it("renders every list and shows the requested person", async () => {
    const alice = await seedMember("Alice");
    const bob = await seedMember("Bob");
    await seedItem(alice, "Alice gift");
    await seedItem(bob, "Bob gift");

    const alicePage = compact(await pageText(`/?open=${alice}`));
    expect(alicePage).toContain("Alice gift");
    expect(alicePage).toContain("Bob gift");
    expect(alicePage).toMatch(
      new RegExp(`data-person="${alice}"[^>]*aria-current="page"`),
    );
    expect(alicePage).not.toMatch(
      new RegExp(`data-person-list="${alice}"[^>]*hidden`),
    );
    expect(alicePage).toMatch(
      new RegExp(`data-person-list="${bob}"[^>]*hidden`),
    );

    const fallback = compact(await pageText("/?open=abc"));
    expect(fallback).toMatch(
      new RegExp(`data-person="${alice}"[^>]*aria-current="page"`),
    );
    expect(fallback).not.toMatch(
      new RegExp(`data-person-list="${alice}"[^>]*hidden`),
    );
    expect(fallback).toMatch(
      new RegExp(`data-person-list="${bob}"[^>]*hidden`),
    );
  });
});

// --------------------------------------------------------------- reservations

describe("reservations", () => {
  it("reserves an item anonymously and preserves the open section", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId);

    const response = await formPost(`/items/${itemId}/reserve`, {});

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe(
      `/?open=${memberId}#item-${itemId}`,
    );

    const row = await env.DB.prepare(
      "SELECT reserved FROM items WHERE id = ?",
    )
      .bind(itemId)
      .first<{ reserved: number }>();
    expect(row?.reserved).toBe(1);

    expect(await pageText("/")).toContain("Taken");
  });

  it("does not expose a public unreserve route", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId);

    const response = await formPost(`/items/${itemId}/unreserve`, {});
    expect(response.status).toBe(404);
  });

  it("returns 400 for a non-numeric item id", async () => {
    const response = await formPost("/items/abc/reserve", {});
    expect(response.status).toBe(400);
  });
});

// ----------------------------------------------------------------------- auth

describe("auth", () => {
  it("redirects /edit to /login without a cookie", async () => {
    const response = await SELF.fetch(`${BASE}/edit`, { redirect: "manual" });
    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("/login");
  });

  it("rejects a wrong password without setting a cookie", async () => {
    const response = await formPost("/login", { password: "wrong" });

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("/login?error=password");
    expect(response.headers.get("set-cookie")).toBeNull();
  });

  it("sets a hardened session cookie on success", async () => {
    const response = await formPost("/login", { password: PASSWORD });
    const cookie = response.headers.get("set-cookie") ?? "";

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("/edit");
    expect(cookie).toMatch(/HttpOnly/i);
    expect(cookie).toMatch(/Secure/i);
    expect(cookie).toMatch(/SameSite=Strict/i);
    expect(cookie).toMatch(/Max-Age=43200/i);
  });

  it("redirects an authenticated user away from /login", async () => {
    const cookie = await login();
    const response = await SELF.fetch(`${BASE}/login`, {
      headers: { cookie },
      redirect: "manual",
    });

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("/edit");
  });

  it("rejects a tampered cookie", async () => {
    const cookie = await login();
    const tampered = `${cookie.split(".")[0]}.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`;

    const response = await formPost(
      "/admin/members",
      { name: "Mallory" },
      tampered,
    );
    expect(response.status).toBe(401);
    expect(await countRows("members")).toBe(0);
  });

  it("rejects an expired cookie", async () => {
    const response = await formPost(
      "/admin/members",
      { name: "Mallory" },
      "1.deadbeef",
    );
    expect(response.status).toBe(401);
  });

  it("blocks admin routes without a cookie", async () => {
    const response = await formPost("/admin/members", { name: "Mallory" });
    expect(response.status).toBe(401);
  });

  it("logs out and clears the cookie", async () => {
    const cookie = await login();

    const response = await formPost("/logout", {}, cookie);
    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("/");
    expect(response.headers.get("set-cookie")).toMatch(/Max-Age=0/i);

    // Sessions are stateless: logout drops the cookie, so the next request
    // without it is unauthenticated. The old token stays valid until expiry.
    const edit = await SELF.fetch(`${BASE}/edit`, { redirect: "manual" });
    expect(edit.status).toBe(303);
    expect(edit.headers.get("location")).toBe("/login");
  });
});

// ---------------------------------------------------------------------- admin

describe("admin", () => {
  it("creates a family member", async () => {
    const cookie = await login();

    const response = await formPost(
      "/admin/members",
      { name: "Charlie" },
      cookie,
    );
    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toMatch(/^\/edit\?open=\d+$/);

    expect(await pageText("/edit", cookie)).toContain("Charlie");
  });

  it("rejects a duplicate family member name", async () => {
    await seedMember("Alice");
    const cookie = await login();

    const response = await formPost(
      "/admin/members",
      { name: "Alice" },
      cookie,
    );
    expect(response.status).toBe(400);
    expect(await response.text()).toContain("already exists");
  });

  it("defaults an empty member name to 'New family member'", async () => {
    const cookie = await login();
    await formPost("/admin/members", { name: "   " }, cookie);

    expect(await pageText("/edit", cookie)).toContain("New family member");
  });

  it("creates an item and stores the price in cents", async () => {
    const memberId = await seedMember();
    const cookie = await login();

    const response = await formPost(
      `/admin/members/${memberId}/items`,
      { name: "Sweater", link: "https://shop.example/sweater", price: "12.34" },
      cookie,
    );

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toMatch(
      new RegExp(`^/edit\\?open=${memberId}#item-\\d+$`),
    );

    const row = await env.DB.prepare(
      "SELECT link, price_cents FROM items WHERE name = ?",
    )
      .bind("Sweater")
      .first<{ link: string; price_cents: number }>();

    expect(row?.link).toBe("https://shop.example/sweater");
    expect(row?.price_cents).toBe(1234);
  });

  it("rejects unsafe links and negative prices", async () => {
    const memberId = await seedMember();
    const cookie = await login();

    const badLink = await formPost(
      `/admin/members/${memberId}/items`,
      { name: "Bad link", link: "javascript:alert(1)", price: "" },
      cookie,
    );
    expect(badLink.status).toBe(400);
    expect(await badLink.text()).toContain("valid HTTP or HTTPS");

    const badPrice = await formPost(
      `/admin/members/${memberId}/items`,
      { name: "Bad price", link: "", price: "-1" },
      cookie,
    );
    expect(badPrice.status).toBe(400);
    expect(await badPrice.text()).toContain("cannot be negative");

    expect(await countRows("items")).toBe(0);
  });

  it("keeps submitted values and focuses the bad field on error", async () => {
    const memberId = await seedMember();
    const cookie = await login();

    const response = await formPost(
      `/admin/members/${memberId}/items`,
      { name: "Nice name", link: "javascript:alert(1)", price: "10" },
      cookie,
    );

    expect(response.status).toBe(400);
    const html = compact(await response.text());

    expect(html).toContain('value="Nice name"');
    expect(html).toContain('value="javascript:alert(1)"');
    expect(html).toContain('value="10"');
    expect(html).toMatch(/name="link"[^>]*aria-invalid="true"/);
    expect(html).toMatch(/name="link"[^>]*autofocus/);
    expect(html).toContain('role="alert"');
    expect(html).toContain("Edit Alice's list");
  });

  it("keeps the member name and focuses the field on a duplicate", async () => {
    await seedMember("Alice");
    const cookie = await login();

    const response = await formPost(
      "/admin/members",
      { name: "Alice" },
      cookie,
    );

    expect(response.status).toBe(400);
    const html = compact(await response.text());

    expect(html).toContain('value="Alice"');
    expect(html).toMatch(/name="name"[^>]*autofocus/);
    expect(html).toMatch(/name="name"[^>]*aria-invalid="true"/);
  });

  it("keeps edited values and focuses the bad field on an edit error", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId, "Old", 999);
    const cookie = await login();

    const response = await formPost(
      `/admin/items/${itemId}`,
      { name: "Changed", link: "", price: "-5" },
      cookie,
    );

    expect(response.status).toBe(400);
    const html = compact(await response.text());

    expect(html).toContain('value="Changed"');
    expect(html).toContain('value="-5"');
    expect(html).toMatch(/name="price"[^>]*autofocus/);

    const row = await env.DB.prepare("SELECT name FROM items WHERE id = ?")
      .bind(itemId)
      .first<{ name: string }>();
    expect(row?.name).toBe("Old");
  });

  it("rejects a duplicate item name per member but allows it for others", async () => {
    const alice = await seedMember("Alice");
    const bob = await seedMember("Bob");
    await seedItem(alice, "Book");
    const cookie = await login();

    const duplicate = await formPost(
      `/admin/members/${alice}/items`,
      { name: "Book", link: "", price: "" },
      cookie,
    );
    expect(duplicate.status).toBe(400);
    expect(await duplicate.text()).toContain("already exists");

    const otherMember = await formPost(
      `/admin/members/${bob}/items`,
      { name: "Book", link: "", price: "" },
      cookie,
    );
    expect(otherMember.status).toBe(303);
    expect(await countRows("items")).toBe(2);
  });

  it("updates an item and clears empty link and price", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId, "Old", 999);
    const cookie = await login();

    const response = await formPost(
      `/admin/items/${itemId}`,
      { name: "New", link: "", price: "" },
      cookie,
    );

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe(
      `/edit?open=${memberId}#item-${itemId}`,
    );

    const row = await env.DB.prepare(
      "SELECT name, link, price_cents FROM items WHERE id = ?",
    )
      .bind(itemId)
      .first<{ name: string; link: string | null; price_cents: number | null }>();

    expect(row?.name).toBe("New");
    expect(row?.link).toBeNull();
    expect(row?.price_cents).toBeNull();
  });

  it("reserves and un-reserves as an admin", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId);
    const cookie = await login();

    await formPost(`/admin/items/${itemId}/reserve`, {}, cookie);
    let row = await env.DB.prepare("SELECT reserved FROM items WHERE id = ?")
      .bind(itemId)
      .first<{ reserved: number }>();
    expect(row?.reserved).toBe(1);

    await formPost(`/admin/items/${itemId}/unreserve`, {}, cookie);
    row = await env.DB.prepare("SELECT reserved FROM items WHERE id = ?")
      .bind(itemId)
      .first<{ reserved: number }>();
    expect(row?.reserved).toBe(0);
  });

  it("deletes an item", async () => {
    const memberId = await seedMember();
    const itemId = await seedItem(memberId);
    const cookie = await login();

    const response = await formPost(
      `/admin/items/${itemId}/delete`,
      {},
      cookie,
    );

    expect(response.status).toBe(303);
    expect(await countRows("items")).toBe(0);
  });

  it("deletes a member and cascades to their items", async () => {
    const memberId = await seedMember();
    await seedItem(memberId);
    const cookie = await login();

    const response = await formPost(
      `/admin/members/${memberId}/delete`,
      {},
      cookie,
    );

    expect(response.status).toBe(303);
    expect(await countRows("members")).toBe(0);
    expect(await countRows("items")).toBe(0);
  });

  it("returns 400 for a non-numeric admin id", async () => {
    const cookie = await login();
    const response = await formPost("/admin/items/abc/delete", {}, cookie);
    expect(response.status).toBe(400);
  });

  it("returns 404 for a missing item", async () => {
    const cookie = await login();
    const response = await formPost("/admin/items/999/delete", {}, cookie);
    expect(response.status).toBe(404);
  });
});

// Static assets are served by the Workers assets layer, outside the Worker
// script, so they are verified with `wrangler dev` rather than here.
