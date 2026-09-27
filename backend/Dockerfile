# syntax=docker/dockerfile:1

# Build stage. The binary is static (CGO_ENABLED=0): migrations are embedded
# (internal/store/migrations.go) and tzdata is embedded (cmd/api/main.go asserts
# it at boot), so nothing else from this stage is needed at runtime.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime stage. alpine rather than distroless so the HEALTHCHECK has an HTTP
# client (busybox wget); ca-certificates is for Google, Gemini, OpenAI, DeepSeek
# and the TLS Postgres/Redis URLs the runbook documents.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
 && adduser -D -H -u 10001 api
COPY --from=build /out/api /usr/local/bin/api
USER api
# config.Load reads PORT; Railway overrides it, compose and Dokploy keep 8080.
ENV PORT=8080
EXPOSE 8080
# GET /healthz is 200 only when Postgres and Redis answer (internal/health), so
# an unhealthy container means a dependency is down, not just the process.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${PORT}/healthz" || exit 1
ENTRYPOINT ["/usr/local/bin/api"]
