import type { ItemInput } from "./db";

const MAX_NAME_LENGTH = 200;
const LINK_ERROR = "Link must be a valid HTTP or HTTPS URL";

export type Field = "name" | "link" | "price";

type Result<T> =
  | { ok: true; value: T }
  | { ok: false; error: string; field?: Field };

/** Form bodies can contain File values; anything that is not a string is empty. */
export function fieldString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

export function validateMemberName(value: unknown): Result<string> {
  const name = fieldString(value).trim() || "New family member";
  if (name.length > MAX_NAME_LENGTH) {
    return {
      ok: false,
      error: `Name too long (max ${MAX_NAME_LENGTH} characters)`,
      field: "name",
    };
  }
  return { ok: true, value: name };
}

export function validateItem(fields: Record<string, unknown>): Result<ItemInput> {
  const name = fieldString(fields.name).trim();
  if (name === "") {
    return { ok: false, error: "Item name is required", field: "name" };
  }
  if (name.length > MAX_NAME_LENGTH) {
    return {
      ok: false,
      error: `Item name too long (max ${MAX_NAME_LENGTH} characters)`,
      field: "name",
    };
  }

  const rawPrice = fieldString(fields.price).trim();
  let priceCents: number | null = null;
  if (rawPrice !== "") {
    const price = Number(rawPrice);
    if (!Number.isFinite(price)) {
      return { ok: false, error: "Invalid price", field: "price" };
    }
    if (price < 0) {
      return { ok: false, error: "Price cannot be negative", field: "price" };
    }
    priceCents = Math.round(price * 100);
  }

  const rawLink = fieldString(fields.link).trim();
  let link: string | null = null;
  if (rawLink !== "") {
    let parsed: URL;
    try {
      parsed = new URL(rawLink);
    } catch {
      return { ok: false, error: LINK_ERROR, field: "link" };
    }
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return { ok: false, error: LINK_ERROR, field: "link" };
    }
    link = rawLink;
  }

  return { ok: true, value: { name, link, price_cents: priceCents } };
}

/** Formats integer cents as "12.34", or null when no price is set. */
export function formatPrice(cents: number | null): string | null {
  return cents === null ? null : (cents / 100).toFixed(2);
}
