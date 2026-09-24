#
# Tools image: EVERY apps/api/cmd/* binary in one image, for kungal's one-off
# migration / backfill jobs the api/migrate images don't carry — e.g. the
# deferred migrations (005/006/007/012/015), check-dup-email,
# backfill-provider-names (see nextmoe-docs docs/shared/user-migration + infra docs/deploy/03-bootstrap.md §B).
#
# The per-service Dockerfile builds ONE binary (ARG CMD); this bundles them all
# and invokes a job by name:
#
#   docker run --rm --network kun-galgame-infra_default \
#     --env-file docker/api.env ghcr.io/kunmoe/kungal-tools backfill-provider-names
#
# kungal has no cgo → pure static binaries. Build context MUST be the repo root.
ARG GO_VERSION=1.26

# ---- build ----
FROM golang:${GO_VERSION}-trixie AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
# -o <dir>/ writes one binary per cmd package, named after its directory.
RUN mkdir -p /out && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/ ./cmd/...

# ---- run (debian-slim; binaries on PATH, invoked by name) ----
FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --uid 10001 --create-home --shell /usr/sbin/nologin appuser
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
# migrate reads SQL from ./migrations relative to WORKDIR (/app) at runtime;
# without it the deferred migrations silently apply nothing.
COPY apps/api/migrations /app/migrations
# backfill-friend-link-banners reads the legacy static banners from disk (the
# in-cluster HTTP fetch of these is unreliable), then re-uploads via image_service.
COPY apps/web/public/friends /app/friends
# backfill-cover-hashes reads doc article banners (root-relative /content/*.avif
# static assets) from disk for the same reason — run it with `-webroot /app` so
# `/content/x/banner.avif` resolves to /app/content/x/banner.avif.
COPY apps/web/public/content /app/content
USER appuser
# No ENTRYPOINT: run a job by name, e.g. `docker run ... kungal-tools migrate --only=005,006,007,012,015`.
