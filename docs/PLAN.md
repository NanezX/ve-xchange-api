# PLAN — ve-xchange-api

Technical roadmap for what comes next: pending tech debt and planned features.

---

## Tech Debt

### TD-0 — BCV Daily Scheduler ✅ (done)

**Problem:** BCV publishes the next day's rate at 5-6 PM. A 6-hour polling interval would apply tomorrow's rate during the current day, serving incorrect data.

**Solution:** Replaced the `Every: 6h` ticker with a `DailyAt` field on `ProviderJob`. The BCV job fires once daily at 00:05 AM UTC-4, ensuring the newly published rate is now the valid rate for the current day.

**Implementation:** `internal/worker/worker.go` — `TimeOfDay` struct, `nextDaily()` pure function, daily `time.NewTimer` loop. Binance job is unchanged (`Every: 5min`).

---

### TD-1 — Warm Cache on Startup ✅ (done)

**Problem:** On startup `AppState` is zero-valued. Any request to `/rates` before the first worker tick returns empty/zero data. For Binance the wait is up to 5 minutes; for BCV up to 6 hours.

**Solution:** After the DB connection is established, query the latest value per currency and pre-load `AppState` before starting the HTTP server or the worker.

```
SELECT DISTINCT ON (currency) currency, value, recorded_at
FROM prices_history
ORDER BY currency, recorded_at DESC;
```

**Impact:** High. Zero-downtime restarts and container restarts will serve correct data immediately.

---

### TD-2 — Consolidation Worker (`is_average` in history) ✅ (done)

**Problem:** `GET /rates/{currency}/history` returns raw observations. For USDT/Binance this means ~288 rows/day (one every 5 min). The API contract defines `is_average: true` for daily aggregates but this field was never populated.

**Solution:** A nightly consolidation worker that:
1. Groups raw `usdt` observations by day and computes the average.
2. Inserts one summary row with `is_average=true` (recorded_at = midnight UTC-4 of that day).
3. Deletes the raw rows for that day to bound table growth.

**Implementation:**
- Migration `00002_add_is_average.sql`: `ALTER TABLE prices_history ADD COLUMN is_average BOOLEAN NOT NULL DEFAULT FALSE`
- `db.ConsolidateDay(ctx, currency, from, to)`: atomic transaction (avg → delete raw → delete old avg → insert new avg)
- `worker.TaskJob` + `worker.StartTaskWorker`: daily scheduled tasks without provider coupling
- `main.go`: consolidation `TaskJob` at 01:00 AM UTC-4 for `usdt`
- `api.HistoryEntry.IsAverage bool`: exposed in OpenAPI spec and JSON responses

---

## Planned Features

### F-1 — Observability with Prometheus

Expose a `GET /metrics` endpoint (Prometheus text format). A Prometheus instance scrapes it on a schedule. Grafana connects to Prometheus for dashboards and alerting.

**Metrics to expose:**
- `provider_fetch_total{provider, status}` — counter of successful/failed fetches
- `provider_consecutive_failures{provider}` — current failure streak gauge
- `rate_value{currency}` — current exchange rate as gauge (useful for graphing price history)
- `http_request_duration_seconds{method, path, status}` — histogram

**Library:** `github.com/prometheus/client_golang`

**Setup in prod:** A dedicated `docker-compose.monitoring.yml` (Prometheus + Grafana) that connects to the main app network via an `external` Docker network. Kept separate so monitoring can be started/stopped independently without touching the app stack.

---

### F-2 — Alerts

Two separate channels:

**Email (provider failures):**
- Trigger: `provider_consecutive_failures > 3` OR `is_stale = true` on `/health`
- Library: `net/smtp` (stdlib) or `github.com/wneessen/go-mail` for a nicer API
- Config: `ALERT_EMAIL_TO`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`

**Telegram Webhook (operational events):**
- Trigger: same as above, plus optional recovery notification
- Implementation: `POST https://api.telegram.org/bot<TOKEN>/sendMessage`
- Config: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`
- No external library needed — a single `http.Post` call suffices

Both channels can be optional (disabled if env vars are not set).

---

### F-3 — Rate Limiting

`golang.org/x/time/rate` token-bucket limiter per IP. Only relevant if the API is ever exposed publicly. **Skip for now** — the port is not exposed in production.

---

### F-4 — `Cache-Control` Headers

Add `Cache-Control: public, max-age=30` to `/rates` and `/rates/{currency}` responses. Allows the consuming SvelteKit app (or any HTTP proxy in between) to skip duplicate requests within the refresh window. One-liner middleware.

---

## Infrastructure

### I-1 — DB Schema Migrations ✅ (done with goose)

Replaced the ad-hoc `CreateSchema` with versioned SQL migrations managed by `pressly/goose`. Runs automatically at container start.

### I-2 — Docker & Docker Compose ✅ (done)

See `Dockerfile`, `docker-compose.yml` (local dev), and `docker-compose.prod.yml`.

### I-3 — Shared PostgreSQL in Production

In production, `ve-xchange-api` shares the same PostgreSQL container already used by the optikt SvelteKit app. They use **separate databases** on the same server:

| App | Database |
|-----|----------|
| optikt | `optikt_db` |
| ve-xchange-api | `vexchange_db` |

See the shared compose snippet in `docker-compose.prod.yml` — the `postgres` service creates both databases via an init script, and each app gets its own `DATABASE_URL`.
