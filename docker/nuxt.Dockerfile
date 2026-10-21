#
# Build for kungal's Nuxt 4 frontend (apps/web, Nitro node-server preset).
#
# Build context is the repo root (pnpm workspace: apps/web + apps/api). KunUI is
# now consumed as the published @kungal/ui-nuxt npm layer (a normal dependency
# resolved from node_modules), so there is no local packages/ui to copy in.
#
# Public runtime config (apiBase, oauth client, image/wiki URLs) is read by
# nuxt.config.ts from process.env at BUILD time, so it is passed as build args
# and baked. Any public key can still be overridden at RUNTIME via the
# canonical NUXT_PUBLIC_* env names (build once, configure per env — see
# docker/README.md).
ARG NODE_VERSION=24.22.1

FROM node:${NODE_VERSION}-trixie-slim AS base
RUN corepack enable
WORKDIR /repo

# ---- deps: copy every workspace manifest, install the whole workspace ----
FROM base AS deps
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/web/package.json    apps/web/package.json
COPY apps/api/package.json    apps/api/package.json
# KunUI is a published dependency (@kungal/ui-*), pulled from npm by this
# install — there's no local packages/ui to copy. apps/api is Go-only (no JS
# deps), so a plain whole-workspace install just resolves apps/web's deps.
# --ignore-scripts: web's `postinstall: nuxt prepare` can't run here (app
# source isn't copied yet); the build stage's `nuxt build` runs prepare itself.
RUN pnpm install --frozen-lockfile --ignore-scripts

# ---- build ----
FROM deps AS build
# Frontend public config, baked at build. Empty args fall back to the
# in-config defaults (`process.env.X || '<default>'` in nuxt.config.ts).
ARG API_BASE_URL=
ARG OAUTH_SERVER_URL=
ARG OAUTH_FRONTEND_URL=
ARG OAUTH_CLIENT_ID=
ARG OAUTH_REDIRECT_URI=
ARG KUN_GALGAME_URL=
ENV API_BASE_URL=${API_BASE_URL} \
    OAUTH_SERVER_URL=${OAUTH_SERVER_URL} \
    OAUTH_FRONTEND_URL=${OAUTH_FRONTEND_URL} \
    OAUTH_CLIENT_ID=${OAUTH_CLIENT_ID} \
    OAUTH_REDIRECT_URI=${OAUTH_REDIRECT_URI} \
    KUN_GALGAME_URL=${KUN_GALGAME_URL}
COPY apps/web apps/web
# build:limit bumps Node's heap (--max-old-space-size=8192). The web build is
# memory-heavy and OOM-aborts (exit 134 / SIGABRT) under the default heap in
# CI's constrained build env; the GitHub runner has 16 GB so 8 GB heap fits.
# `nuxt build` resolves the extended @kungal/ui-nuxt layer from node_modules and
# generates apps/web/.nuxt itself — no separate layer prepare needed.
RUN pnpm --filter web run build:limit

# ---- run: just Node + the self-contained .output (no pnpm, no sources) ----
FROM node:${NODE_VERSION}-trixie-slim AS run
ENV NODE_ENV=production HOST=0.0.0.0 NITRO_PORT=7777
WORKDIR /app
COPY --from=build /repo/apps/web/.output ./.output
USER node
EXPOSE 7777
CMD ["node", ".output/server/index.mjs"]
