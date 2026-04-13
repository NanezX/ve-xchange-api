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
6. [Suggested Implementation Order](#suggested-implementation-order)

---

## Pillar 1: Testing & Code Quality

### 1.1 — Unit Testing with HTTP Client Mocking

| | Details |
|---|---|
| **Improvement** | Create unit tests for `BinaceProvider` and `DolarVzlaProvider` by injecting a mock HTTP client. |
| **Current state** | No tests exist in the project. Both providers (`binanceProvider.go`, `dolarVzlaProvider.go`) create an `http.Client{Timeout: 10 * time.Second}` internally within each method (`fetchPrices`, `GetPrices`), making it **impossible** to substitute the client with a mock in tests. |
| **Rationale** | Testing providers without calling the real APIs is critical because: **(a)** external APIs are slow and introduce flakiness into tests, **(b)** they have rate-limits and in the case of DolarVzla require a paid API key, **(c)** their responses change constantly, making it impossible to verify expected values. Without mocks, tests would be fragile, slow, and non-deterministic. The solution is to define an `HTTPDoer` interface (with method `Do(*http.Request) (*http.Response, error)`) and inject it into providers via constructor. In production, the real `http.Client` is passed; in tests, a mock returning predefined responses is used. |

### 1.2 — Leverage the Existing `PriceProvider` Interface to Test the Worker

| | Details |
|---|---|
| **Improvement** | Refactor `updateBcv` and `updateBinance` in `worker.go` to accept `PriceProvider` (the interface) instead of concrete types. |
| **Current state** | The `PriceProvider` interface already exists in `types.go` with methods `GetPrices()` and `GetName()`, but the worker uses concrete types directly (`*BinaceProvider`, `*DolarVzlaProvider`). Functions `updateBcv(p *DolarVzlaProvider)` and `updateBinance(p *BinaceProvider)` accept concrete pointers. |
| **Rationale** | If the update functions accepted `PriceProvider`, they could be tested with mock structs that implement the interface, without depending on real APIs. This allows verifying that the worker updates `AppState` correctly (prices, timestamps) and handles errors without crashing — all without network access. The interface already exists; it just needs to be used. |

### 1.3 — Test the `/rates` Handler

| | Details |
|---|---|
| **Improvement** | Write tests for `RatesHandler` using `httptest.NewRecorder()` with an injected state. |
| **Current state** | The `RatesHandler` in `handlers.go` accesses the global `AppState` directly, preventing isolated testing. |
| **Rationale** | Injecting the state as a dependency (struct field or constructor parameter) allows: **(a)** pre-loading a state with known values, **(b)** verifying the handler returns valid JSON with `Content-Type: application/json`, **(c)** using `httptest.NewRecorder` to simulate requests without starting a real server. This type of test is standard in Go and executes in milliseconds. |

---

## Pillar 2: CI/CD with GitHub Actions

### 2.1 — CI Workflow: `go test` + `golangci-lint` on Every Push/PR

| | Details |
|---|---|
| **Improvement** | Create a GitHub Actions workflow that automatically runs tests and linting on every Push and Pull Request to `main`. |
| **Current state** | No CI/CD pipeline exists. Code enters `main` without any automated validation. |
| **Rationale** | Automating `go test ./...` and `golangci-lint run` on every Push and PR ensures that: **(a)** no change breaks existing tests (regression), **(b)** style errors, potential bugs, and code smells are detected before merge, **(c)** a minimum quality standard is established that scales with the team. Without CI, the only quality gate is individual discipline — which inevitably fails. |

### 2.2 — `.github/workflows/ci.yml` File

| | Details |
|---|---|
| **Improvement** | Create the workflow configuration file with concrete CI steps. |
| **Current state** | The `.github/workflows/` directory does not exist. |
| **Rationale** | The workflow should include: **(1)** `actions/setup-go` with the Go version from `go.mod`, **(2)** `go vet ./...` for basic static analysis, **(3)** `golangci/golangci-lint-action` for advanced linting, **(4)** `go test -race -coverprofile=coverage.out ./...` for tests with data race detection. The `-race` flag is especially important because the project uses goroutines and mutex — it will detect data races that normal tests miss. Optionally, upload the coverage report as an artifact. |

### 2.3 — Branch Protection on `main`

| | Details |
|---|---|
| **Improvement** | Configure a branch protection rule on GitHub requiring green CI to merge into `main`. |
| **Current state** | No branch protection configured. |
| **Rationale** | Without branch protection, CI exists but can be ignored. Configuring "Require status checks to pass before merging" ensures the pipeline is a mandatory gate, not an optional one. This prevents accidental merges of broken code and establishes a quality culture from the start of the project. |

---

## Pillar 3: Architecture & Refactoring

### 3.1 — Single, Reusable `HttpClient` (Injected)

| | Details |
|---|---|
| **Improvement** | Create a single `http.Client` in `main()` with centralized configuration and pass it to all providers via constructor. |
| **Current state** | Both `BinaceProvider.fetchPrices()` and `DolarVzlaProvider.GetPrices()` create a **new** `http.Client{Timeout: 10 * time.Second}` on **every call**. |
| **Rationale** | Creating a new `http.Client` per request wastes Go's `http.Transport` **connection pooling**. The runtime reuses TCP connections (keep-alive) **only** if the same client/transport is shared. A client injected once: **(a)** reuses TCP connections, reducing latency and file descriptor consumption, **(b)** allows centralized configuration of timeouts, transport, and proxy, **(c)** enables testing with a mock (ties into improvement 1.1). Creating clients per request is a recognized anti-pattern in Go: the `net/http` documentation itself states *"Clients should be reused instead of created as needed."* |

### 3.2 — Generic `FetchJSON[T]` Function

| | Details |
|---|---|
| **Improvement** | Implement a generic function `FetchJSON[T any](client HTTPDoer, req *http.Request) (T, error)` that encapsulates the repeated pattern of HTTP request → verify status → decode JSON. |
| **Current state** | Both providers repeat the same pattern: create request → execute with client → check `StatusCode` → read error body on failure → create decoder → decode JSON → handle error. That's approximately 25 duplicated lines in each provider. |
| **Rationale** | Duplication violates DRY and multiplies the places where a bug can hide. A generic function: **(a)** reduces the duplication to a single testable implementation, **(b)** centralizes HTTP error handling (status codes, error body), **(c)** any future improvement (retry, logging, metrics, tracing) is automatically applied to all providers without touching their code. Go 1.18+ supports generics, enabling strong typing without sacrificing reusability. |

### 3.3 — Migration to Standard Folder Structure (`/internal`, `/cmd`)

| | Details |
|---|---|
| **Improvement** | Reorganize the code into: `cmd/server/main.go`, `internal/provider/`, `internal/state/`, `internal/handler/`, `internal/config/`, `internal/worker/`. |
| **Current state** | All files are in the `main` package at the project root. There is no folder separation or package-level separation of concerns. |
| **Rationale** | A flat `main` package creates the following problems as it grows: **(a)** types and functions cannot be imported from other projects or tools, **(b)** there is no encapsulation — everything is accessible from everywhere, **(c)** name collisions become frequent. The `internal/` structure is the de facto standard in Go: the compiler **prevents** external packages from importing code inside `internal/`, protecting the module's public API. `cmd/server/` separates the entrypoint from business logic, allowing future entrypoints (CLI, separate workers, etc.) to be added. |

### 3.4 — Remove Global Variables (`AppState`, `AppConfig`)

| | Details |
|---|---|
| **Improvement** | Create instances of `State` and `Config` in `main()` and inject them as dependencies into handlers, worker, and providers (constructor injection). |
| **Current state** | `AppState` (`state.go`) and `AppConfig` (`config.go`) are package-level global variables, accessed directly from `worker.go`, `handlers.go`, and `dolarVzlaProvider.go`. |
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
Phase 1 — Foundation (Architecture)
  3.1  Single, injected HttpClient
  3.2  Generic FetchJSON[T] function
  3.4  Remove global variables (dependency injection)
  3.3  Migration to standard folder structure

Phase 2 — Testing
  1.1  Unit Testing with HTTP mocking
  1.2  Worker tests with PriceProvider mock
  1.3  /rates handler tests

Phase 3 — CI/CD
  2.1  GitHub Actions workflow (go test + lint)
  2.2  .github/workflows/ci.yml file
  2.3  Branch protection on main

Phase 4 — Resilience & Observability
  4.2  Graceful Shutdown
  4.3  Retry with exponential backoff
  5.1  Stale data detection
  5.2  Structured logging with log/slog
  5.3  Consecutive failure counter

Phase 5 — Persistence
  4.1  State as cache + DB for historical data
```

> **Why this order?**
>
> - **Phase 1 first** because the architecture refactors (dependency injection, removing globals) are a **prerequisite** for writing quality unit tests.
> - **Phase 2 before Phase 3** because you need tests for CI to have something to run.
> - **Phase 3 before Phase 4** because CI will catch regressions when you implement resilience changes.
> - **Phase 5 last** because it requires the most additional infrastructure and benefits the most from having everything else in place (tests, CI, graceful shutdown, logging).

---

## Additional Notes

- **TODO in `types.go`**: There is an existing TODO to separate mutexes per provider. This is naturally resolved by removing globals (improvement 3.4) and moving each provider to its own package (improvement 3.3).
