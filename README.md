# Wishpage

A minimal family wishlist running as a single Cloudflare Worker with D1
(managed SQLite). Server-rendered HTML, plain forms, one shared edit password,
and one small progressive-enhancement script for instant person switching, the
claim dialog, and in-place claiming.

## Stack

| Piece | Choice |
|---|---|
| Runtime | Cloudflare Workers |
| Storage | D1 (SQLite) |
| Language | TypeScript |
| Framework | Hono (`hono/html` escapes by default) |
| Auth | HMAC-signed cookie, one `EDIT_PASSWORD` |
| CSS | One hand-written stylesheet (own reset + form controls), Fraunces display face |
| JS | ~90 lines: person switching, claim dialog, in-place claim |
| Deploy | `npm run deploy` (config guard, then `wrangler deploy`) |

Runtime dependency: `hono` only. Everything else is development tooling.

## Features

- Public wishlist with a typographic people switcher; every list is rendered
  once, so switching is instant and requires no request
- Items with name, optional link, optional price, and a taken state
- Anyone can claim a gift through a native modal confirmation (no login)
- Password-protected edit mode: add/delete members, add/edit/delete items,
  mark as taken/available
- Same validation as the Go version: required and max-length names, unique
  member names, unique item name per member, HTTP(S)-only links, non-negative
  prices
- Inline SVG icon set instead of platform-dependent emoji, with a matching
  favicon and a consistent 24px stroke grid
- Inline validation errors keep what you typed, mark the bad field, and focus it
- `?open=<member id>` selects the person; edit actions anchor to the affected item
- Claiming swaps the row in place: the order, the page position, and the rest of
  the list never move
- Done sits in the masthead, so leaving edit mode is always one click away
- Cream/ink palette with one signal red in light and dark, Fraunces display type

## Quick start

Requirements: Node.js 20+ and npm. `sqlite3` is only needed to import an old
database.

```bash
cp wrangler.jsonc.example wrangler.jsonc   # placeholder id is fine locally
npm install
npm run db:local     # create the local D1 database and apply migrations
npm run dev          # http://localhost:8787
```

`wrangler.jsonc` is git-ignored so your account's D1 id stays out of the public
repo; `wrangler.jsonc.example` is the template. Local D1 state is keyed by that
id, so if you later replace the placeholder with a real one, run
`npm run db:local` again and re-import your data.

Local secrets live in `.dev.vars` (not committed):

```
EDIT_PASSWORD=dev-password
SESSION_SECRET=dev-secret
```

## Scripts

| Command | Purpose |
|---|---|
| `npm run dev` | Local server with local D1 |
| `npm test` | 39 integration tests in the Workers runtime |
| `npm run typecheck` | `tsc --noEmit` |
| `npm run db:local` | Apply migrations to local D1 |
| `npm run db:remote` | Apply migrations to remote D1 |
| `npm run deploy` | Check config, then deploy the Worker |
| `npm run release` | Test, then deploy (with the same config check) |
| `npm run types` | Regenerate env types after changing `wrangler.jsonc` |

## Deploy

```bash
cp wrangler.jsonc.example wrangler.jsonc
npx wrangler login
npx wrangler d1 create wishpage
# paste the printed database_id into wrangler.jsonc

npm run db:remote
npx wrangler secret put EDIT_PASSWORD
npx wrangler secret put SESSION_SECRET   # openssl rand -base64 32
npm run deploy
```

`npm run deploy` stops early and prints this checklist if `wrangler.jsonc` is
missing or still has the placeholder `database_id`, so a broken D1 binding
cannot be deployed by accident.

Test on the `*.workers.dev` URL first. To use a custom domain, add it to
`wrangler.jsonc` (the zone must be in the same Cloudflare account):

```jsonc
"routes": [{ "pattern": "wishlist.example.com", "custom_domain": true }]
```

Deployment is just the Worker and its D1 database — no container, volume,
tunnel, or reverse proxy.

## Import the old data

`scripts/convert-db.sh` reads the Go app's SQLite file and writes
`INSERT` statements for the new schema. It preserves IDs and converts
`price REAL` to `price_cents INTEGER`. It prints counts only.

```bash
./scripts/convert-db.sh wishlist.db seed.sql

# local check
npx wrangler d1 execute wishpage --local --file=seed.sql

# remote import (clear first if re-importing)
npx wrangler d1 execute wishpage --remote --command \
  "DELETE FROM items; DELETE FROM members;"
npx wrangler d1 execute wishpage --remote --file=seed.sql
```

Then run `npm run dev` and walk through the site before deploying.

## Backups

D1 keeps automatic point-in-time history (7 days on the Free plan, 30 on Paid):

```bash
npx wrangler d1 time-travel info wishpage
```

For an offsite copy, export after bulk changes or on a schedule:

```bash
mkdir -p backups
npx wrangler d1 export wishpage --remote \
  --output "backups/wishpage-$(date +%F).sql"
```

## Smoke check

```bash
./scripts/smoke.sh https://wishlist.example.com
```

Checks `GET /`, that an unauthenticated admin POST is rejected, and that
`robots.txt` is served.

## Tests

`npm test` runs 39 integration tests against the real Workers runtime with a
local D1 database. The suite covers rendering, stable ordering, escaping,
reservations, auth/cookie hardening, validation, CRUD, cascade deletes, and
redirects. GitHub Actions runs the same typecheck and tests on `main` and pull
requests.

Static assets are served by the Workers assets layer rather than the Worker
script, so they are checked locally with `npm run dev` instead of the unit
suite.

## Layout

```
src/
  index.ts     routes, middleware, handlers
  db.ts        D1 queries and row types
  views.ts     server-rendered HTML
  auth.ts      cookie signing, password check
  validate.ts  name/link/price validation
  env.d.ts     secret bindings
migrations/    0001_init.sql
public/        app.css, app.js, favicon.svg, robots.txt,
               fraunces.woff2, fraunces.OFL.txt
test/          integration tests and migration setup
scripts/       check-deploy.sh, convert-db.sh, smoke.sh
.github/       CI workflow (typecheck + tests) and dependabot
wrangler.jsonc.example  Worker, assets, and D1 binding (copy to wrangler.jsonc)
worker-configuration.d.ts  generated binding types
vitest.config.ts  Workers test pool + local D1 migrations
```

## Differences from the Go app

- **No email notifications.** Removed intentionally.
- **A people rail instead of an accordion.** All lists are sent in the initial
  HTML. Switching is client-side, updates `?open=<id>` with `pushState`, and
  supports the back/forward buttons. Without JavaScript the rail links navigate
  normally, so every list remains reachable.
- **Form POSTs, not an API.** Every mutation is an HTML form followed by a
  redirect; the selected person is preserved with `?open=<id>`. Claiming is the
  one exception: it swaps the returned row in place with `fetch`, so the list,
  its order, and the scroll position stay put. Without JavaScript the form posts
  normally and the redirect lands on the item's anchor.
- **One small client script** (~90 lines) switches people without a request,
  opens the claim confirmation with the native `<dialog>` element, and applies
  the claimed row in place. The claim button also uses `commandfor`, so browsers
  with invoker-command support work even if the script does not load.
- **Claim and taken share the action column**: the status label occupies the
  same slot and height as the button, so rows never change height.
- **Inline validation** keeps what you typed, marks the offending field, and
  focuses it on the returned page.
- **Editing is the same list**, not a separate settings screen: one flat row per
  gift with inline fields, one column header, and Done in the masthead.
- **Stateless sessions.** Logout clears the cookie in the browser, but a signed
  token that was already extracted stays valid until its 12-hour expiry.
- **A full visual redesign.** A small hand-written base supplies the reset and
  form controls; the product adds a cream/ink palette with one signal red,
  Fraunces display type, a marigold active tab, and a numbered table-like list
  with no cards or shadows.
- **No JSON API.** Every mutation is an HTML form POST followed by a redirect.
- Items are ordered by: unpriced first, then price, then name. Reserving an
  item does not move it.

## Operational notes

The Free plan is far above this app's needs: 100k Worker requests/day, 10 ms CPU
per request, D1 at 5M rows read and 100k rows written per day, 5 GB storage.
Requests for static assets are free and do not invoke the Worker. If a limit is
hit, requests fail until the daily reset rather than billing.

`public/robots.txt` disallows crawling, which keeps the public page out of
search results and avoids accidental traffic.

## Optional next steps

- Put Cloudflare Access in front of `/edit` and `/admin/*` to remove the
  password, login page, and cookie code entirely.
- Add a WAF rate-limit rule on `POST /items/:id/reserve` if the public link is
  ever shared widely.
- Add a deploy workflow using `wrangler deploy`, gated on the CI job, with
  `CLOUDFLARE_API_TOKEN`, `EDIT_PASSWORD`, and `SESSION_SECRET` as repository
  secrets.

## License

GPL-3.0. See [LICENSE](LICENSE).
