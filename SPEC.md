# Snibox Next — Build Spec

A self-hosted, single-user snippets/notes/links library. Modern dark-mode reimagining
of [snibox/snibox](https://github.com/snibox/snibox). Reference UI is the React
prototype in this repo (`Snibox Next.html`) — match its layout, density, and
interaction model.

---

## 1. Stack

| Concern        | Choice                                                |
|----------------|-------------------------------------------------------|
| Language       | Go 1.22+                                              |
| HTTP           | `net/http` + `chi` router                             |
| Templating     | [`a-h/templ`](https://templ.guide)                    |
| Interactivity  | HTMX 2.x + minimal vanilla JS                         |
| Database       | SQLite via `modernc.org/sqlite` (CGO-free)            |
| Migrations     | `golang-migrate` or embedded `goose`                  |
| Markdown       | `gomarkdown/markdown` or `yuin/goldmark`              |
| Highlight      | `alecthomas/chroma` (server-side, classes mode)       |
| Build          | `air` for dev reload, single static binary for prod   |
| Assets         | Embedded via `embed.FS`                               |

**Non-goals:** auth, multi-user, sync, mobile app, plugins, AI features.
Single binary + single SQLite file deployed behind reverse proxy on a homelab.

### 1.1 Auth assumption (v1)

Auth is **out of scope**. The app trusts whoever can reach it. Deployment
expectation: place it behind a reverse proxy that handles authentication
(nginx-proxy-manager, Authelia, Caddy basic_auth, Tailscale-only ingress, etc.).

To keep that assumption explicit:

- The server binds to `127.0.0.1` by default. Public binds (`0.0.0.0`, `::`)
  are refused unless `--trust-network` is passed.
- No CSRF protection (single-user, same-origin only, no cookies).
- No login UI, no session table, no user concept anywhere in the schema.

If you need multi-user later, that's a fork, not a feature flag.

---

## 2. Data model

### 2.1 Item

```go
type Item struct {
    ID        string    // ULID or nanoid, 12+ chars
    Title     string    // required, max 200
    Body      string    // markdown for notes, raw code for snippets, description for links
    Type      string    // "snippet" | "note" | "link"
    Language  *string   // non-null only when Type=="snippet"; chroma lexer name
    URL       *string   // non-null only when Type=="link"
    Tags      []string  // lowercase, no leading '#', no spaces, max 32 per item
    Pinned    bool
    Archived  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2.2 SQL schema

```sql
CREATE TABLE items (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('snippet','note','link')),
    language    TEXT,
    url         TEXT,
    pinned      INTEGER NOT NULL DEFAULT 0,
    archived    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    search_blob TEXT NOT NULL DEFAULT ''   -- lowercased concatenation
                                           -- of title|body|language|url|tags
);

CREATE INDEX idx_items_type      ON items(type)     WHERE archived = 0;
CREATE INDEX idx_items_pinned    ON items(pinned)   WHERE archived = 0;
CREATE INDEX idx_items_archived  ON items(archived);
CREATE INDEX idx_items_updated   ON items(updated_at DESC);

CREATE TABLE tags (
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (item_id, tag)
);
CREATE INDEX idx_tags_tag ON tags(tag);

-- Triggers keep search_blob in sync.
-- See migrations/001_init.sql for actual trigger SQL covering:
--   * AFTER INSERT/UPDATE on items: rebuild search_blob from row + tag join
--   * AFTER INSERT/DELETE on tags: rebuild parent item's search_blob
```

### 2.3 Constraints / invariants

- `language` MUST be `NULL` unless `type='snippet'`.
- `url` MUST be `NULL` unless `type='link'`. URLs must parse via `net/url`.
- Tags are stored lowercased; reject anything matching `[^a-z0-9\-_]`.
- `updated_at` is bumped server-side on every write.
- An archived item cannot also be pinned in derived views (Pinned view filters
  `archived=0`); the underlying column is allowed to be both for round-trip
  fidelity with imports.

---

## 3. Routes

All routes return either a full HTML page (initial load / direct nav) or an
HTMX fragment depending on `HX-Request` header. Use `chi` with content
negotiation helpers.

```
GET    /                          Redirect to /all
GET    /{view}                    Page shell. view ∈ {all,snippets,notes,links,pinned,archive,search}
GET    /tag/{tag}                 Page shell scoped to a tag

GET    /partials/list             List pane fragment. Query params drive filters.
GET    /partials/item/{id}        Main pane fragment for a given item

POST   /items                     Create. Form: title,type,language?,url?,tags
PATCH  /items/{id}                Update one or more fields (HTMX inline)
DELETE /items/{id}                Hard delete
POST   /items/{id}/pin            Toggle pinned
POST   /items/{id}/archive        Toggle archived
POST   /items/{id}/tags           Add tag (form: tag)
DELETE /items/{id}/tags/{tag}     Remove tag

GET    /export.json               Stream JSON array of all items
POST   /import                    Multipart or JSON body; query ?mode=merge|append|replace

GET    /static/*                  Embedded static assets (CSS, htmx.js, fonts)
GET    /healthz                   200 OK
```

### 3.1 List query params

`/partials/list` accepts:

| Param    | Effect                                                          |
|----------|-----------------------------------------------------------------|
| `view`   | scope (one of the views above; default `all`)                   |
| `q`      | search query; when set, overrides view to search results        |
| `tag`    | repeatable; AND semantics                                       |
| `type`   | filter to one of snippet/note/link                              |
| `lang`   | filter snippets by chroma lexer name                            |
| `sort`   | `updated` (default) \| `created` \| `title`                     |
| `cursor` | optional opaque pagination cursor; page size 50                 |

Pinned items always sort to the top **except** in the `archive` view.

---

## 4. Views

Order in the nav rail, matching the prototype:

1. **All** — everything not archived
2. **Snippets** — `type='snippet'` not archived
3. **Notes** — `type='note'` not archived
4. **Links** — `type='link'` not archived
5. **Pinned** — `pinned=1` not archived
6. **Archive** — `archived=1`
7. **Tags** (section header) — list of tags with counts; click filters by tag
8. **Search results** — implicit view shown while `q` is non-empty

---

## 5. UI behavior

Reference the prototype for exact layout. Notable behaviors to preserve:

### 5.1 Layout

v1 is **desktop-first**. No fancy mobile animation work until core CRUD/search
ships solid.

- Desktop ≥ 900px (primary target): three columns — nav rail (220px,
  collapsible to 56px), list (360px), main (fills).
- < 900px (best-effort fallback): single column, plain stack-and-scroll.
  Nav rail above list above main pane. No drawer, no slide-in panel, no JS
  choreography. Use CSS-only `display: block` reflow.

Mobile slide-in / drawer behaviour is explicitly deferred until v2.

### 5.2 List rows

- Type icon (colored), title, pin indicator on the right.
- Preview line: for snippets/links use monospace; for notes use UI font with
  markdown stripped.
- Foot row: up to 3 tags, then `+N`; right-aligned relative timestamp.
- Active row gets a 2px accent left-border and a raised background.

### 5.3 Editor / viewer

| Type    | Behavior                                                                       |
|---------|--------------------------------------------------------------------------------|
| Note    | Side-by-side: textarea on left, rendered markdown on right. Mobile: toggle.    |
| Snippet | Highlighted view by default; toolbar `Edit` flips to a textarea. `Copy` button.|
| Link    | Single card: URL input + Open button + description textarea.                   |

- Title is editable inline in the main header.
- Tags editable inline in the meta bar; `+ tag` shows an inline input.
- Language is a `<select>` in the meta bar for snippets only.
- Pin / Archive / Delete buttons in the main header. Delete confirms.

### 5.4 Server-side highlighting

Use chroma with `html.WithClasses(true)`. Emit a single classed stylesheet at
build time (`chroma --html-styles --style=onedark > static/chroma.css`) and
include it. This keeps the snippet view zero-JS and fast.

### 5.5 Search

- Search uses a denormalised `items.search_blob` column populated by triggers
  (no FTS5 virtual table). The blob is the lowercased concatenation of title,
  body, language, url, and the joined tag list, separated by `\n`.
- For `q` length ≥ 2: `WHERE search_blob LIKE '%' || lower(q) || '%'`.
- For `q` length 1: `WHERE title LIKE ...` only (avoid useless full-body scan).
- Search is **across**: title, body, tags, language, url. Sort is `updated_at
  DESC` (no ranking — single-user library, query latency is the bottleneck
  before relevance is).
- Tag filters AND with search.
- Search results view is implicit — entering the search box switches the
  list pane title to "Search results" without changing the URL. Clearing q
  returns to the previous view.

Rationale for dropping FTS5: single-user homelab scale (≤ tens of thousands of
items) makes a plain `LIKE` on an indexed-by-prefix-or-not text column fine.
Schema stays portable, no virtual tables to maintain through migrations.

### 5.6 HTMX patterns

- Search input: `hx-get="/partials/list" hx-trigger="input changed delay:120ms" hx-target="#list-scroll" hx-push-url="true"`.
- Row click: `hx-get="/partials/item/{id}" hx-target="#main" hx-swap="innerHTML"`. Push URL.
- Pin/archive: `hx-post` with `hx-swap="outerHTML"` on the button + OOB swap to update the row in the list.
- Inline title save: `hx-patch` on blur or Enter.
- Toast feedback via `HX-Trigger: {"toast":"Pinned"}` response header + a tiny JS listener.

### 5.7 Keyboard shortcuts

| Key       | Action                       |
|-----------|------------------------------|
| `/`       | Focus search                 |
| `n`       | Open New-item modal          |
| `Esc`     | Close modals, blur input     |
| `j` / `k` | Next / previous item in list |
| `Enter`   | Open focused item            |
| `e`       | Toggle edit on snippet       |
| `p`       | Toggle pin on current item   |

---

## 6. Import / export

### 6.1 Export

`GET /export.json` streams a JSON array of items in the model order. Field
names are snake_case:

```json
[
  {
    "id": "01HX...",
    "title": "Jellyfin docker-compose",
    "body": "services:\n  jellyfin: ...",
    "type": "snippet",
    "language": "yaml",
    "url": null,
    "tags": ["docker","homelab","media"],
    "pinned": true,
    "archived": false,
    "created_at": "2026-04-12T09:21:00Z",
    "updated_at": "2026-05-18T14:02:00Z"
  }
]
```

### 6.2 Import

`POST /import?mode=merge|append|replace` accepts the same shape.

- `merge` (default): upsert by `id`. Preserve imported `created_at` /
  `updated_at` on inserted **and** updated rows.
- `append`: assign new IDs, insert all. **Regenerate** `created_at` /
  `updated_at` to "now" — these are new local copies.
- `replace`: wipe the table inside a transaction, then insert. Preserve
  imported `created_at` / `updated_at`.

Validate every record. Return `400` with a JSON list of per-record errors on
failure; the entire import is wrapped in a transaction.

---

## 7. Theme & assets

- Warm-dark palette using `oklch()`. Tokens live in `static/theme.css`; do
  not introduce additional palettes. Accent is `#3ecf8e`.
- Inter (UI) + JetBrains Mono (code). Self-host the `.woff2` files under
  `static/fonts/` — no Google Fonts CDN in production.
- Icons: lucide static SVGs in `static/icons/`, inlined via templ helper.
- Embed everything (`go:embed`) so the binary is self-contained.

---

## 8. File layout

```
snibox-next/
├── cmd/snibox/main.go             # entrypoint, flag parsing, server bootstrap
├── internal/
│   ├── store/                     # SQLite repo (CRUD, search, FTS triggers)
│   ├── handlers/                  # chi handlers; one file per route group
│   ├── views/                     # templ components (page shells + partials)
│   ├── importer/                  # JSON import/export
│   ├── markdown/                  # gomarkdown wrapper, chroma renderer
│   └── id/                        # ULID helper
├── static/                        # css, js, fonts, icons (embedded)
├── migrations/                    # SQL files
├── testdata/                      # seed JSON for dev + tests
└── README.md
```

---

## 9. Configuration

Flags + env, env wins:

```
--addr           127.0.0.1:8080  SNIBOX_ADDR            # see §1.1
--db             ./snibox.db     SNIBOX_DB
--seed-demo      false           SNIBOX_SEED_DEMO       # load testdata/seed.json on empty db
--read-only      false           SNIBOX_READ_ONLY       # 405 on POST/PUT/PATCH/DELETE + /import
--trust-network  false           SNIBOX_TRUST_NETWORK   # allow 0.0.0.0 bind (see §1.1)
```

No config file. No remote anything.

`--seed-demo` is the only way to import the bundled demo data; it must be
passed explicitly and is intended for dev / first-run preview, not production.

---

## 10. Acceptance criteria

A build is "done" when:

1. `go run ./cmd/snibox` boots, creates `snibox.db`, serves on `:8080`.
2. Empty DB + `--seed` populates the 20 demo items from `testdata/seed.json`.
3. All 8 views render and route correctly.
4. Create → edit → pin → archive → unarchive → delete round trip works for
   each of the three types, with no full-page reload (HTMX swaps only).
5. Search finds an item by a substring of its title, body, tag, language,
   and URL — independently.
6. Tag filter chips AND with search and type filters.
7. Snippets render with chroma-class highlighting; copy button copies raw body.
8. Notes render markdown server-side (no client markdown lib).
9. `/export.json` round-trips through `/import?mode=replace` byte-identical
   modulo `updated_at` regeneration policy (decide and document).
10. Lighthouse mobile score ≥ 95 performance, ≥ 95 accessibility.
11. `go test ./...` green; minimum coverage: `store` 80%, `importer` 90%.
12. Single static binary < 25 MB; cold start < 100 ms on a Pi 4.

---

## 11. Out of scope (do not build)

- Authentication, sessions, users, sharing, ACLs
- Calendar / timeline / dashboard widgets
- Tasks, reminders, deadlines, projects
- Wiki-style backlinks or `[[…]]` syntax
- AI: summarization, embeddings, semantic search
- File attachments, image uploads
- Cloud sync, S3 backup, webhooks
- Plugins / extensions / themes UI
- Multi-pane navigation history (back-stack beyond browser default)

If a feature isn't listed in this spec, leave it out. Ship the small thing.
