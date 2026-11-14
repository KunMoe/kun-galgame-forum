# kungal — Docker

Container build + compose for **kungal** (`kun-galgame-forum`): the Go Fiber API
(`apps/api`) and the Nuxt 4 SSR site (`apps/web`).

kungal is **not** the infra. Postgres / Redis (and the OAuth, image, and
wiki services) are owned by **nextmoe-infra** and shared. kungal ships only its
own `api` + `web`; its `docker-compose.yml` joins infra's external network
(`kun-galgame-infra_default`), so **nextmoe-infra must be running first** — both
locally and in production. (Same shape as moyu — no self-contained override.)

## Images

| Image            | Dockerfile            | Base (runtime)                         | Notes |
|------------------|-----------------------|----------------------------------------|-------|
| `kungal/api`     | `docker/go.Dockerfile`  | `distroless/static-debian12:nonroot` | `CGO_ENABLED=0`, pure Go, ~no shell |
| `kungal/migrate` | `docker/go.Dockerfile`  | same (`--build-arg CMD=migrate`)     | one-off job, `profiles: [jobs]` |
| `kungal/web`     | `docker/nuxt.Dockerfile`| `node:24.22.1-trixie-slim`              | serves `.output` (Nitro node-server) |

Both Dockerfiles take the **repo root** as build context (it's a pnpm workspace —
`apps/web` + `apps/api` — so the lockfile and every manifest must be in context).
KunUI is a published npm dependency (`@kungal/ui-nuxt`), not a local source layer.

## Host ports

| Service | Container | Host (compose) | Why 1xxxx |
|---------|-----------|----------------|-----------|
| api     | 2334      | **15012**      | coexist with a running `air` dev server (2334) |
| web     | 7777      | **15013**      | coexist with `nuxt dev` (2333) |

(Postgres / Redis are infra's — kungal publishes no DB ports of its own.)

## Configure

```bash
cp docker/api.env.example docker/api.env   # API secrets + service URLs
cp docker/web.env.example docker/web.env   # Nuxt runtime overrides
```

`docker/*.env` is gitignored and read at **runtime** via `env_file` — never baked
into an image.

### The api URL is two values in a container

Nuxt SSR and the browser reach the API differently:

- `NUXT_API_BASE_URL` → **SSR** (server). Internal docker network, by service
  name: `http://kungal-api:2334`.
- `NUXT_PUBLIC_API_BASE_URL` → **browser**. Host port / public domain:
  `http://localhost:15012` (or `https://www.kungal.com`).

Dev sets a single `API_BASE_URL` for both; in a container that breaks SSR. The
`web.env.example` splits them — keep both.

## Run — local (against infra)

kungal has no DB of its own; bring up **nextmoe-infra** first (it owns
Postgres / Redis and creates the `kun-galgame-infra_default` network), then start
kungal on that shared network — same flow as moyu.

```bash
# 1) infra up first (creates the network + Postgres/Redis) — in the infra repo:
( cd ../nextmoe-infra && docker compose up -d postgres redis )

# 2) kungal on the shared network:
C="docker compose"
$C build
$C run --rm migrate            # default set — see migration order below
$C up -d api web
# api  → http://localhost:15012/healthz
# web  → http://localhost:15013
```

## Run — production (umbrella)

The umbrella `website/compose.yaml` `include:`s this file and provides the shared
`postgres` / `redis` / `oauth` / `image` / `galgame` services. kungal's `api` +
`web` resolve them by service name. **Cross-repo prerequisite:** kungal stores its
data in a `kungalgame` database on the shared Postgres, so infra must create it —
add `CREATE DATABASE kungalgame;` to infra's `docker/initdb.d/`.

## Migration order (important)

`cmd/migrate` defaults to `-exclude=005,006,007,012,015`: those are the
**post-OAuth-migration** steps and must run *after* the OAuth-side
`migrate-users` (and the galgame-wiki service migrations) have completed —
running them early would backfill cursors / state against pre-remap user IDs.
`015` ALTERs `kungal_user_state`, which `007` creates, so it must run after `007`.

```bash
# 1) routine first pass (everything except the deferred five)
$C run --rm migrate

# 2) AFTER oauth migrate-users + galgame migrations — ascending order in one run
$C run --rm migrate --only=005,006,007,012,015
```

Flags: `-dir up|down`, `-step N`, `-only=<prefixes>`, `-exclude=<prefixes>`.
For a fresh cross-repo bootstrap follow the project migration runbook in `docs/`.

## Healthchecks

- **api** — distroless has no shell, so the binary self-probes: the compose
  healthcheck is `["CMD","/app","healthcheck"]`, which GETs `/healthz` and exits
  0/1 (`pkg/health` + `cmd/server`). `/healthz` is a plain 200 liveness check —
  deliberately no DB/Redis ping.
- **web** — node-slim has no curl; a tiny inline `net.connect(7777)` TCP probe.
