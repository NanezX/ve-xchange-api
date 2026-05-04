# ROADMAP — ve-xchange-api

> Technical roadmap for future improvements to the project.
> Each improvement documents the **current state**, the **proposed improvement**, and the **technical rationale**.
>
> **This document contains no code** — it only defines the WHAT and the WHY behind each decision.

---

## Table of Contents

1. [Testing & Code Quality](#pillar-1-testing--code-quality)
2. [CI/CD with GitHub Actions](#pillar-2-cicd-with-github-actions)
3. [Architecture & Refactoring](#pillar-3-architecture--refactoring)
4. [Persistence & Resilience](#pillar-4-persistence--resilience)
5. [Advanced Error Handling](#pillar-5-advanced-error-handling)
6. [Extend Test Coverage](#pillar-6-extend-test-coverage)
7. [OpenAPI & API Documentation](#pillar-7-openapi--api-documentation)
8. [Suggested Implementation Order](#suggested-implementation-order)

---

## Pillar 1: Testing & Code Quality [DONE]

### 1.1 — Unit Testing with HTTP Client Mocking [DONE]

| | Details |
|---|---|
| **Improvement** | Create unit tests for `BinanceProvider` and `DolarApiProvider` by injecting a mock HTTP client. |
| **Current state** | No tests exist in the project. Both providers (`binanceProvider.go`, `dolarApiProvider.go`) create an `http.Client{Timeout: 10 * time.Second}` internally within each method (`fetchPrices`, `GetPrices`), making it **impossible** to substitute the client with a mock in tests. |
| **Rationale** | Testing providers without calling the real APIs is critical because: **(a)** external APIs are slow and introduce flakiness into tests, **(b)** their responses change constantly, making it impossible to verify expected values. Without mocks, tests would be fragile, slow, and non-deterministic. The solution is to define an `HTTPDoer` interface (with method `Do(*http.Request) (*http.Response, error)`) and inject it into providers via constructor. In production, the real `http.Client` is passed; in tests, a mock returning predefined responses is used. |

### 1.2 — Leverage the Existing `PriceProvider` Interface to Test the Worker [DONE]

| | Details |
|---|---|
| **Improvement** | Refactor `updateBcv` and `updateBinance` in `worker.go` to accept `PriceProvider` (the interface) instead of concrete types. |
| **Current state** | The `PriceProvider` interface already exists in `types.go` with methods `GetPrices()` and `GetName()`, but the worker uses concrete types directly (`*BinanceProvider`, `*DolarApiProvider`). Functions `updateBcv(p *DolarApiProvider)` and `updateBinance(p *BinanceProvider)` accept concrete pointers. |
| **Rationale** | If the update functions accepted `PriceProvider`, they could be tested with mock structs that implement the interface, without depending on real APIs. This allows verifying that the worker updates `AppState` correctly (prices, timestamps) and handles errors without crashing — all without network access. The interface already exists; it just needs to be used. |

### 1.3 — Test the `/rates` Handler [DONE]

| | Details |
|---|---|
| **Improvement** | Write tests for `RatesHandler` using `httptest.NewRecorder()` with an injected state. |
| **Current state** | The `RatesHandler` in `handlers.go` accesses the global `AppState` directly, preventing isolated testing. |
| **Rationale** | Injecting the state as a dependency (struct field or constructor parameter) allows: **(a)** pre-loading a state with known values, **(b)** verifying the handler returns valid JSON with `Content-Type: application/json`, **(c)** using `httptest.NewRecorder` to simulate requests without starting a real server. This type of test is standard in Go and executes in milliseconds. |

---

## Pillar 2: CI/CD with GitHub Actions [DONE]

### 2.1 — CI Workflow: `go test` + `golangci-lint` on Every Push/PR [DONE]

| | Details |
|---|---|
| **Improvement** | Create a GitHub Actions workflow that automatically runs tests and linting on every Push and Pull Request to `main`. |
| **Current state** | No CI/CD pipeline exists. Code enters `main` without any automated validation. |
| **Rationale** | Automating `go test ./...` and `golangci-lint run` on every Push and PR ensures that: **(a)** no change breaks existing tests (regression), **(b)** style errors, potential bugs, and code smells are detected before merge, **(c)** a minimum quality standard is established that scales with the team. Without CI, the only quality gate is individual discipline — which inevitably fails. |

### 2.2 — `.github/workflows/ci.yml` File [DONE]

| | Details |
|---|---|
| **Improvement** | Create the workflow configuration file with concrete CI steps. |
| **Current state** | The `.github/workflows/` directory does not exist. |
| **Rationale** | The workflow should include: **(1)** `actions/setup-go` with the Go version from `go.mod`, **(2)** `go vet ./...` for basic static analysis, **(3)** `golangci/golangci-lint-action` for advanced linting, **(4)** `go test -race -coverprofile=coverage.out ./...` for tests with data race detection. The `-race` flag is especially important because the project uses goroutines and mutex — it will detect data races that normal tests miss. Optionally, upload the coverage report as an artifact. |

### 2.3 — Branch Protection on `main` [DONE]

| | Details |
|---|---|
| **Improvement** | Configure a branch protection rule on GitHub requiring green CI to merge into `main`. |
| **Current state** | No branch protection configured. |
| **Rationale** | Without branch protection, CI exists but can be ignored. Configuring "Require status checks to pass before merging" ensures the pipeline is a mandatory gate, not an optional one. This prevents accidental merges of broken code and establishes a quality culture from the start of the project. |

---

## Pillar 3: Architecture & Refactoring [DONE]

### 3.1 — Single, Reusable `HttpClient` (Injected) [DONE]

| | Details |
|---|---|
| **Improvement** | Create a single `http.Client` in `main()` with centralized configuration and pass it to all providers via constructor. |
| **Current state** | Both `BinanceProvider.fetchPrices()` and `DolarApiProvider.GetPrices()` create a **new** `http.Client{Timeout: 10 * time.Second}` on **every call**. |
| **Rationale** | Creating a new `http.Client` per request wastes Go's `http.Transport` **connection pooling**. The runtime reuses TCP connections (keep-alive) **only** if the same client/transport is shared. A client injected once: **(a)** reuses TCP connections, reducing latency and file descriptor consumption, **(b)** allows centralized configuration of timeouts, transport, and proxy, **(c)** enables testing with a mock (ties into improvement 1.1). Creating clients per request is a recognized anti-pattern in Go: the `net/http` documentation itself states *"Clients should be reused instead of created as needed."* |

### 3.2 — Generic `FetchJSON[T]` Function [DONE]

| | Details |
|---|---|
| **Improvement** | Implement a generic function `FetchJSON[T any](client HTTPDoer, req *http.Request) (T, error)` that encapsulates the repeated pattern of HTTP request → verify status → decode JSON. |
| **Current state** | Both providers repeat the same pattern: create request → execute with client → check `StatusCode` → read error body on failure → create decoder → decode JSON → handle error. That's approximately 25 duplicated lines in each provider. |
| **Rationale** | Duplication violates DRY and multiplies the places where a bug can hide. A generic function: **(a)** reduces the duplication to a single testable implementation, **(b)** centralizes HTTP error handling (status codes, error body), **(c)** any future improvement (retry, logging, metrics, tracing) is automatically applied to all providers without touching their code. Go 1.18+ supports generics, enabling strong typing without sacrificing reusability. |

### 3.3 — Migration to Standard Folder Structure (`/internal`, `/cmd`) [DONE]

| | Details |
|---|---|
| **Improvement** | Reorganize the code into: `cmd/server/main.go`, `internal/provider/`, `internal/state/`, `internal/handler/`, `internal/config/`, `internal/worker/`. |
| **Current state** | All files are in the `main` package at the project root. There is no folder separation or package-level separation of concerns. |
| **Rationale** | A flat `main` package creates the following problems as it grows: **(a)** types and functions cannot be imported from other projects or tools, **(b)** there is no encapsulation — everything is accessible from everywhere, **(c)** name collisions become frequent. The `internal/` structure is the de facto standard in Go: the compiler **prevents** external packages from importing code inside `internal/`, protecting the module's public API. `cmd/server/` separates the entrypoint from business logic, allowing future entrypoints (CLI, separate workers, etc.) to be added. |

### 3.4 — Remove Global Variables (`AppState`, `AppConfig`) [DONE]

| | Details |
|---|---|
| **Improvement** | Create instances of `State` and `Config` in `main()` and inject them as dependencies into handlers, worker, and providers (constructor injection). |
| **Current state** | `AppState` (`state.go`) and `AppConfig` (`config.go`) are package-level global variables, accessed directly from `worker.go`, `handlers.go`, and `dolarApiProvider.go`. |
| **Rationale** | Global variables create implicit coupling and hinder testing: **(a)** tests that modify `AppState` affect other tests — impossible to run tests in parallel (`t.Parallel()`), **(b)** there is no way to construct a provider with a different config in tests without mutating the global, **(c)** implicit imports (accessing a global from another file) make data flow invisible. Constructor injection makes dependencies **explicit**: if a function needs `Config`, it receives it as a parameter, and it is obvious who depends on what. |

---

## Pillar 4: Persistence & Resilience

### 4.1 — State as Cache + DB for Historical Data

| | Details |
|---|---|
| **Improvement** | Add a database (SQLite for development, PostgreSQL for production) to persist historical rates. `AppState` remains as a fast in-memory read cache. |
| **Current state** | `AppState` only stores the **latest value** of each rate (`UsdBCV`, `EurBCV`, `UsdtBinance`). If the process restarts, rates are zero-valued until the tickers fetch new data. No historical data exists. |
| **Rationale** | **(a)** On startup, the latest rate can be loaded from the DB for immediate data availability (warm cache) instead of waiting for the first tick. **(b)** Historical data enables endpoints like "average rate over the last month", "daily variation", "trend chart" — high-value data for API consumers. **(c)** In-memory `AppState` continues serving `/rates` with ~0 latency (mutex read-lock only); the DB is written to asynchronously. SQLite (via `modernc.org/sqlite`, pure Go without CGO) is the simplest option (single file, no extra infrastructure); PostgreSQL for scenarios with multiple instances or high write volume. |

### 4.2 — Graceful Shutdown

| | Details |
|---|---|
| **Improvement** | Implement clean shutdown of the HTTP server, the worker, and future DB connections using `signal.NotifyContext`. |
| **Current state** | `main.go` uses `http.ListenAndServe` which blocks indefinitely with no controlled shutdown option. The worker is launched with `go StartPriceWorker()` without any cancellation mechanism. The `defer ticker.Stop()` calls in the worker **never execute** because the `for-select` is an infinite loop with no exit. |
| **Rationale** | Without graceful shutdown: **(a)** a `SIGTERM` (deploy, restart, container scaling) cuts in-flight HTTP requests, leaving clients without a response, **(b)** tickers and goroutines remain as leaks until the OS kills the process, **(c)** with a future DB, a write cut in half could corrupt data. The standard Go solution: create a `context.Context` with `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)`, pass it to the worker (so `case <-ctx.Done()` breaks the loop) and to `http.Server` (using `server.Shutdown(ctx)` which waits for in-flight requests). This is a requirement for any container-based deployment (Docker, Kubernetes). |

### 4.3 — Retry with Exponential Backoff

| | Details |
|---|---|
| **Improvement** | Add retry logic with exponential backoff for external API HTTP calls. |
| **Current state** | If an API fails, the error is printed with `fmt.Printf` and ignored. The next update opportunity depends on the ticker: 5 minutes for Binance, **6 hours** for BCV. |
| **Rationale** | External APIs fail transiently (timeouts, 429/503 errors, momentary network issues). Retry with exponential backoff (e.g., 1s → 2s → 4s, max 3 attempts) handles these cases without waiting for the next ticker cycle. For BCV, where the tick interval is **6 hours**, losing an update due to a 1-second network timeout is a disproportionate cost. Exponential backoff prevents hammering the API during sustained failures. It can be implemented inside `FetchJSON` (improvement 3.2) or as `http.Client` middleware. |

---

## Pillar 5: Advanced Error Handling

### 5.1 — Stale Data Detection

| | Details |
|---|---|
| **Improvement** | Implement logic to detect and communicate when served rates are outdated. |
| **Current state** | `AppState.Rates.LastUpdate` is updated on each successful write, but **nobody checks its age**. If the Binance API fails for 2 hours, the `/rates` endpoint continues serving the 2-hour-old price **as if it were current**, with no warning to the consumer. |
| **Rationale** | The API consumer needs to know whether the data is reliable. Serving stale data without a warning is **worse** than returning an explicit error — the consumer makes financial decisions based on data they believe is current but isn't. Implementation: **(a)** define staleness thresholds per provider (e.g., Binance > 15 min = stale, BCV > 12 hours = stale), **(b)** add fields to the response such as `"is_stale": false` and `"data_age_seconds": 42`, **(c)** optionally add a `/health` endpoint that returns `503 Service Unavailable` if any rate is stale — useful for load balancers and orchestrator health checks. |

### 5.2 — Structured Logging with `log/slog`

| | Details |
|---|---|
| **Improvement** | Migrate from `fmt.Printf`/`fmt.Println` to `log/slog` (standard since Go 1.21). |
| **Current state** | All logs use `fmt.Printf` and `fmt.Println`. There are no log levels, no structure, and no context (which provider failed, what price was updated, how long it took). |
| **Rationale** | `fmt.Printf` is not logging — it is stdout output without metadata. `log/slog` provides: **(a)** levels (Debug, Info, Warn, Error) for filtering by severity, **(b)** JSON format for production (parseable by any monitoring system), **(c)** typed contextual fields (`slog.String("provider", "binance"), slog.Float64("price", 42.5), slog.Duration("latency", elapsed)`). This is a requirement for production debugging — when something fails at 3am, you need logs that tell you which provider failed, with what error, how long ago, and what the last successful price was. |

### 5.3 — Consecutive Failure Counter per Provider

| | Details |
|---|---|
| **Improvement** | Maintain an atomic counter of consecutive failures per provider. |
| **Current state** | When a provider fails, a generic log is printed and execution waits for the next tick. There is no way to know whether a provider has had 1 or 100 consecutive failures. Each failure is treated in isolation, with no memory of the error history. |
| **Rationale** | An isolated failure is normal (intermittent network). Three consecutive failures suggest a real problem (API down, expired API key, format change). The counter enables: **(a)** automatically escalating log severity (Info → Warn → Error), **(b)** marking data as stale when a threshold is exceeded (direct link to improvement 5.1), **(c)** in the future, triggering alerts (webhook, email). Reset the counter on each success. Simple implementation with `atomic.Int64` per provider — no additional locks needed. |

---

## Suggested Implementation Order

```
Phase 1 — Foundation (Architecture)           [DONE]
  3.1  Single, injected HttpClient
  3.2  Generic FetchJSON[T] function
  3.4  Remove global variables (dependency injection)
  3.3  Migration to standard folder structure

Phase 2 — Testing                             [DONE]
  1.1  Unit Testing with HTTP mocking
  1.2  Worker tests with PriceProvider mock
  1.3  /rates handler tests

Phase 3 — CI/CD                               [DONE]
  2.1  GitHub Actions workflow (go test + lint)
  2.2  .github/workflows/ci.yml file
  2.3  Branch protection on main

Phase 4 — API Contract & Documentation
  7.1  Spec-first design with oapi-codegen (redefines response schemas)
  7.2  Swagger UI with swaggest/swgui (always available)
  7.3  New endpoints: /health, /rates/{currency}

Phase 5 — Resilience & Observability
  4.2  Graceful Shutdown
  4.3  Retry with exponential backoff
  5.1  Stale data detection (integrates with /health from 7.3)
  5.2  Structured logging with log/slog
  5.3  Consecutive failure counter

Phase 6 — Persistence
  4.1  State as cache + DB for historical data
  7.4  /rates/{currency}/history endpoint (depends on 4.1)
```

> **Why this order?**
>
> - **Phase 1 first** because the architecture refactors (dependency injection, removing globals) are a **prerequisite** for writing quality unit tests.
> - **Phase 2 before Phase 3** because you need tests for CI to have something to run.
> - **Phase 3 before Phase 4** because CI will catch regressions when you implement resilience changes.
> - **Phase 4 (API Contract) before Phase 5** because the OpenAPI spec redefines the response schemas (per-currency metadata: `value`, `last_updated`, `is_stale`) that Pillar 5 depends on. The `/health` endpoint from 7.3 is also the integration point for stale data detection (5.1).
> - **Phase 6 last** because it requires the most additional infrastructure (PostgreSQL) and the `/history` endpoint (7.4) has a hard dependency on the DB being in place.

---

## Pillar 6: Extend Test Coverage

### 6.1 — Unit Tests for `State`

| | Details |
|---|---|
| **Improvement** | Write unit tests for `UpdateRates`, `GetRates`, `UpdateBcvPrice`, and `UpdateBinancePrice`. |
| **Current state** | `State` has no tests. Its correctness is only verified indirectly through handler tests. |
| **Rationale** | **(a)** `UpdateBcvPrice` and `UpdateBinancePrice` access fields by key from a `PriceResponse` map with no safety checks (marked with `FIXME`). A test that passes a malformed map would expose the nil/zero behavior. **(b)** Thread safety should be verified with `-race` flag — concurrent reads and writes to `State` must not cause data races. **(c)** These tests are cheap to write and provide a safety net before adding the DB layer (Pillar 4). |

### 6.2 — Unit Tests for `Config`

| | Details |
|---|---|
| **Improvement** | Write unit tests for `LoadConfig` covering environment variable loading and missing/invalid values. |
| **Current state** | `LoadConfig` has no tests. If an env var is missing or has an unexpected format, the failure only surfaces at runtime when the server starts. |
| **Rationale** | Config loading is a boundary — it reads from the environment, which is external input. Testing it ensures: **(a)** valid env vars are parsed correctly, **(b)** missing required vars  fail fast with a clear error, **(c)** invalid values (e.g., a non-numeric port) are caught. Tests can use `os.Setenv`/`os.Unsetenv` to control the environment. |

### 6.3 — Negative/Boundary Values in Providers

| | Details |
|---|---|
| **Improvement** | Add tests for negative prices, `NaN`, `Infinity`, and extreme outliers in provider responses. |
| **Current state** | Provider tests cover zero values (`<= 0`) but not floating-point edge cases like `math.NaN()`, `math.Inf(1)`, or suspiciously large values (e.g., `999999999.0`). |
| **Rationale** | JSON allows valid floats that are semantically broken for financial data. `json.Unmarshal` will happily decode a response with `"usd": 1e308` into a `float64`. These edge cases should be caught at the provider boundary before reaching `AppState` or the DB. |

---

## Pillar 7: OpenAPI & API Documentation

### 7.1 — Spec-First Design with `oapi-codegen`

| | Details |
|---|---|
| **Improvement** | Write `api/openapi.yaml` as the single source of truth and use `oapi-codegen` to generate Go types and a `ServerInterface` that handlers must implement. |
| **Current state** | The API has no formal contract. `ExchangeRates` is a flat struct with a single `LastUpdate` for all currencies. There is no machine-readable spec, no path-param routes, and no `/health` endpoint. |
| **Rationale** | Spec-first development means the contract is defined before implementation, not inferred from it. `oapi-codegen` with the `std-http` generator produces: **(a)** typed Go structs from `#/components/schemas` (replacing the hand-written `ExchangeRates`), **(b)** a `ServerInterface` with one method per endpoint — the compiler enforces that every endpoint is implemented, **(c)** a `HandlerFromMux(si ServerInterface, mux *http.ServeMux)` function that wires routes automatically. The new response schema will be per-currency objects (`{ value, last_updated, data_age_seconds, is_stale }`) instead of a flat struct, which is a prerequisite for Pillar 5.1 (stale data detection). No external router is introduced — `net/http` ServeMux (Go 1.22+) supports path parameters natively. |

### 7.2 — Interactive API Documentation with `swaggest/swgui`

| | Details |
|---|---|
| **Improvement** | Embed Swagger UI in the binary using `//go:embed` and serve it at `GET /docs`, always available in all environments. The raw spec is served at `GET /openapi.yaml`. |
| **Current state** | No API documentation exists. API consumers have no way to discover endpoints, required parameters, or response shapes without reading the source code. |
| **Rationale** | `swaggest/swgui` embeds Swagger UI assets directly into the Go binary — no CDN, no external dependencies, works offline and in production. Serving docs unconditionally (not behind a dev-only flag) is appropriate for a public-facing exchange rate API: **(a)** external consumers benefit from live interactive documentation, **(b)** embedding eliminates the operational complexity of serving static files separately, **(c)** both `/docs` and `/openapi.yaml` are registered as standard `http.Handler` — zero framework coupling. Both `oapi-codegen` and `swgui` consume the same `api/openapi.yaml`, so docs are always in sync with the generated types. |

### 7.3 — New Endpoints: `GET /health` and `GET /rates/{currency}`

| | Details |
|---|---|
| **Improvement** | Add `GET /health` as a structured health check and `GET /rates/{currency}` for single-currency queries. Replace the existing `/hello` stub with `/health`. |
| **Current state** | `/hello` returns a plain string with no semantic value. There is no way to query a single currency rate. All rates are returned in a single flat response. |
| **Rationale** | **(a)** `GET /health` returns `200 OK` with `{ status: "ok" }` under normal conditions and `503 Service Unavailable` with stale currency details when any rate exceeds its staleness threshold (direct integration point for Pillar 5.1). This is the standard contract for load balancers, Kubernetes liveness/readiness probes, and uptime monitors. **(b)** `GET /rates/{currency}` accepts an enum path parameter (`usd_bcv`, `eur_bcv`, `usdt_binance`) and returns a single `RateEntry` object. This reduces payload size for consumers who only need one rate and enables per-currency caching at the HTTP layer in the future. `{currency}` as an enum is enforced both in the OpenAPI spec and validated in the generated handler stub. |

### 7.4 — Historical Data Endpoint: `GET /rates/{currency}/history`

| | Details |
|---|---|
| **Improvement** | Add `GET /rates/{currency}/history?fromDate=YYYY-MM-DD&toDate=YYYY-MM-DD` to expose the `prices_history` table over HTTP. |
| **Current state** | No historical data endpoint exists. This endpoint has a hard dependency on Pillar 4.1 (PostgreSQL persistence). |
| **Rationale** | **(a)** `fromDate` and `toDate` are required query parameters of type `date` (ISO 8601, `YYYY-MM-DD`). Making them required avoids unbounded queries that would return the entire history table. **(b)** The response is an array of `{ date, value, is_average }` objects — `is_average: true` indicates the value is a daily USDT average computed by the consolidation worker, while `false` indicates an official BCV rate. **(c)** This endpoint is defined in the spec from day one (7.1) even though the backing DB layer arrives in Phase 6, allowing API consumers to plan integrations in advance. The handler returns `501 Not Implemented` until 4.1 is complete. |

---

## Additional Notes

