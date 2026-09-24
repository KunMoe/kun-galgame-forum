# kun-galgame-forum (kungal) — AI Agent Project Guide

## 铁律 (Iron Rules — non-negotiable; these override every other guideline in this file)

1. **No background gradients in any UI, ever — except three sanctioned cases.** Never use gradient backgrounds in UI design (`bg-gradient-*`, `from-*/via-*/to-*`, `linear-gradient()`, `radial-gradient()`, `conic-gradient()`, etc.); use solid colors from the project's palette. **The only three sanctioned exceptions (do NOT remove them in a "no-gradient" sweep)**: (a) the galgame card cover's bottom→top legibility scrim — `bg-gradient-to-t from-black/60 to-transparent` in `apps/web/app/components/galgame/card/Card.vue`; (b) the console ASCII startup banner's text gradient in `apps/web/app/widget/showMoeMessage.ts`; (c) the quiz detail panel's collapsed-state "peek then reveal" fade in `apps/web/app/components/galgame/quiz/DetailPanel.vue`. All three are annotated in-code with a comment pointing back to this rule. A fourth file also matches a `rg 'gradient' apps/web/app` sweep and is **not** a UI background, so the rule does not reach it: the ApexCharts area-series `fill` in `apps/web/app/components/admin/OverviewChart.vue` — the fade under a plotted line. It carries its own annotation saying so. Those four files are the whole result set; anything else the sweep returns is a violation.
2. **Prefer KunUI components; do not modify KunUI itself.** When adding or changing frontend UI, reach for a KunUI component (`@kungal/ui-*`) first — do not hand-roll a native/custom component unless there is genuinely no KunUI equivalent for what you need. If KunUI appears to have a bug or is missing a feature, **do not edit KunUI's code** (it is a shared upstream library) — report it to the user directly instead, and let them decide how to proceed.
3. **gid ≡ work_id: the forum's galgame id is the catalog work id.** `galgame.id`, the number in `/galgame/:id`, every `work_id` column, and every galgame id the forum sends to or reads from catalog, community, trust, or moyu are one number: `catalog_work.id`. There is no mapping between a "forum id" and a "catalog id", and none may be added — no bridge function, no lookup table, no translation through a claim's `product_work_id` / `site_work_id`, no branch that treats the two as different spaces. The name `gid` is retired; write `work_id` / `workId` / `WorkID`. When catalog names an id the forum has no row for, the local row is missing: create it under that id. The two drifted apart once (catalog minted its own sequence in 2026-07, and from gid 923 on the numbers disagreed); every bridge built around the drift misrouted something — 148,548 folder items landed on the wrong work — and the G0 window renumbered the forum onto catalog's ids. Plan and rulings: `docs/proj/gid-is-work-id.md`.


Visual novel / galgame **forum**. `apps/api` = Go Fiber v3 + GORM + Postgres, `apps/web` = Nuxt 4.
This repo is one of the **downstreams of nextmoe-infra (the OAuth / identity / contract hub)** (the other is kun-galgame-patch / moyu).

## Core Engineering Principles

> Shared baseline across all KUN Galgame repositories. Defaults, not dogma — apply judgment.

1. All commit messages must be written entirely in English.
2. All code comments must be written entirely in English.
3. Keep each source file under ~500 lines where practical; once a file grows past ~300 lines, consider splitting it (a guideline, not a hard rule).
4. Write every frontend function as an arrow function; compose/merge class names with `cn` wherever practical.
5. Deliberately balance elegant modularity against necessary duplication — choose per case instead of always favoring either.
6. Constantly verify that frontend and backend agree on the data: field shapes and response formats must match what each side expects.
7. After every change, watch for unintended side effects elsewhere.
8. If a change requires running a migration, tell the user explicitly at the end — which command, and against which database.
9. Always seek the most modern, elegant solution that fits the project's current state; consult the latest official docs and resources online when useful.
10. Never let the pursuit of elegance or modularity make the code complex or hard to follow, and don't write over-defensive code.
11. A Nuxt page — and any component used as a page/route root — must have a **single real root element**: never `display: contents` (generates no box, so the transition can't attach) and never a leading comment / whitespace / sibling at the template root (a comment is itself a root node). Either trips Nuxt's "does not have a single root node" warning and drops the page-transition enter animation (the page appears without animating). Keep explanatory comments *inside* the root element.
12. Reserve the scrollbar gutter globally — `html { scrollbar-gutter: stable }`, with an `overflow-y: scroll` `@supports` fallback — so the document width is constant across routes. Otherwise navigating from a scrolling page to a height-locked one (no scrollbar) removes the classic scrollbar's ~15px and the centered layout shifts sideways: a "teleport" at the tail of the page transition. This is a browser layout fact, not a transition bug. Use single-edge `stable` (`both-edges` is buggy in Chrome); it's a harmless no-op under overlay scrollbars (macOS/iOS).
13. **One task = one Codex session; one assigned target repo = one branch = one worktree.** Never let two sessions write this checkout; prefer `codex-session new kun-galgame-forum <session>`. The launcher exposes source-repo reference material through `$CODEX_SESSION_REFS` when present; read it in place and never copy it into any worktree. A single-repo session may write only its own worktree; an explicitly coordinated cross-repo operation may write only separately assigned target worktrees. Launcher source checkouts and refs are always read-only.
14. DB-backed tests must use only the launcher-provided, explicit, unique `TEST_DATABASE_DSN`; never discover or fall back to a DSN from `.env`, and never print the DSN. Run shared-database Go integration suites with `-count=1 -p 1`, never against a live or rehearsal database.

## Comments

**Default: none.** Code that can be understood by reading it gets no comment. Most code is that code. `[review]`

**A comment is earned by a mistake that already happened, not by one you predict.** Do not comment while writing — you cannot tell yet which parts are traps. Comment when something went wrong there: an agent or a person got it wrong, a review caught it, a test went red, production broke. The comment records the wrong conclusion that was actually reached, so the next reader does not reach it again. If you cannot name the incident, there is no comment to write. `[review]`

Two standing exceptions, where the comment is a record rather than a warning:

- `apps/api/migrations/**` — a migration is history and cannot be re-read from the current schema. Say what it changes and why, including what was done about existing rows.
- A constraint that is true but invisible from this file: a version floor, an upstream bug, a required ordering. `huma/v2 >= v2.39.0` is one; a reader who does not know it will "simplify" the dependency back and break SSE.

Write the conclusion, not the mechanism. `// splitCommand takes the subcommand off before flag.Parse` is a restatement; `flag.Parse stops at the first non-flag argument, so 'migrate down -steps 1' parsed no flags and rolled back nothing` is the trap. Quote real system output verbatim when reproducing a symptom.

Never write: restatements of the code, section banners, `TODO` without an owner, or doc comments that only echo the identifier (`// New creates a new X`). Exported Go identifiers get a doc comment only when the name alone is ambiguous. If a comment explains what a name means, rename the thing and delete the comment.

English, and short. When in doubt, delete it — a wrong comment costs more than a missing one, and the missing one gets written the day it is needed.

### Scope in this repo

Applies to `.go`, `.ts`, and `.vue`. It does **not** apply to config and onboarding files (`.env.example`, `.air.toml`, `docker-compose*.yml`, CI workflows), where the comment is the only documentation surface a person reads before running anything.

Four things a sweep must **not** delete, all of them the invisible-constraint exception:

- The four gradient annotations 铁律 #1 depends on (`galgame/card/Card.vue`, `widget/showMoeMessage.ts`, `galgame/quiz/DetailPanel.vue`, and the out-of-scope note on `admin/OverviewChart.vue`). They are load-bearing: without them the next no-gradient sweep deletes the gradients themselves.
- Cross-service contract invariants (C1–C5) whose evidence lives in nextmoe-infra: identity is the same integer across three databases, the local `users.moemoepoint` is a cached view, catalog `site` and catalog source key are separate identities.
- `docs/{oauth,image_service,artifact}/` — infra's read-only mirrors, never edited here at all.
- Deploy-order and migration hazards (migrate-before-deploy, deploy-then-drop).

## Current catalog cutover state

- The `w161-p3` line moves submission, claims, editing, and the cron inbox from the retired wiki family to catalog. Treat the read-only source-workspace file at `${CODEX_SESSION_REPO%/*}/nextmoe-infra/refs/proj/161-n5-grand-window.md` as the coordination ledger—not a `../` path from this worktree—and verify both branch and deployment state before assuming the cutover is live.
- The old `/galgame/messages/feed` S2S source is retired; the claim-event cron consumes `GET /v2/catalog/claim-events` (exclusive `cur_` cursor, `limit` ≤ 100 or catalog answers 400, needs `claim_events:read` on top of `catalog:read`) with a distinct cursor/idempotency namespace. The forum-side wiki-message proxy and `wiki_message_read_state` table are gone (migration 071).
- Catalog `site` and catalog source key are different identities. Keep dual-read compatibility where the Wave 161 branch records it; do not collapse them back into one constant.
- 方案③ (2026-08-21, letmoe + infra signed; kungal/moyu reuse): catalog is the existence layer. `/galgame` browse and site search do **not** send `claim_state`. Users do not claim games; the write that indexes a page is publishing a resource. `galgame.published` is the sticky SEO flag (first resource, not cleared on delete; migration 078). Hidden/ban still unpublishes. Do not reintroduce a user-facing 认领 flow or a local full replica.
- The forum reaches catalog **only over `/v2`** (reads since 2026-08-28; user writes with `Authorization: Bearer nmk_…`, problem+json, string ids). Catalog's `/api/v1` answers 410 on every path since 2026-08-27, and the claim-event cron failed silently every ten minutes until someone read a log line. Do not reach for `/v2/catalog/changes` as a claim channel: it carries only `{target_object, id, updated_at}` and accepts `from=` / `actor=` without applying them.
- A read that compiles and passes is not proof the face works: the first move of reads to `/v2` rewrote the test assertions along with the code and blanked the galgame detail page. Before changing a read face, check field by field that the include token and field it needs exist on `/v2` (infra's `collect/work.go` is the authority), and observe the page at runtime.

## Cross-Service Contracts (inviolable — owned by nextmoe-infra)

The active authoritative contract docs are synced as **read-only mirrors** under `docs/{oauth,image_service,artifact}/` (the file headers carry a GENERATED banner). Catalog is consumed from infra/portal without a vendored mirror; galgame-wiki is a retired portal/tombstone contract, not a live local mirror.
**To change a contract, change it at the infra source — do not touch the copies here**; the copies are regenerated by kungal-docs's `pnpm docs:sync`. Core invariants:

- **Identity (C1/C2)**: `user.id` is **the same integer** across this database, OAuth, and the other downstream — never renumber a user; local tables align with OAuth `users.id` via `*_user_id`. OAuth owns identity and issues JWTs; this service only **verifies signatures, never issues** (see `internal/middleware/auth.go`).
- **User profile (C6)**: **do not persist it to the local user table, do not treat it as the source of truth** (a short-TTL in-memory cache is fine — `pkg/userclient` already has a built-in ~10min TTL); fetch by id list via `GET /users/batch` (OAuth Client Basic Auth, ≤100 ids, **does not return** email / moemoepoint / created_at). Use `GET /users/search` for @-mention autocomplete (**do not cache**); use `/oauth/userinfo` for the current user. OAuth ships no SDK — implement a thin client yourself.
- **Moemoepoint (C3)**: a single balance per user, **single-sourced in OAuth**; the local `users.moemoepoint` is a cached view and must not be treated as the source of truth. Grant/deduct goes through the s2s API, with idempotency key = `<app>:<event>:<ref>` (e.g. `kungal:liked:topic_1207`). Reasons available to downstreams: `content_approved` / `content_removed` / `daily_checkin` / `liked`; **reserved by OAuth, forbidden over s2s**: `admin_grant` / `admin_deduct` / `migration` / `register_gift`. The pusher is in `internal/moemoepoint/pusher.go`. The s2s endpoints are **already implemented** (`POST/GET /users/:id/moemoepoint`, `Adjust` is idempotent; see infra `internal/platform/auth/handler/moemoepoint_handler.go` and `cmd/oauth/main.go`).
- **Images (C4)**: the content-addressed image host lives in OAuth (avatars / shared images **do not go through local S3**). URL = `{base}/{aa}/{bb}/{hash}[_variant].webp` (two-level hex sharding); pass `*_image_hash` fields and resolve them via the image client.
- **Catalog claims (C5 successor)**: catalog owns claim lifecycle events. New synchronization consumes `GET /v2/catalog/claim-events`; never add a new consumer of the retired wiki message feed. User-facing notification/read-state replacement must follow the Wave 161 coordination ledger rather than reviving a removed upstream route.

For active vendored contracts see `docs/oauth/`, `docs/image_service/`, and `docs/artifact/`; for catalog and the galgame retirement tombstone use the infra-owned source/portal.

## This Repo's Key Points

- Auth lives in `internal/middleware/auth.go`: the web uses a **BFF opaque session** (`kungal_session` cookie + the OAuth token stored in Redis) with **90-day sliding renewal** — the model and the 2026-06 fix are in `docs/proj/session-lifetime.md`. The first-party Flutter App instead sends `Authorization: Bearer <OP access token>`, verified locally against the OP's JWKS and gated by the `KUN_BEARER_CLIENT_IDS` allow-list (`middleware/bearer.go`). **A Bearer request never holds staff powers**: check capabilities with `user.Can` / `user.CanModerate`, never `perm.CanUser` / `role.Can*` on `user.Roles` (`bearer_guard_test.go` enforces this). The App contract (Bearer, `Idempotency-Key`, `/api/v1/app/version`) is in `docs/proj/app-direct-api.md`. **Neither a Bearer request nor a web session carries a content preference into v1 reads**: the result set is chosen by the explicit `include_nsfw` query param (`docs/proj/api-v1/01-standard.md` §3). `X-Kungal-Nsfw` was deleted 2026-09-23 and the per-request stance pipeline (`IsSFW`, `middleware/bearer_stance.go`) on 2026-09-24 (#248); `GET /api/v1/me/account` returns `content_stance: null` for a Bearer request.
- Modifying any file under `docs/{oauth,image_service,artifact}/` is **wrong** — those are infra's mirrors; to change them, change infra and then run `docs:sync`.
- **A database schema change must come with a migration reminder**: this repo's schema goes through `apps/api/migrations/NNN_*.up.sql` (idempotent, `IF NOT EXISTS`) plus a built-in migrate runner, and **does not run AutoMigrate at startup**. Whenever you add a migration file / change a table structure, **at the end of the task you must explicitly tell the user: whether a migration needs to be run in production, and which command to run**. Skipping it → live code reads a column that does not exist → silent failure (cf. the 2026-06 infra moemoepoint grant outage: a missing column made moemoepoint unobtainable site-wide for ~29h).
