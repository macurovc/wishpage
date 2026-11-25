export const SESSION_COOKIE = "wishpage_session";
export const SESSION_MAX_AGE = 43200; // 12 hours, in seconds

const encoder = new TextEncoder();

function base64url(bytes: ArrayBuffer): string {
  const view = new Uint8Array(bytes);
  let binary = "";
  for (const byte of view) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

async function hmac(secret: string, value: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  return base64url(await crypto.subtle.sign("HMAC", key, encoder.encode(value)));
}

/** Session cookie value is "<unix expiry>.<hmac>"; no server-side state. */
export async function createSession(secret: string): Promise<string> {
  const expires = Math.floor(Date.now() / 1000) + SESSION_MAX_AGE;
  return `${expires}.${await hmac(secret, String(expires))}`;
}

export async function isSessionValid(
  cookie: string | undefined,
  secret: string | undefined,
): Promise<boolean> {
  if (!cookie || !secret) return false;

  const [expiresRaw, signature] = cookie.split(".");
  if (!expiresRaw || !signature) return false;

  const expires = Number(expiresRaw);
  if (!Number.isFinite(expires) || expires < Math.floor(Date.now() / 1000)) {
    return false;
  }

  return timingSafeEqual(signature, await hmac(secret, expiresRaw));
}

/**
 * Compares SHA-256 digests so the comparison has a fixed length and does not
 * leak information about where two strings differ.
 */
export async function passwordMatches(
  input: string,
  expected: string | undefined,
): Promise<boolean> {
  if (!expected) return false;

  const [inputDigest, expectedDigest] = await Promise.all([
    crypto.subtle.digest("SHA-256", encoder.encode(input)),
    crypto.subtle.digest("SHA-256", encoder.encode(expected)),
  ]);

  const a = new Uint8Array(inputDigest);
  const b = new Uint8Array(expectedDigest);
  let difference = 0;
  for (let i = 0; i < a.length; i++) {
    difference |= a[i] ^ b[i];
  }
  return difference === 0;
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let difference = 0;
  for (let i = 0; i < a.length; i++) {
    difference |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return difference === 0;
}
