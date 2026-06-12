# ve-xchange-api

REST API for Venezuelan exchange rates. Fetches BCV (Banco Central de Venezuela) rates and Binance P2P prices and exposes them over HTTP.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check — `200 OK` or `503` if any rate is stale |
| `GET` | `/rates` | All current rates (`usd_bcv`, `eur_bcv`, `usdt`) |
| `GET` | `/rates/{currency}` | Single currency rate |
| `GET` | `/rates/{currency}/history` | Historical rates (`?fromDate=YYYY-MM-DD&toDate=YYYY-MM-DD`) |
| `GET` | `/docs/` | Interactive Swagger UI |
| `GET` | `/openapi.yaml` | Raw OpenAPI 3.0 spec |

**Currency values:** `usd_bcv`, `eur_bcv`, `usdt`

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | **yes** | — | PostgreSQL connection string (`postgresql://user:pass@host:5432/db`) |
| `APP_PORT` | no | `8080` | HTTP port |

## Running locally

### With Docker Compose (recommended)

```bash
cp .env.example .env
# edit .env if needed
docker compose up
```

The API will be available at `http://localhost:8080`. Swagger UI at `http://localhost:8080/docs/`.

### Without Docker

Requires a running PostgreSQL instance.

```bash
cp .env.example .env
# set DATABASE_URL in .env
go run ./cmd/server
```

## Development

```bash
# run tests
go test ./...

# run tests with race detector
go test -race ./...

# run linter
golangci-lint run

# regenerate OpenAPI types (after editing api/openapi.yaml)
go generate ./internal/api/...
```

## Data sources

| Currency | Source | Refresh |
|----------|--------|---------|
| `usd_bcv` | [DolarAPI](https://ve.dolarapi.com) | every 6 hours |
| `eur_bcv` | [DolarAPI](https://ve.dolarapi.com) | every 6 hours |
| `usdt` | Binance P2P API | every 5 minutes |

## Docker image

Built and published to Docker Hub via the [Docker Release](.github/workflows/docker-build.yml) workflow (manual trigger from `main`).

```bash
docker pull nanezx/ve-xchange-api:latest
```

## Architecture

```
cmd/server/main.go          — entrypoint, wires everything together
internal/
  api/                      — oapi-codegen generated types & router
  config/                   — env var loading
  db/                       — PostgreSQL store + goose migrations
  handler/                  — HTTP handlers (implements ServerInterface)
  provider/                 — DolarAPI & Binance HTTP clients
  rates/                    — PriceResponse type
  state/                    — in-memory cache (AppState)
  worker/                   — periodic price fetch loop
api/openapi.yaml            — OpenAPI 3.0 spec (source of truth)
```
