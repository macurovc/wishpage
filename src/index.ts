import { Hono } from "hono";
import { deleteCookie, getCookie, setCookie } from "hono/cookie";
import type { Context } from "hono";
import type { ContentfulStatusCode } from "hono/utils/http-status";

import {
  SESSION_COOKIE,
  SESSION_MAX_AGE,
  createSession,
  isSessionValid,
  passwordMatches,
} from "./auth";
import {
  createItem,
  createMember,
  deleteItem,
  deleteMember,
  getItem,
  getMember,
  itemNameTaken,
  loadAll,
  memberNameTaken,
  setReserved,
  updateItem,
} from "./db";
import { editPage, errorPage, loginPage, publicPage } from "./views";
import type { EditError, ItemDraft } from "./views";
import { fieldString, validateItem, validateMemberName } from "./validate";

type AppEnv = { Bindings: Env };

const app = new Hono<AppEnv>();

// ---------------------------------------------------------------- helpers

function parseId(raw: string | undefined): number | null {
  if (!raw) return null;
  const id = Number(raw);
  return Number.isInteger(id) && id > 0 ? id : null;
}

async function isAuthenticated(c: Context<AppEnv>): Promise<boolean> {
  return isSessionValid(getCookie(c, SESSION_COOKIE), c.env.SESSION_SECRET);
}

function itemDraftFrom(body: Record<string, string | File>): ItemDraft {
  return {
    name: fieldString(body.name),
    link: fieldString(body.link),
    price: fieldString(body.price),
  };
}

async function renderEdit(
  c: Context<AppEnv>,
  options: {
    openId?: number | null;
    status?: ContentfulStatusCode;
    error?: EditError;
  } = {},
) {
  const { members, items } = await loadAll(c.env.DB);
  return c.html(
    editPage(members, items, options.openId ?? null, options.error),
    options.status ?? 200,
  );
}

const DUPLICATE_ITEM =
  "An item with this name already exists for this family member";

// ------------------------------------------------------------ auth middleware

app.use("/admin/*", async (c, next) => {
  if (!(await isAuthenticated(c))) {
    return c.html(errorPage("Unauthorized: Please log in"), 401);
  }
  await next();
});

// --------------------------------------------------------------------- pages

app.get("/", async (c) => {
  const { members, items } = await loadAll(c.env.DB);
  return c.html(
    publicPage(members, items, parseId(c.req.query("open"))),
  );
});

app.get("/edit", async (c) => {
  if (!(await isAuthenticated(c))) {
    return c.redirect("/login", 303);
  }
  return renderEdit(c, { openId: parseId(c.req.query("open")) });
});

app.get("/login", async (c) => {
  if (await isAuthenticated(c)) {
    return c.redirect("/edit", 303);
  }
  return c.html(loginPage(c.req.query("error")));
});

app.post("/login", async (c) => {
  const body = await c.req.parseBody();
  const password = fieldString(body.password);

  if (!c.env.EDIT_PASSWORD || !c.env.SESSION_SECRET) {
    return c.redirect("/login?error=config", 303);
  }
  if (!(await passwordMatches(password, c.env.EDIT_PASSWORD))) {
    return c.redirect("/login?error=password", 303);
  }

  setCookie(c, SESSION_COOKIE, await createSession(c.env.SESSION_SECRET), {
    httpOnly: true,
    secure: true,
    sameSite: "Strict",
    path: "/",
    maxAge: SESSION_MAX_AGE,
  });
  return c.redirect("/edit", 303);
});

app.post("/logout", (c) => {
  deleteCookie(c, SESSION_COOKIE, { path: "/", secure: true });
  return c.redirect("/", 303);
});

// -------------------------------------------------------------- public writes

app.post("/items/:id/reserve", async (c) => {
  const id = parseId(c.req.param("id"));
  if (id === null) {
    return c.html(errorPage("Invalid item ID"), 400);
  }

  const item = await getItem(c.env.DB, id);
  if (item === null) {
    return c.html(errorPage("Item not found"), 404);
  }

  await setReserved(c.env.DB, id, 1);
  return c.redirect(`/?open=${item.member_id}#item-${id}`, 303);
});

// ---------------------------------------------------------------- members

app.post("/admin/members", async (c) => {
  const body = await c.req.parseBody();
  const rawName = fieldString(body.name);
  const parsed = validateMemberName(body.name);

  if (!parsed.ok) {
    return renderEdit(c, {
      status: 400,
      error: { kind: "add-member", message: parsed.error, name: rawName },
    });
  }

  if (await memberNameTaken(c.env.DB, parsed.value)) {
    return renderEdit(c, {
      status: 400,
      error: {
        kind: "add-member",
        message: "A family member with this name already exists",
        name: rawName,
      },
    });
  }

  const id = await createMember(c.env.DB, parsed.value);
  return c.redirect(`/edit?open=${id}`, 303);
});

app.post("/admin/members/:id/delete", async (c) => {
  const id = parseId(c.req.param("id"));
  if (id === null) {
    return c.html(errorPage("Invalid family member ID"), 400);
  }

  await deleteMember(c.env.DB, id);
  return c.redirect("/edit", 303);
});

// ------------------------------------------------------------------- items

app.post("/admin/members/:id/items", async (c) => {
  const memberId = parseId(c.req.param("id"));
  if (memberId === null) {
    return c.html(errorPage("Invalid family member ID"), 400);
  }

  const member = await getMember(c.env.DB, memberId);
  if (member === null) {
    return c.html(errorPage("Family member not found"), 404);
  }

  const body = await c.req.parseBody();
  const draft = itemDraftFrom(body);
  const parsed = validateItem(body);

  if (!parsed.ok) {
    return renderEdit(c, {
      openId: memberId,
      status: 400,
      error: {
        kind: "add-item",
        message: parsed.error,
        memberId,
        draft,
        field: parsed.field,
      },
    });
  }

  if (await itemNameTaken(c.env.DB, memberId, parsed.value.name)) {
    return renderEdit(c, {
      openId: memberId,
      status: 400,
      error: {
        kind: "add-item",
        message: DUPLICATE_ITEM,
        memberId,
        draft,
        field: "name",
      },
    });
  }

  const newId = await createItem(c.env.DB, memberId, parsed.value);
  return c.redirect(`/edit?open=${memberId}#item-${newId}`, 303);
});

app.post("/admin/items/:id", async (c) => {
  const id = parseId(c.req.param("id"));
  if (id === null) {
    return c.html(errorPage("Invalid item ID"), 400);
  }

  const item = await getItem(c.env.DB, id);
  if (item === null) {
    return c.html(errorPage("Item not found"), 404);
  }

  const body = await c.req.parseBody();
  const draft = itemDraftFrom(body);
  const parsed = validateItem(body);

  if (!parsed.ok) {
    return renderEdit(c, {
      openId: item.member_id,
      status: 400,
      error: {
        kind: "edit-item",
        message: parsed.error,
        memberId: item.member_id,
        itemId: id,
        draft,
        field: parsed.field,
      },
    });
  }

  if (
    await itemNameTaken(c.env.DB, item.member_id, parsed.value.name, item.id)
  ) {
    return renderEdit(c, {
      openId: item.member_id,
      status: 400,
      error: {
        kind: "edit-item",
        message: DUPLICATE_ITEM,
        memberId: item.member_id,
        itemId: id,
        draft,
        field: "name",
      },
    });
  }

  await updateItem(c.env.DB, id, parsed.value);
  return c.redirect(`/edit?open=${item.member_id}#item-${id}`, 303);
});

app.post("/admin/items/:id/reserve", async (c) => {
  return setReservation(c, 1);
});

app.post("/admin/items/:id/unreserve", async (c) => {
  return setReservation(c, 0);
});

async function setReservation(c: Context<AppEnv>, reserved: 0 | 1) {
  const id = parseId(c.req.param("id"));
  if (id === null) {
    return c.html(errorPage("Invalid item ID"), 400);
  }

  const item = await getItem(c.env.DB, id);
  if (item === null) {
    return c.html(errorPage("Item not found"), 404);
  }

  await setReserved(c.env.DB, id, reserved);
  return c.redirect(`/edit?open=${item.member_id}#item-${id}`, 303);
}

app.post("/admin/items/:id/delete", async (c) => {
  const id = parseId(c.req.param("id"));
  if (id === null) {
    return c.html(errorPage("Invalid item ID"), 400);
  }

  const item = await getItem(c.env.DB, id);
  if (item === null) {
    return c.html(errorPage("Item not found"), 404);
  }

  await deleteItem(c.env.DB, id);
  return c.redirect(`/edit?open=${item.member_id}`, 303);
});

// ------------------------------------------------------------------ fallbacks

app.notFound((c) => c.html(errorPage("Page not found"), 404));

app.onError((error, c) => {
  console.error("Unhandled error", error);
  return c.html(errorPage("Internal server error"), 500);
});

export default app;
