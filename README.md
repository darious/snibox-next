# snibox-next

Self-hosted, single-user snippets / notes / links library. Modern dark-mode
reimagining of [snibox/snibox](https://github.com/snibox/snibox). Single
static binary, single SQLite file, HTMX UI, deployed behind your existing
reverse proxy.

See [`SPEC.md`](SPEC.md) for the full design spec.

## Stack

- Go 1.22+ (`net/http` + chi)
- [`a-h/templ`](https://templ.guide) HTML components, HTMX 2.x for interactivity
- SQLite via `modernc.org/sqlite` (pure-Go, CGO-free — no toolchain headaches)
- `yuin/goldmark` markdown, `alecthomas/chroma` syntax highlighting (server-side, class mode)
- All assets embedded via `embed.FS`

## Auth (read this first)

**Auth is out of scope.** v1 trusts whoever can reach the port. The binary
refuses to bind to a non-loopback address unless you pass `--trust-network`
— see [SPEC §1.1](SPEC.md#11-auth-assumption-v1).

Deploy behind a reverse proxy that handles authentication:
nginx-proxy-manager, Authelia/Authentik, Caddy `basic_auth`, or a Tailscale-only
ingress. No login UI, no session table, no CSRF logic. If you need multi-user,
fork it — don't bolt it on.

## Quick start (local)

```sh
go install github.com/a-h/templ/cmd/templ@latest
templ generate
go run ./cmd/snibox --seed-demo
```

Opens on `http://127.0.0.1:8979`. Adds 20 demo items on first boot.

## Production deployment

The repo ships a `docker-compose.yml` tuned for the **q-docker** stack
pattern (port-published, on the external `web` network, WUD watch labels,
bind-mount under `/mnt/config/q-docker/service-data/`).

### 1. Pull the image

The release workflow publishes multi-arch (`amd64` + `arm64`) images to
GHCR on every push to `main` and on `v*` tags.

```
ghcr.io/darious/snibox-next:latest
ghcr.io/darious/snibox-next:v0.1.0
ghcr.io/darious/snibox-next:sha-<short>
```

GHCR images are public; no `docker login` needed to pull.

### 2. Stack drop-in

Copy `docker-compose.yml` into `/mnt/config/q-docker/stacks/snibox-next/`
(Dockge will manage it). The compose file:

- Pulls `ghcr.io/darious/snibox-next:latest`
- Binds `/mnt/config/q-docker/service-data/snibox-next` → `/data`
- Publishes `:8979`
- Joins the external `web` network so nginx-proxy-manager can reach it
- Labels `wud.watch=true` so WUD notifies on new digests
- Sets `SNIBOX_TRUST_NETWORK=true` (the container binds `0.0.0.0`)

```sh
mkdir -p /mnt/config/q-docker/service-data/snibox-next
cd /mnt/config/q-docker/stacks/snibox-next
docker compose up -d
```

### 3. nginx-proxy-manager host rule

| Field | Value |
|---|---|
| Domain | `snibox.lan` (or whatever) |
| Forward Hostname / IP | `snibox-next` (container name on `web` network) |
| Forward Port | `8979` |
| Block Common Exploits | on |
| Websockets Support | on |
| Access List | **set one** — that is the auth layer |

Add TLS via Let's Encrypt as you do for the other services.

### Bare-metal alternative (no Docker)

If you'd rather run the binary directly:

```sh
# build
CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always)" \
    -o /usr/local/bin/snibox ./cmd/snibox

# systemd unit (/etc/systemd/system/snibox.service)
[Unit]
Description=snibox-next
After=network-online.target

[Service]
Type=simple
User=snibox
WorkingDirectory=/var/lib/snibox
ExecStart=/usr/local/bin/snibox --addr 127.0.0.1:8979 --db /var/lib/snibox/snibox.db
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Flags / env

| Flag | Env | Default | Effect |
|------|-----|---------|--------|
| `--addr` | `SNIBOX_ADDR` | `127.0.0.1:8979` | Listen address. |
| `--db` | `SNIBOX_DB` | `./snibox.db` | SQLite database path. |
| `--seed-demo` | `SNIBOX_SEED_DEMO` | `false` | Import 20 demo items into an empty DB. Dev/preview only. |
| `--read-only` | `SNIBOX_READ_ONLY` | `false` | Block all writes (`POST`/`PUT`/`PATCH`/`DELETE`/`/import`). |
| `--trust-network` | `SNIBOX_TRUST_NETWORK` | `false` | Permit non-loopback bind. Set when behind Docker / a proxy. |
| `--version` | — | — | Print version and exit. |

## Backup & restore

The whole app is one SQLite file (`/data/snibox.db` in the container).
Three safe options:

### A. Online backup with `sqlite3`

Recommended — works while snibox is running.

```sh
docker exec snibox-next sh -c \
  'apk add --no-cache sqlite >/dev/null 2>&1; \
   sqlite3 /data/snibox.db ".backup /data/snibox-$(date +%F).bak"'
```

This uses SQLite's online backup API and is safe with WAL.

### B. `VACUUM INTO`

Compacts and atomically writes a fresh file:

```sh
docker exec snibox-next sh -c \
  'sqlite3 /data/snibox.db "VACUUM INTO '"'"'/data/snibox-$(date +%F).db'"'"'"'
```

### C. JSON export

For a human-readable, version-controllable snapshot:

```sh
curl -s http://localhost:8979/export.json > snibox-$(date +%F).json
```

Restore via `POST /import?mode=replace` with the JSON body. Timestamps are
preserved per [SPEC §6.2](SPEC.md#62-import).

### Restoring from a `.bak` / `.db`

Stop the container, replace the file, start again:

```sh
docker compose down snibox-next
cp snibox-2026-05-21.bak /mnt/config/q-docker/service-data/snibox-next/snibox.db
docker compose up -d snibox-next
```

## Upgrading

If you're on `:latest`, WUD notifies you of new digests; pull + recreate:

```sh
docker compose pull snibox-next
docker compose up -d snibox-next
```

Migrations run automatically on boot from the embedded `migrations/`
directory; existing rows are untouched.

## Development

```sh
templ generate                          # rebuild .templ → .go
go test ./... -race -cover              # full test suite
go build -o snibox ./cmd/snibox         # single binary
```

Tests hit a real SQLite database (in-memory). No mocks.

- `internal/store` — repo unit tests, ~78% covered
- `internal/importer` — JSON import/export, ~89% covered
- `internal/handlers` — HTTP route tests through `httptest.Server`, ~60% covered

## Project layout

```
cmd/snibox/             entrypoint (flags, server boot, --version)
internal/
  store/                repo + migrations + sqlite triggers
  handlers/             chi routes + HTMX content-negotiation + middleware
  views/                templ components (page + partials)
  importer/             JSON import/export
  markdown/             goldmark + chroma wrappers
  id/                   ULID helper
  assets/               embed.FS for static + seed.json
docs/reference/         original prototype HTML/JSX/CSS (read-only reference)
```

## License

MIT — see [LICENSE](LICENSE).
