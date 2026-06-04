# ── Stage 1: Builder ──────────────────────────────────────────────────────────
# Compiles the Go binary once. Output is shared by both runtime targets below.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependency layer separately — only invalidated when go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG BUILD_TAGS="production"
# -s -w: strip debug symbols and DWARF info (~20% smaller binary)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -tags "${BUILD_TAGS}" \
    -o /identity-service \
    ./cmd/server

# ── Stage 2: test ─────────────────────────────────────────────────────────────
# Alpine-based runtime for CI smoke testing.
# Keeps wget/sh for docker-compose healthcheck (CMD-SHELL).
# NOT pushed to GHCR — used only as the smoke-test target in CI.
FROM alpine:3.19 AS test

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /identity-service .

EXPOSE 8080
EXPOSE 50051

ENTRYPOINT ["./identity-service"]

# ── Stage 3: production ───────────────────────────────────────────────────────
# Distroless runtime — no shell, no package manager, minimal attack surface.
# ca-certificates and tzdata are included in the base image.
# This is the image pushed to GHCR.
FROM gcr.io/distroless/static-debian12 AS production

COPY --from=builder /identity-service /identity-service

EXPOSE 8080
EXPOSE 50051

ENTRYPOINT ["/identity-service"]

