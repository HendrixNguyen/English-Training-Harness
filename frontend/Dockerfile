# syntax=docker/dockerfile:1

# Build stage. The PWA is client-rendered (nuxt.config.ts: ssr: false), so
# `nuxi generate` emits a static site. The three NUXT_PUBLIC_* values are baked
# into that site at build time (Nuxt runtimeConfig.public); pass them as
# --build-arg (compose.yml does) — there is no runtime injection.
FROM node:20-alpine AS build
WORKDIR /src
# The whole tree first: package.json's postinstall runs `nuxi prepare`, which
# needs nuxt.config.ts and the app present.
COPY . .
RUN npm ci
ARG NUXT_PUBLIC_API_BASE=http://localhost:8080
ARG NUXT_PUBLIC_GOOGLE_CLIENT_ID=
ARG NUXT_PUBLIC_VAPID_PUBLIC_KEY=
ENV NUXT_PUBLIC_API_BASE=$NUXT_PUBLIC_API_BASE \
    NUXT_PUBLIC_GOOGLE_CLIENT_ID=$NUXT_PUBLIC_GOOGLE_CLIENT_ID \
    NUXT_PUBLIC_VAPID_PUBLIC_KEY=$NUXT_PUBLIC_VAPID_PUBLIC_KEY
RUN npx nuxi generate

# Runtime stage: Caddy serving the static output (Caddyfile has the SPA
# fallback and the cache rules the service worker depends on).
FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/.output/public /srv
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:80/healthz || exit 1
