# snibox-next

Self-hosted, single-user snippets / notes / links library. Modern dark-mode
reimagining of [snibox/snibox](https://github.com/snibox/snibox). Single
static binary, single SQLite file, HTMX UI, deployed behind your existing
reverse proxy.

See [`SPEC.md`](SPEC.md) for the full design spec. This README covers
running and contributing.

## Stack

- Go 1.22+ (`net/http` + chi)
- [`a-h/templ`](https://templ.guide) for HTML, HTMX 2.x for interactivity
- SQLite via `modernc.org/sqlite` (pure-Go, CGO-free — no toolchain headaches)
- `gomarkdown`/goldmark for markdown, `alecthomas/chroma` for snippet highlighting
- All assets embedded via `embed.FS`

## Auth (read this first)

**Auth is out of scope.** v1 trusts whoever can reach the port. The
binary refuses to bind to a non-loopback address unless you pass
`--trust-network` — see [SPEC §1.1](SPEC.md#11-auth-assumption-v1).

Deploy behind a reverse proxy that handles authentication. Confirmed-working
setups:

- nginx-proxy-manager + its built-in access list
- Authelia / Authentik in front of nginx or Caddy
- Tailscale (ingress restricted to your tailnet)
- Caddy with `basic_auth`

There is no login UI, no session table, no CSRF logic. If you need
multi-user, fork it — don't bolt it on.

## Running

### From source

```sh
go install github.com/a-h/templ/cmd/templ@latest
templ generate
go run ./cmd/snibox --seed-demo
```

Opens on `http://127.0.0.1:8080`.

### Docker

```sh
docker compose up -d
```

By default the compose file publishes only on `127.0.0.1:8080` — put a
proxy in front of it. Data is persisted under `./data/snibox.db`.

### Flags / env

| Flag | Env | Default | Effect |
|------|-----|---------|--------|
| `--addr` | `SNIBOX_ADDR` | `127.0.0.1:8080` | Listen address. |
| `--db` | `SNIBOX_DB` | `./snibox.db` | SQLite database path. |
| `--seed-demo` | `SNIBOX_SEED_DEMO` | `false` | Import 20 demo items on empty DB. Dev/preview only — never use in production. |
| `--read-only` | `SNIBOX_READ_ONLY` | `false` | Block all `POST`/`PUT`/`PATCH`/`DELETE` and `/import`. Useful for safe browsing from a phone or demoing read access. |
| `--trust-network` | `SNIBOX_TRUST_NETWORK` | `false` | Permit non-loopback bind. Required when running behind Docker / a proxy. |

## Development

```sh
templ generate                          # rebuild .templ → .go
go test ./...                            # full test suite
go build -o snibox ./cmd/snibox          # single binary
```

Test layout:

- `internal/store` — repo unit tests against an in-memory SQLite. Covers
  CRUD, search, filters, tag triggers, pagination.
- `internal/importer` — JSON import/export, mode semantics, round-trip.
- `internal/handlers` — HTTP route tests through `httptest.Server` with a
  real SQLite backend.

No mocks. Every test hits an actual SQLite database.

## Project layout

```
cmd/snibox/             entrypoint (flags, server boot)
internal/
  store/                repo + migrations + sqlite triggers
  handlers/             chi routes + HTMX content-negotiation
  views/                templ components (page + partials)
  importer/             JSON import/export
  markdown/             goldmark + chroma wrappers
  id/                   ULID helper
  assets/               embed.FS for static + seed.json
docs/reference/         original prototype HTML/JSX/CSS (read-only reference)
```

## Acceptance status

See SPEC §10. As of initial commit:

- [x] Boots, creates `snibox.db`, serves on `:8080`.
- [x] `--seed-demo` populates 20 items.
- [x] All views render.
- [x] HTMX CRUD round trip for snippet/note/link, no full page reload.
- [x] Search across title/body/tag/language/url via `search_blob` LIKE.
- [x] Tag chips AND with search and type filters.
- [x] Snippets highlighted with classed chroma CSS; Copy button copies raw body.
- [x] Notes render markdown server-side (goldmark).
- [x] `/export.json` ↔ `/import?mode=replace` round-trip; timestamps preserved per SPEC §6.2.
- [x] `go test ./...` green; store 78.5%, importer 89.1%, handlers covered.
- [x] Static binary ~21 MB.

## License

MIT — see [LICENSE](LICENSE).
