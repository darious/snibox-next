// Seed data — mixed homelab + dev. Each item conforms to the spec model.
window.SEED_ITEMS = [
  {
    id: "i01",
    title: "Jellyfin docker-compose",
    body: `services:
  jellyfin:
    image: jellyfin/jellyfin:latest
    container_name: jellyfin
    user: 1000:1000
    network_mode: host
    volumes:
      - ./config:/config
      - ./cache:/cache
      - /mnt/media:/media:ro
    restart: unless-stopped
    environment:
      - JELLYFIN_PublishedServerUrl=https://watch.lan
`,
    type: "snippet",
    language: "yaml",
    url: null,
    tags: ["docker", "homelab", "media"],
    pinned: true,
    archived: false,
    created_at: "2026-04-12T09:21:00Z",
    updated_at: "2026-05-18T14:02:00Z",
  },
  {
    id: "i02",
    title: "Generate ed25519 SSH key",
    body: `# SSH key generation

Modern default. **ed25519** is shorter, faster, and as secure as RSA-4096.

\`\`\`bash
ssh-keygen -t ed25519 -C "alex@homelab" -f ~/.ssh/id_homelab
\`\`\`

Then add to agent:

\`\`\`bash
ssh-add --apple-use-keychain ~/.ssh/id_homelab
\`\`\`

> Copy the **public** half (\`.pub\`) to the server's \`~/.ssh/authorized_keys\`.
`,
    type: "note",
    language: null,
    url: null,
    tags: ["ssh", "security"],
    pinned: false,
    archived: false,
    created_at: "2026-03-02T11:10:00Z",
    updated_at: "2026-03-02T11:10:00Z",
  },
  {
    id: "i03",
    title: "Tailscale admin console",
    body: "Device list, ACLs, key expiry. Bookmark for the homelab tailnet.",
    type: "link",
    language: null,
    url: "https://login.tailscale.com/admin/machines",
    tags: ["tailscale", "homelab", "network"],
    pinned: true,
    archived: false,
    created_at: "2026-01-18T20:44:00Z",
    updated_at: "2026-05-10T08:01:00Z",
  },
  {
    id: "i04",
    title: "Find largest files on disk",
    body: `du -ah / 2>/dev/null | sort -hr | head -n 20`,
    type: "snippet",
    language: "bash",
    url: null,
    tags: ["bash", "ops"],
    pinned: false,
    archived: false,
    created_at: "2026-02-19T07:55:00Z",
    updated_at: "2026-02-19T07:55:00Z",
  },
  {
    id: "i05",
    title: "Go: context with cancellation",
    body: `func fetchAll(ctx context.Context, urls []string) ([]Result, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    g, ctx := errgroup.WithContext(ctx)
    results := make([]Result, len(urls))
    for i, u := range urls {
        i, u := i, u
        g.Go(func() error {
            r, err := fetch(ctx, u)
            if err != nil {
                return err
            }
            results[i] = r
            return nil
        })
    }
    return results, g.Wait()
}`,
    type: "snippet",
    language: "go",
    url: null,
    tags: ["go", "concurrency"],
    pinned: false,
    archived: false,
    created_at: "2026-04-30T16:11:00Z",
    updated_at: "2026-05-09T10:22:00Z",
  },
  {
    id: "i06",
    title: "Self-hosting reading list",
    body: `# Self-hosting reading list

A running list of posts and docs that shaped how I run the homelab.

## Storage
- ZFS primer (Klara Systems)
- Snapraid + Mergerfs vs. ZFS — the long argument
- Restic encryption design doc

## Networking
- Tailscale: how it actually routes
- Wireguard whitepaper
- CGNAT bypass patterns

## Habits
- Nightly snapshots > weekly backups
- Boring tech > shiny tech
- Document on day one, not "later"
`,
    type: "note",
    language: null,
    url: null,
    tags: ["homelab", "reading"],
    pinned: false,
    archived: false,
    created_at: "2026-01-05T19:00:00Z",
    updated_at: "2026-05-15T22:30:00Z",
  },
  {
    id: "i07",
    title: "Proxmox VE documentation",
    body: "Official admin guide. Reference for cluster, storage, backup.",
    type: "link",
    language: null,
    url: "https://pve.proxmox.com/pve-docs/",
    tags: ["proxmox", "homelab", "docs"],
    pinned: false,
    archived: false,
    created_at: "2025-11-12T12:00:00Z",
    updated_at: "2025-11-12T12:00:00Z",
  },
  {
    id: "i08",
    title: "SQL window: running total per user",
    body: `SELECT
  user_id,
  created_at,
  amount,
  SUM(amount) OVER (
    PARTITION BY user_id
    ORDER BY created_at
    ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
  ) AS running_total
FROM payments
ORDER BY user_id, created_at;`,
    type: "snippet",
    language: "sql",
    url: null,
    tags: ["sql", "postgres"],
    pinned: false,
    archived: false,
    created_at: "2026-03-28T13:14:00Z",
    updated_at: "2026-03-28T13:14:00Z",
  },
  {
    id: "i09",
    title: "Python venv cheatsheet",
    body: `# Python venv cheatsheet

\`\`\`bash
python3 -m venv .venv
source .venv/bin/activate
pip install -U pip
pip install -r requirements.txt
\`\`\`

Freeze:

\`\`\`bash
pip freeze > requirements.txt
\`\`\`

Use \`uv\` if you want it 10–100x faster.
`,
    type: "note",
    language: null,
    url: null,
    tags: ["python"],
    pinned: false,
    archived: false,
    created_at: "2026-02-08T09:00:00Z",
    updated_at: "2026-02-08T09:00:00Z",
  },
  {
    id: "i10",
    title: "Cloudflare tunnel config",
    body: `tunnel: homelab
credentials-file: /etc/cloudflared/homelab.json

ingress:
  - hostname: jellyfin.example.com
    service: http://127.0.0.1:8096
  - hostname: grafana.example.com
    service: http://127.0.0.1:3000
  - service: http_status:404
`,
    type: "snippet",
    language: "yaml",
    url: null,
    tags: ["cloudflare", "tunnel", "homelab"],
    pinned: false,
    archived: false,
    created_at: "2026-04-02T18:21:00Z",
    updated_at: "2026-04-02T18:21:00Z",
  },
  {
    id: "i11",
    title: "Nginx reverse proxy block",
    body: `server {
    listen 443 ssl http2;
    server_name app.lan;

    ssl_certificate     /etc/ssl/lan/fullchain.pem;
    ssl_certificate_key /etc/ssl/lan/privkey.pem;

    location / {
        proxy_pass         http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   Upgrade           $http_upgrade;
        proxy_set_header   Connection        "upgrade";
    }
}`,
    type: "snippet",
    language: "nginx",
    url: null,
    tags: ["nginx", "proxy"],
    pinned: false,
    archived: false,
    created_at: "2026-03-14T11:42:00Z",
    updated_at: "2026-03-14T11:42:00Z",
  },
  {
    id: "i12",
    title: "Caddyfile — auto TLS",
    body: `app.example.com {
    reverse_proxy 127.0.0.1:3000
    encode zstd gzip
    log {
        output file /var/log/caddy/app.log
    }
}`,
    type: "snippet",
    language: "caddyfile",
    url: null,
    tags: ["caddy", "proxy"],
    pinned: false,
    archived: false,
    created_at: "2026-04-21T20:08:00Z",
    updated_at: "2026-04-21T20:08:00Z",
  },
  {
    id: "i13",
    title: "Raycast extensions API",
    body: "API docs for building local Raycast extensions in TypeScript.",
    type: "link",
    language: null,
    url: "https://developers.raycast.com/api-reference",
    tags: ["raycast", "docs"],
    pinned: false,
    archived: false,
    created_at: "2026-02-22T14:18:00Z",
    updated_at: "2026-02-22T14:18:00Z",
  },
  {
    id: "i14",
    title: "Old dotfiles bootstrap (deprecated)",
    body: `# Old bootstrap

Used Ansible. Switched to chezmoi in March 2026 — keeping this around in case I revert.
`,
    type: "note",
    language: null,
    url: null,
    tags: ["dotfiles", "archive"],
    pinned: false,
    archived: true,
    created_at: "2025-08-03T10:00:00Z",
    updated_at: "2026-03-09T10:00:00Z",
  },
  {
    id: "i15",
    title: "Homelab infra inventory",
    body: `# Homelab inventory

| Host        | Role              | OS         | Notes                |
|-------------|-------------------|------------|----------------------|
| pve-01      | Proxmox node      | Debian 12  | 64GB, 6×8TB ZFS      |
| nas-01      | Storage           | TrueNAS    | mirror of pve-01     |
| edge-01     | Reverse proxy     | Alpine     | Caddy + Cloudflared  |
| pi-dns      | DNS / DHCP        | Raspbian   | Pi-hole + Unbound    |

Spare capacity ~ 38%. Next purchase: another 8TB drive in Q3.
`,
    type: "note",
    language: null,
    url: null,
    tags: ["homelab", "inventory"],
    pinned: true,
    archived: false,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-05-19T21:00:00Z",
  },
  {
    id: "i16",
    title: "React: useEffect cleanup gotcha",
    body: `# useEffect cleanup

If you return a function from \`useEffect\`, it runs **before** the next effect and on unmount.

\`\`\`js
useEffect(() => {
  const id = setInterval(tick, 1000);
  return () => clearInterval(id);
}, []);
\`\`\`

Forgetting the cleanup is the #1 source of "ghost" timers and listeners in strict mode.
`,
    type: "note",
    language: null,
    url: null,
    tags: ["react", "javascript"],
    pinned: false,
    archived: false,
    created_at: "2026-05-01T16:00:00Z",
    updated_at: "2026-05-01T16:00:00Z",
  },
  {
    id: "i17",
    title: "Grafana dashboard — home",
    body: "Main overview: hosts, ZFS pools, network throughput.",
    type: "link",
    language: null,
    url: "https://grafana.lan/d/home",
    tags: ["grafana", "monitoring"],
    pinned: false,
    archived: false,
    created_at: "2026-03-30T08:45:00Z",
    updated_at: "2026-03-30T08:45:00Z",
  },
  {
    id: "i18",
    title: "Postgres backup → restic",
    body: `#!/usr/bin/env bash
set -euo pipefail

export PGPASSWORD="$(cat /etc/pg.pass)"
DATE=$(date +%F)

pg_dump -h db.lan -U backup -Fc app \\
  | restic --repo /backups/pg backup \\
    --stdin --stdin-filename "app-$DATE.dump"

restic --repo /backups/pg forget \\
  --keep-daily 7 --keep-weekly 4 --keep-monthly 6 --prune
`,
    type: "snippet",
    language: "bash",
    url: null,
    tags: ["postgres", "backup", "restic"],
    pinned: false,
    archived: false,
    created_at: "2026-02-11T22:30:00Z",
    updated_at: "2026-04-04T12:00:00Z",
  },
  {
    id: "i19",
    title: "NixOS flake — minimal server",
    body: `{
  description = "Minimal server flake";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }: {
    nixosConfigurations.edge-01 = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        ./hardware-configuration.nix
        ({ pkgs, ... }: {
          networking.hostName = "edge-01";
          services.openssh.enable = true;
          environment.systemPackages = with pkgs; [ vim git ];
          system.stateVersion = "25.05";
        })
      ];
    };
  };
}`,
    type: "snippet",
    language: "nix",
    url: null,
    tags: ["nix", "nixos"],
    pinned: false,
    archived: false,
    created_at: "2026-04-18T09:15:00Z",
    updated_at: "2026-04-18T09:15:00Z",
  },
  {
    id: "i20",
    title: "SSH config — homelab hosts",
    body: `Host pve
    HostName 10.0.0.10
    User alex
    IdentityFile ~/.ssh/id_homelab
    ForwardAgent yes

Host nas
    HostName 10.0.0.11
    User alex
    IdentityFile ~/.ssh/id_homelab

Host edge
    HostName edge.lan
    User root
    IdentityFile ~/.ssh/id_homelab
    Port 2222
`,
    type: "snippet",
    language: "ini",
    url: null,
    tags: ["ssh", "config", "homelab"],
    pinned: false,
    archived: false,
    created_at: "2026-01-20T08:00:00Z",
    updated_at: "2026-05-02T11:30:00Z",
  },
];
