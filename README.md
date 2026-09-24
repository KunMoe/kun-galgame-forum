![kun-galgame-forum](https://kungal.com/kungalgame.webp)

### **[English](README.md)** | **[日本語](docs/readme/jp.md)** | **[简体中文](docs/readme/chs.md)** | **[繁體中文](docs/readme/cht.md)**

**Contact us: [Telegram](https://telegram.me/kungalgame) | [Discord](https://discord.com/invite/5F4FS2cXhX)**

The image is sourced from the game [Ark Order](https://apps.qoo-app.com/en/app/9593) and features the character KUN (鲲).

> **AI-assisted development:** Since version **5.1.0**, this project has used LLM-assisted development tools, including Codex and Claude Code. All code through version **5.0.70** was written entirely by hand. The last fully hand-written revision is [v5.0.70 (commit b4ad59e)](https://github.com/KunMoe/kun-galgame-forum/tree/b4ad59eb77d3eaf36d082aa528651039816e1dfa).

# 鲲 Galgame 论坛 (KUN Galgame Forum)

## Website Introduction

鲲 Galgame is a community of people who love visual novels and Galgames. Its public sites include:

- [鲲 Galgame 论坛](https://www.kungal.com) (this project)
- [鲲 Galgame 表情包](https://sticker.kungal.com)
- [鲲 Galgame 开发文档](https://soft.moe/kun-visualnovel-docs/kun-forum.html)
- [鲲 Galgame 导航](https://nav.kungal.org/)
- [鲲 Galgame 补丁站](https://www.moyu.moe)
- [鲲 Galgame 论坛状态页](https://down.kungal.com/)

For more information, visit [About 鲲 Galgame](https://www.kungal.com/kungalgame).

## Features

- **Visual novel catalog** — Community submissions and review backed by the shared Catalog registry, with VNDB references, multilingual metadata, ratings, tags, engines, companies, and release information
- **Resource sharing** — Game patches, translations, voice packs, and other resources with provider tracking plus platform and language filters
- **Discussion forum** — Topics, replies, nested comments, polls, reactions, upvotes, favorites, drafts, and moderation workflows
- **Collaborative editing** — Git-style proposals, review, revision history, reverts, and contributor credits through the Catalog editing engine
- **Private messaging and chat** — Direct messages, contacts, read state, reactions, and recall support
- **Moemoepoint** — An ecosystem-wide contribution score whose authoritative balance and ledger are owned by OAuth
- **Rich content editing** — The shared KUN editor, built on Milkdown and CodeMirror, with KaTeX, syntax highlighting, mentions, and image uploads
- **Personalized appearance** — Light and dark modes, configurable transparency, fonts, background images, and content-rating preferences
- **Search and discovery** — Site-wide search across topics, replies, comments, users, and visual novels, plus rankings, activity feeds, release calendars, collections, and recommendations
- **SEO and feeds** — Nuxt SSR, Schema.org metadata, chunked sitemaps, canonical redirects, and RSS feeds for topics and visual novels

## Architecture

This repository is a **pnpm workspace monorepo** containing a Go API and a Nuxt frontend. It is a downstream application of [`nextmoe-infra`](https://github.com/next-moe/nextmoe-infra), the identity and shared-service hub for the KUN ecosystem.

| Package    | Responsibility                                                                                                                                                                         |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `apps/api` | **Go Fiber v3 BFF and REST API** — owns forum data and interactions, enriches responses, talks to shared services, and runs scheduled synchronization jobs                             |
| `apps/web` | **Nuxt 4 SSR frontend** — Vue pages, components, Pinia state, validation, SEO, and browser interaction; Nitro serves feeds, sitemap data, content-image handling, and legacy redirects |

The forum owns topics, replies, messages, resources, ratings, quizzes, collections, toolsets, local counters, and its PostgreSQL schema. Shared capabilities remain single-sourced in infra:

| Capability                                      | Source of truth                                                                               |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------- |
| Identity and user profile                       | OAuth; numeric user IDs stay identical across services                                        |
| Browser authentication                          | An opaque `kungal_session` cookie backed by Redis; OAuth tokens never live in browser storage |
| Moemoepoint balance and ledger                  | OAuth; the forum keeps only a cached local view where required                                |
| Visual novel metadata, claims, and edit history | Catalog; the forum galgame id is the catalog work id, with no mapping                         |
| Avatars and content images                      | The content-addressed `image_service`                                                         |
| Galgame and resource comment threads            | The shared Community service                                                                  |
| Reports and enforcement                         | The shared Trust service                                                                      |
| Toolset archives                                | The shared Artifact service                                                                   |

The retired Galgame Wiki family is not a contract for new work. Some compatibility reads and notification surfaces remain temporarily while the catalog cutover finishes, but new submission, claim, review, and editing flows target Catalog.

## Tech Stack

| Layer                | Technology                                                                                                             |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Frontend             | [Nuxt 4](https://nuxt.com/) + Vue 3 SSR (Nitro node-server)                                                            |
| UI                   | `@kungal/ui-nuxt` and `@kungal/ui-vue`                                                                                 |
| Editor               | `@kungal/editor-nuxt`, Milkdown, and CodeMirror                                                                        |
| Styling              | [Tailwind CSS 4](https://tailwindcss.com/)                                                                             |
| State management     | [Pinia](https://pinia.vuejs.org/) with persisted state                                                                 |
| Backend API          | [Go 1.26](https://go.dev/) + [Fiber v3](https://gofiber.io/)                                                           |
| Database             | PostgreSQL through [GORM](https://gorm.io/), with explicit raw-SQL migrations and no AutoMigrate                       |
| Cache and sessions   | Redis                                                                                                                  |
| Search               | PostgreSQL for forum content; the shared Catalog service for visual novels and taxonomy                                |
| Authentication       | OAuth-backed BFF session with a 90-day sliding lifetime                                                                |
| Shared services      | Catalog, Community, Trust, Image, Artifact, and OAuth clients                                                          |
| Scheduler            | [robfig/cron](https://github.com/robfig/cron) for daily maintenance and Catalog event synchronization                  |
| Validation and tests | Zod, Vitest, Vue Test Utils, and Go tests                                                                              |
| Deployment           | Docker images published to GHCR and deployed with [Dokploy](https://dokploy.com/); legacy PM2 scripts remain available |
| Analytics            | [Umami](https://umami.is/)                                                                                             |

## Project Structure

```text
├── apps/
│   ├── api/                 # Go Fiber v3 BFF / REST API
│   │   ├── cmd/             # server, migrate, and one-off maintenance tools
│   │   ├── internal/        # domain handlers, services, repositories, and models
│   │   ├── migrations/      # explicit .up.sql / .down.sql migrations
│   │   └── pkg/             # shared clients, config, errors, permissions, and utilities
│   └── web/                 # Nuxt 4 SSR frontend
│       ├── app/             # pages, components, composables, Pinia stores, and styles
│       ├── server/          # Nitro feeds, sitemap source, middleware, and redirects
│       └── shared/          # shared TypeScript types and utilities
├── docker/                  # Dockerfiles, environment examples, and deployment notes
├── docker-compose*.yml      # local integration and production stacks
├── scripts/                 # legacy PM2 lifecycle scripts
└── docs/                    # project docs and generated read-only contract mirrors
```

## Getting Started

**Prerequisites:** Node.js 24+ with Corepack, Go 1.26+, PostgreSQL, and Redis. Full local functionality also requires the OAuth, Catalog, Image, Artifact, Community, and Trust services from `nextmoe-infra`.

Start the local shared platform first:

```bash
cd /path/to/nextmoe-infra
docker compose -f docker-compose.dev.yml --profile full up -d
# Optional: refresh the local databases with real-shaped, desensitized data.
./scripts/refresh-dev-db.sh
```

Then configure and start the forum:

```bash
corepack enable
pnpm install

cp apps/api/.env.example apps/api/.env
cp apps/web/.env.example apps/web/.env

# Applies this repository's routine local migrations.
pnpm migrate

# API: http://127.0.0.1:2334, Web: http://127.0.0.1:2333
pnpm dev
```

The checked-in environment examples target the local infra stack. See the [infra development-environment guide](https://github.com/next-moe/nextmoe-infra/blob/main/docs/dev-environment.md) for service profiles, database refreshes, and local credentials. Cross-repository identity migrations have additional ordering requirements documented in the [user-migration runbook](https://github.com/KunMoe/nextmoe-docs/tree/main/docs/shared/user-migration).

To run the forum in containers after the infra network is available:

```bash
docker compose run --rm migrate
docker compose up -d kungal-api web
```

See [docker/README.md](docker/README.md) for the complete container and deployment runbook.

## Scripts

| Command                                                          | Description                                              |
| ---------------------------------------------------------------- | -------------------------------------------------------- |
| `pnpm dev`                                                       | Run API and Web together                                 |
| `pnpm dev:web` / `pnpm dev:api`                                  | Run one application                                      |
| `pnpm build`                                                     | Build the Go API, then the Nuxt frontend                 |
| `pnpm lint` / `pnpm lint:fix`                                    | Check or fix frontend ESLint issues                      |
| `pnpm typecheck`                                                 | Run the frontend `vue-tsc` check                         |
| `pnpm -F web test`                                               | Run frontend Vitest tests                                |
| `pnpm test:api`                                                  | Run Go tests                                             |
| `pnpm vet`                                                       | Run `go vet`                                             |
| `pnpm format`                                                    | Format both applications with Prettier and gofmt         |
| `pnpm migrate` / `pnpm migrate:down`                             | Apply or roll back this repository's database migrations |
| `pnpm prod:deploy` / `prod:start` / `prod:stop` / `prod:restart` | Run the legacy PM2 lifecycle scripts                     |
| `pnpm prod:logs`                                                 | Follow legacy PM2 logs                                   |

## Development Boundaries

- Contract docs live in `nextmoe-infra`. The one mirror here, `docs/oauth/oauth-integration-guide.md`, is generated and read-only: change its source in `nextmoe-infra`, then synchronize it through `nextmoe-docs`. Design docs are kept out of the repository.
- Database schema changes require a numbered migration in `apps/api/migrations/`; the API never runs GORM AutoMigrate at startup.
- Frontend work should use KunUI components before introducing local replacements. KunUI itself is an upstream package and is not modified in this repository.
- Keep frontend and backend response shapes aligned. The Go API returns a stable `{ code, message, data }` envelope.

## Join / Contact Us

- [Telegram Group](https://telegram.me/kungalgame)
- [Twitter / X](https://twitter.com/kungalgame)
- [GitHub Repository](https://github.com/KunMoe/kun-galgame-forum)
- [Discord Group](https://discord.com/invite/5F4FS2cXhX)
- [YouTube Channel](https://youtube.com/@kungalgame)
- [Bilibili](https://space.bilibili.com/1748455574)

## License

This project is licensed under the `AGPL-3.0` license.
