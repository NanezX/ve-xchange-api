# syntax=docker/dockerfile:1.7

# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o ve-xchange-api ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.21 AS runner

# ca-certificates: needed for HTTPS calls to Binance & DolarAPI
# tzdata: correct timestamps in logs and DB
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder --chown=nobody:nobody /app/ve-xchange-api .
# openapi.yaml is served at GET /openapi.yaml (relative path from CWD)
COPY --from=builder --chown=nobody:nobody /app/api/openapi.yaml ./api/openapi.yaml

USER nobody

EXPOSE 8080

CMD ["./ve-xchange-api"]
