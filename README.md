# ve-xchange-api

REST API for Venezuelan exchange rates. Fetches BCV (Banco Central de Venezuela) rates and Binance P2P prices and exposes them over HTTP.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check — `200 OK` or `503` if any rate is stale |
| `GET` | `/rates` | All current rates stored in PostgreSQL |
| `GET` | `/rates/{currency}` | Single current rate stored in PostgreSQL |
| `GET` | `/rates/{currency}/history` | Historical rates (`?fromDate=YYYY-MM-DD&toDate=YYYY-MM-DD`) |
| `GET` | `/docs/` | Interactive Swagger UI |
| `GET` | `/openapi.yaml` | Raw OpenAPI 3.0 spec |

**Currency values:** `usd_bcv`, `eur_bcv`, `usdt`, `usdt_venta`, `usdt_compra`

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
# or manually:
go tool oapi-codegen --config api/oapi-codegen.yaml api/openapi.yaml
```

## Data sources

| Currency | Source | Refresh |
|----------|--------|---------|
| `usd_bcv` | [Banco Central de Venezuela](https://bcv.org.ve) | on startup; weekdays from 17:00 UTC-4, retrying every 30 min after failure until 19:00 |
| `eur_bcv` | [Banco Central de Venezuela](https://bcv.org.ve) | on startup; weekdays from 17:00 UTC-4, retrying every 30 min after failure until 19:00 |
| `usdt` | Binance P2P API | every 5 minutes |

The rate endpoints only query PostgreSQL. Provider HTTP requests occur in
background workers and never during a client API request.

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
  provider/                 — BCV HTML scraper & Binance HTTP clients
  rates/                    — PriceResponse type
  state/                    — provider health state
  worker/                   — periodic and business-window fetch loops
api/openapi.yaml            — OpenAPI 3.0 spec (source of truth)
```
