# backend/Dockerfile
# ============================================================
# Production multi-stage build for the PUSINGBERAT Go backend.
#
# Stage 1 (builder) — compiles the binary inside the official
#   Go toolchain image so the host machine needs no Go install.
# Stage 2 (runtime) — copies only the compiled binary into a
#   minimal Alpine image, keeping the final image small and
#   reducing the attack surface.
#
# This file is used by the Jenkins CI pipeline and for
# production deployments. For local development, use
# Dockerfile.dev (Air hot-reload) instead.
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.25.0-alpine AS builder

# git is required by go mod download for modules fetched via VCS.
# ca-certificates enables outbound HTTPS (Discord webhook calls).
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependency layer — only re-downloaded when go.mod / go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source and build a statically-linked binary.
# CGO_ENABLED=0  → no glibc dependency; binary runs in Alpine/scratch.
# -trimpath      → strip local file paths from the binary.
# -ldflags       → strip debug symbols to reduce binary size.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /app/server \
      ./cmd/server

# ── Stage 2: Runtime ────────────────────────────────────────
FROM alpine:3.20

# ca-certificates — needed for Discord webhook HTTPS calls.
# tzdata         — allows TZ env var to work correctly.
RUN apk add --no-cache ca-certificates tzdata

# Run as a non-root user for better container security.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

# Copy only the compiled binary from the builder stage.
COPY --from=builder /app/server /usr/local/bin/server

# Copy the YAML detection rules that are read at startup.
# These can also be mounted as a volume in production to allow
# rule updates without rebuilding the image.
COPY --from=builder /build/rules /app/rules

EXPOSE 8080

# RULES_DIR tells the Go binary where to find the .yaml rule files.
ENV RULES_DIR=/app/rules

CMD ["server"]
