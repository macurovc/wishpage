import { html, raw } from "hono/html";
import type { Item, Member } from "./db";
import { formatPrice } from "./validate";
import type { Field } from "./validate";

type Html = ReturnType<typeof html>;

export type ItemDraft = { name: string; link: string; price: string };

export type EditError =
  | { kind: "add-member"; message: string; name: string }
  | {
      kind: "add-item";
      message: string;
      memberId: number;
      draft: ItemDraft;
      field?: Field;
    }
  | {
      kind: "edit-item";
      message: string;
      memberId: number;
      itemId: number;
      draft: ItemDraft;
      field?: Field;
    };

// --------------------------------------------------------------------- icons

type IconName =
  | "gift"
  | "user"
  | "external"
  | "pencil"
  | "check"
  | "lock"
  | "package"
  | "alert";

/* A single 24px stroke grid keeps the whole interface visually consistent. */
const ICON_PATHS: Record<IconName, string> = {
  gift: `<path d="M20 12v8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-8"/><rect x="2" y="7" width="20" height="5" rx="1"/><path d="M12 22V7"/><path d="M12 7H7.5a2.5 2.5 0 0 1 0-5C11 2 12 7 12 7Z"/><path d="M12 7h4.5a2.5 2.5 0 0 0 0-5C13 2 12 7 12 7Z"/>`,
  user: `<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>`,
  external: `<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><path d="M15 3h6v6"/><path d="M10 14 21 3"/>`,
  pencil: `<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>`,
  check: `<path d="M20 6 9 17l-5-5"/>`,
  lock: `<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>`,
  package: `<path d="M21 8v13H3V8"/><path d="M1 3h22v5H1z"/><path d="M10 12h4"/>`,
  alert: `<circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>`,
};

function icon(name: IconName, size = 16): Html {
  return html`<svg
    class="icon"
    width="${size}"
    height="${size}"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >${raw(ICON_PATHS[name])}</svg>`;
}

// ------------------------------------------------------------------- layout

function layout(title: string, body: unknown): Html {
  return html`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="color-scheme" content="light dark" />
    <meta
      name="theme-color"
      content="#f4f0e6"
      media="(prefers-color-scheme: light)"
    />
    <meta
      name="theme-color"
      content="#171310"
      media="(prefers-color-scheme: dark)"
    />
    <title>${title}</title>
    <meta name="robots" content="noindex, nofollow" />
    <meta
      name="description"
      content="A simple family wishlist. Choose a name to see their list and claim a gift."
    />
    <meta property="og:type" content="website" />
    <meta property="og:title" content="${title}" />
    <meta
      property="og:description"
      content="A simple family wishlist. Choose a name to see their list and claim a gift."
    />
    <meta name="twitter:card" content="summary" />
    <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
    <link
      rel="preload"
      href="/fraunces.woff2"
      as="font"
      type="font/woff2"
      crossorigin
    />
    <link rel="stylesheet" href="/app.css" />
    <script src="/app.js" defer></script>
  </head>
  <body>
    <main class="container">${body}</main>
  </body>
</html>`;
}

function brand(size = 26): Html {
  return html`<span class="brand-icon">${icon("gift", size)}</span
    ><span>Family Wishlist</span>`;
}

function formError(message: string, id: string): Html {
  return html`<p class="form-error" id="${id}" role="alert">
    ${icon("alert", 14)}<span>${message}</span>
  </p>`;
}

/**
 * Emit aria-invalid only for the field that actually failed. The stylesheet
 * marks [aria-invalid="true"]; emitting "false" on every other field would be
 * noise for assistive tech.
 */
function errorAttrs(
  active: boolean,
  errorId: string,
): ReturnType<typeof raw> | "" {
  return active ? raw(`aria-invalid="true" aria-describedby="${errorId}"`) : "";
}

// ------------------------------------------------------------------ helpers

function itemsFor(items: Item[], memberId: number): Item[] {
  return items.filter((item) => item.member_id === memberId);
}

function selectedMember(
  members: Member[],
  openId: number | null,
): Member | null {
  if (members.length === 0) return null;
  return members.find((member) => member.id === openId) ?? members[0];
}

/**
 * Links are validated on write, but never emit a URL that could execute script
 * even if a bad value reaches the database by another route.
 */
function safeLink(link: string | null): string | null {
  if (link === null) return null;
  try {
    const url = new URL(link);
    return url.protocol === "http:" || url.protocol === "https:" ? link : null;
  } catch {
    return null;
  }
}

function peopleRail(
  members: Member[],
  selectedId: number,
  base: string,
): Html {
  return html`<nav class="people" aria-label="Family members">
    ${members.map((member) => {
      const active = member.id === selectedId;
      return html`<a
        class="${active ? "person active" : "person"}"
        data-person="${member.id}"
        href="${base}?open=${member.id}"
        aria-current="${active ? "page" : "false"}"
        >${member.name}</a
      >`;
    })}
  </nav>`;
}

// ---------------------------------------------------------------- empty states

function publicEmptyState(): Html {
  return html`
    <section class="empty">
      <div class="empty-art">${icon("gift", 34)}</div>
      <h2>No wishlist yet</h2>
      <p>Check back soon.</p>
    </section>
  `;
}

function editEmptyState(): Html {
  return html`
    <section class="empty">
      <div class="empty-art">${icon("user", 34)}</div>
      <h2>No family members yet</h2>
      <p>Add the first one below.</p>
    </section>
  `;
}

// ---------------------------------------------------------------- public view

export function publicPage(
  members: Member[],
  items: Item[],
  openId: number | null,
): Html {
  const selected = selectedMember(members, openId);

  return layout(
    "Family Wishlist",
    html`
      <header class="app-header">
        <h1 class="brand">${brand()}</h1>
        <a href="/login" class="edit-link">${icon("pencil", 13)} Edit</a>
      </header>
      ${selected === null
        ? publicEmptyState()
        : html`
            ${members.length > 1
              ? peopleRail(members, selected.id, "/")
              : html`<h2 class="list-title">${selected.name}</h2>`}
            ${members.map((member) =>
              publicList(member, items, member.id === selected.id),
            )}
          `}
    `,
  );
}

function publicList(
  member: Member,
  allItems: Item[],
  active: boolean,
): Html {
  const own = itemsFor(allItems, member.id);

  return html`<section
    data-person-list="${member.id}"
    aria-label="${member.name}'s list"
    ${active ? "" : "hidden"}
  >
    ${own.length === 0
      ? html`<p class="empty-list">${icon("package", 18)} Nothing here yet.</p>`
      : html`
          <div class="list-head" aria-hidden="true">
            <span></span><span>Gift</span><span>Price</span
            ><span>Claim</span>
          </div>
          <ol class="gift-list">
            ${own.map((item, index) => publicGiftRow(item, index + 1))}
          </ol>
        `}
  </section>`;
}

function publicGiftRow(item: Item, index: number): Html {
  const price = formatPrice(item.price_cents);
  const link = safeLink(item.link);
  const title = item.reserved ? html`<del>${item.name}</del>` : item.name;

  return html`
    <li
      class="gift-row${item.reserved ? " reserved" : ""}"
      id="item-${item.id}"
    >
      <span class="gift-index">${String(index).padStart(2, "0")}</span>
      <div class="gift-main">
        <h3>
          ${link === null
            ? title
            : html`<a
                class="gift-title-link"
                href="${link}"
                target="_blank"
                rel="noopener noreferrer"
                >${title}&nbsp;${icon("external", 12)}</a
              >`}
        </h3>
      </div>
      <span class="gift-price${price === null ? " unpriced" : ""}"
        >${price === null
          ? html`<span class="no-price">—</span>`
          : html`<span class="currency">€</span>${price}`}</span
      >
      <div class="gift-action">
        ${item.reserved
          ? html`<span class="reserved-status">Taken</span>`
          : html`
              <button
                type="button"
                class="button button-primary claim"
                commandfor="get-${item.id}"
                command="show-modal"
                data-dialog="get-${item.id}"
              >
                I'll get this
              </button>
              <dialog
                id="get-${item.id}"
                class="reserve-dialog"
                aria-labelledby="get-title-${item.id}"
                aria-describedby="get-note-${item.id}"
              >
                <article>
                  <header>
                    <h3 id="get-title-${item.id}">
                      Get “${item.name}” for ${item.member_name}?
                    </h3>
                  </header>
                  <p id="get-note-${item.id}">
                    It'll show as taken for everyone else.
                  </p>
                  <footer>
                    <form method="dialog">
                      <button class="button dialog-cancel">Cancel</button>
                    </form>
                    <form
                      method="post"
                      action="/items/${item.id}/reserve"
                      data-claim
                      data-item="${item.id}"
                    >
                      <button type="submit" class="button button-primary">
                        Yes, I'll get it
                      </button>
                    </form>
                  </footer>
                </article>
              </dialog>
            `}
      </div>
    </li>
  `;
}

// ------------------------------------------------------------------ edit view

export function editPage(
  members: Member[],
  items: Item[],
  openId: number | null,
  error?: EditError,
): Html {
  const selected = selectedMember(members, openId);
  const addMemberError = error?.kind === "add-member" ? error : null;

  return layout(
    "Edit Wishlist",
    html`
      <header class="app-header">
        <h1 class="brand">${brand()}</h1>
        <form method="post" action="/logout">
          <button type="submit" class="done-link">
            ${icon("check", 13)} Done
          </button>
        </form>
      </header>

      ${selected === null
        ? editEmptyState()
        : html`
            ${members.length > 1
              ? peopleRail(members, selected.id, "/edit")
              : ""}
            ${members.map((member) =>
              managePanel(
                member,
                items,
                error,
                member.id === selected.id,
                members.length === 1,
              ),
            )}
          `}
      ${addPersonForm(addMemberError)}
    `,
  );
}

function managePanel(
  member: Member,
  allItems: Item[],
  error: EditError | undefined,
  active: boolean,
  showTitle: boolean,
): Html {
  const own = itemsFor(allItems, member.id);
  const addError =
    error?.kind === "add-item" && error.memberId === member.id ? error : null;
  const draft = addError?.draft;
  const field = addError?.field;
  const errorId = `add-item-error-${member.id}`;

  return html`
    <section
      class="manage"
      data-person-list="${member.id}"
      aria-label="Edit ${member.name}'s list"
      ${active ? "" : "hidden"}
    >
      ${showTitle ? html`<h2 class="manage-title">${member.name}</h2>` : ""}

      <div class="gift-row edit-head-row" aria-hidden="true">
        <span></span>
        <div class="edit-content">
          <div class="edit-form">
            <span>Name</span><span>Link</span><span>Price (€)</span><span></span>
          </div>
        </div>
      </div>

      ${addError ? formError(addError.message, errorId) : ""}

      <div class="gift-row add-row">
        <span></span>
        <div class="edit-content">
          <form
            method="post"
            action="/admin/members/${member.id}/items"
            class="add-gift-form"
            autocomplete="off"
          >
            <input
              type="text"
              name="name"
              placeholder="Gift name"
              aria-label="Gift name"
              maxlength="200"
              required
              value="${draft?.name ?? ""}"
              ${errorAttrs(field === "name", errorId)}
              ${field === "name" ? "autofocus" : ""}
            />
            <input
              type="url"
              name="link"
              placeholder="Link (optional)"
              aria-label="Link"
              value="${draft?.link ?? ""}"
              ${errorAttrs(field === "link", errorId)}
              ${field === "link" ? "autofocus" : ""}
            />
            <input
              type="number"
              name="price"
              placeholder="Price (€)"
              aria-label="Price in euros"
              step="0.01"
              min="0"
              value="${draft?.price ?? ""}"
              ${errorAttrs(field === "price", errorId)}
              ${field === "price" ? "autofocus" : ""}
            />
            <button type="submit" class="button button-primary">
              Add gift
            </button>
          </form>
        </div>
      </div>

      ${own.length === 0
        ? html`<p class="empty-list">
            ${icon("package", 18)} No gifts yet. Add one above.
          </p>`
        : html`<ol class="gift-list">
            ${own.map((item, index) => editGiftRow(item, index + 1, error))}
          </ol>`}

      <div class="manage-foot">
        <form
          method="post"
          action="/admin/members/${member.id}/delete"
          onsubmit="return confirm('Remove this person and all their gifts?')"
        >
          <button type="submit" class="danger-link">
            Remove ${member.name}
          </button>
        </form>
      </div>
    </section>
  `;
}

function addPersonForm(
  error: Extract<EditError, { kind: "add-member" }> | null,
): Html {
  const errorId = "add-member-error";

  return html`
    <section class="add-person" id="add-person">
      <h2>Add someone</h2>
      ${error ? formError(error.message, errorId) : ""}
      <form
        method="post"
        action="/admin/members"
        class="add-person-form"
        autocomplete="off"
      >
        <input
          type="text"
          name="name"
          placeholder="Name"
          maxlength="200"
          required
          value="${error?.name ?? ""}"
          ${errorAttrs(error !== null, errorId)}
          ${error ? "autofocus" : ""}
        />
        <button type="submit" class="button button-primary">Add</button>
      </form>
    </section>
  `;
}

function editGiftRow(
  item: Item,
  index: number,
  error?: EditError,
): Html {
  const itemError =
    error?.kind === "edit-item" && error.itemId === item.id ? error : null;
  const draft = itemError?.draft;
  const field = itemError?.field;
  const errorId = `edit-item-error-${item.id}`;

  const price = formatPrice(item.price_cents) ?? "";

  return html`
    <li
      class="gift-row edit-row${item.reserved ? " reserved" : ""}"
      id="item-${item.id}"
    >
      <span class="gift-index">${String(index).padStart(2, "0")}</span>
      <div class="edit-content">
        ${itemError ? formError(itemError.message, errorId) : ""}
        <form
          method="post"
          action="/admin/items/${item.id}"
          class="edit-form"
          autocomplete="off"
        >
          <input
            type="text"
            name="name"
            aria-label="Name"
            value="${draft?.name ?? item.name}"
            maxlength="200"
            required
            ${errorAttrs(field === "name", errorId)}
            ${field === "name" ? "autofocus" : ""}
          />
          <input
            type="url"
            name="link"
            aria-label="Link"
            value="${draft?.link ?? item.link ?? ""}"
            placeholder="https://..."
            ${errorAttrs(field === "link", errorId)}
            ${field === "link" ? "autofocus" : ""}
          />
          <input
            type="number"
            name="price"
            aria-label="Price in euros"
            value="${draft?.price ?? price}"
            step="0.01"
            min="0"
            placeholder="0.00"
            ${errorAttrs(field === "price", errorId)}
            ${field === "price" ? "autofocus" : ""}
          />
          <button type="submit" class="button button-quiet">
            ${icon("check", 13)} Save
          </button>
        </form>

        <div class="edit-actions">
          ${item.reserved
            ? html`<span class="reserved-status">Taken</span>`
            : ""}
          ${item.reserved
            ? html`
                <form
                  method="post"
                  action="/admin/items/${item.id}/unreserve"
                >
                  <button type="submit" class="state-button">
                    Mark as available
                  </button>
                </form>
              `
            : html`
                <form method="post" action="/admin/items/${item.id}/reserve">
                  <button type="submit" class="state-button">
                    Mark as taken
                  </button>
                </form>
              `}
          <form
            method="post"
            action="/admin/items/${item.id}/delete"
            onsubmit="return confirm('Delete this gift?')"
          >
            <button type="submit" class="danger-link">Delete</button>
          </form>
        </div>
      </div>
    </li>
  `;
}

// ----------------------------------------------------------------- login/error

export function loginPage(error?: string): Html {
  const message =
    error === "password"
      ? "Incorrect password. Please try again."
      : error === "config"
        ? "Password not configured. Please contact the administrator."
        : error
          ? "An error occurred. Please try again."
          : "";

  return layout(
    "Login - Family Wishlist",
    html`
      <section class="login">
        <h1 class="brand">${brand(22)}</h1>
        <p class="subtitle">Enter the password to edit the list.</p>

        ${message
          ? html`<article class="flash error">
              ${icon("alert", 15)}<span>${message}</span>
            </article>`
          : ""}

        <form method="post" action="/login" autocomplete="on">
          <label>
            Password
            <input
              type="password"
              name="password"
              placeholder="Enter password"
              autocomplete="current-password"
              autofocus
              required
            />
          </label>
          <button type="submit" class="button button-primary button-block">
            ${icon("lock", 15)} Start editing
          </button>
        </form>

        <p class="back"><a href="/">← Back to wishlist</a></p>
      </section>
    `,
  );
}

export function errorPage(message: string): Html {
  return layout(
    "Error - Family Wishlist",
    html`
      <section class="empty">
        <div class="empty-art error">${icon("alert", 32)}</div>
        <h1>Something went wrong</h1>
        <p>${message}</p>
        <p><a href="/" role="button">← Back to wishlist</a></p>
      </section>
    `,
  );
}
