package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/api"
	"github.com/nanezx/ve-xchange-api/internal/config"
	"github.com/nanezx/ve-xchange-api/internal/db"
	"github.com/nanezx/ve-xchange-api/internal/handler"
	"github.com/nanezx/ve-xchange-api/internal/metrics"
	"github.com/nanezx/ve-xchange-api/internal/provider"
	"github.com/nanezx/ve-xchange-api/internal/rates"
	"github.com/nanezx/ve-xchange-api/internal/state"
	"github.com/nanezx/ve-xchange-api/internal/worker"
	v3 "github.com/swaggest/swgui/v3"
)

const shutdownTimeout = 10 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	appConfig, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	appState := state.NewState()

	// Initialise database store if DATABASE_URL is configured.
	var dbStore db.Store
	if appConfig.DatabaseURL != "" {
		if err := db.RunMigrations(appConfig.DatabaseURL); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		store, err := db.New(ctx, appConfig.DatabaseURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		dbStore = store
		defer dbStore.Close()
		slog.Info("database connected")

		// Pre-load the in-memory cache from the latest persisted values so the
		// API serves correct data immediately after a restart.
		latest, err := dbStore.GetLatestRates(ctx)
		if err != nil {
			slog.Warn("warm-up query failed, starting with empty cache", "error", err)
		} else if len(latest) > 0 {
			warm := make(map[string]state.WarmEntry, len(latest))
			for currency, entry := range latest {
				warm[currency] = state.WarmEntry{Value: entry.Value, RecordedAt: entry.RecordedAt}
			}
			state.WarmUp(appState, warm)
			slog.Info("cache warmed up from database", "currencies", len(warm))
		}
	} else {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	// Venezuela is UTC-4 (no DST).
	utcMinus4 := time.FixedZone("UTC-4", -4*60*60)

	// Add provider to lists
	providerJobs := []worker.ProviderJob{
		// DolarAPI — BCV publishes tomorrow's rate at 5-6 PM each day.
		// We fetch at 00:05 AM UTC-4 so we always apply the rate that is
		// currently valid for today, never tomorrow's rate ahead of time.
		{
			Provider: provider.NewDolarDolarApiProvider(client),
			DailyAt:  &worker.TimeOfDay{Hour: 0, Minute: 5, Location: utcMinus4},
			Apply: func(pr rates.PriceResponse) {
				state.UpdateBcvPrice(appState, pr)
				if v, ok := pr[state.KeyUsdBcv]; ok {
					metrics.RateValue.WithLabelValues(string(api.UsdBcv)).Set(v)
				}
				if v, ok := pr[state.KeyEurBcv]; ok {
					metrics.RateValue.WithLabelValues(string(api.EurBcv)).Set(v)
				}
				if dbStore != nil {
					now := time.Now()
					if v, ok := pr[state.KeyUsdBcv]; ok {
						if err := dbStore.InsertRate(ctx, string(api.UsdBcv), v, now); err != nil {
							slog.Warn("failed to persist usd_bcv rate", "error", err)
						}
					}
					if v, ok := pr[state.KeyEurBcv]; ok {
						if err := dbStore.InsertRate(ctx, string(api.EurBcv), v, now); err != nil {
							slog.Warn("failed to persist eur_bcv rate", "error", err)
						}
					}
				}
			},
			OnFail:    func(_ int64) { state.MarkBcvFailing(appState) },
			OnRecover: func() { state.ClearBcvFailing(appState) },
			AfterFetch: func(consecutiveFails int64, success bool) {
				status := "success"
				if !success {
					status = "failure"
				}
				metrics.ProviderFetchTotal.WithLabelValues("DolarAPI", status).Inc()
				metrics.ProviderConsecutiveFailures.WithLabelValues("DolarAPI").Set(float64(consecutiveFails))
			},
		},
		// P2P Binance API
		{
			Provider: provider.NewBinanceProvider(client),
			Every:    5 * time.Minute,
			Apply: func(pr rates.PriceResponse) {
				state.UpdateBinancePrice(appState, pr)
				if v, ok := pr[state.KeyUsdtBinance]; ok {
					metrics.RateValue.WithLabelValues(string(api.UsdtBinance)).Set(v)
				}
				if v, ok := pr[state.KeyUsdtBinanceBuy]; ok {
					metrics.RateValue.WithLabelValues(string(api.UsdtBinanceBuy)).Set(v)
				}
				if v, ok := pr[state.KeyUsdtBinanceSell]; ok {
					metrics.RateValue.WithLabelValues(string(api.UsdtBinanceSell)).Set(v)
				}
				if dbStore != nil {
					now := time.Now()
					if v, ok := pr[state.KeyUsdtBinance]; ok {
						if err := dbStore.InsertRate(ctx, string(api.UsdtBinance), v, now); err != nil {
							slog.Warn("failed to persist usdt_binance rate", "error", err)
						}
					}
					if v, ok := pr[state.KeyUsdtBinanceBuy]; ok {
						if err := dbStore.InsertRate(ctx, string(api.UsdtBinanceBuy), v, now); err != nil {
							slog.Warn("failed to persist usdt_binance_buy rate", "error", err)
						}
					}
					if v, ok := pr[state.KeyUsdtBinanceSell]; ok {
						if err := dbStore.InsertRate(ctx, string(api.UsdtBinanceSell), v, now); err != nil {
							slog.Warn("failed to persist usdt_binance_sell rate", "error", err)
						}
					}
				}
			},
			OnFail:    func(_ int64) { state.MarkBinanceFailing(appState) },
			OnRecover: func() { state.ClearBinanceFailing(appState) },
			AfterFetch: func(consecutiveFails int64, success bool) {
				status := "success"
				if !success {
					status = "failure"
				}
				metrics.ProviderFetchTotal.WithLabelValues("USDT", status).Inc()
				metrics.ProviderConsecutiveFailures.WithLabelValues("USDT").Set(float64(consecutiveFails))
			},
		},
	}

	workerWg := worker.StartPriceWorker(ctx, providerJobs)

	// Nightly consolidation: replace the ~288 raw Binance observations from
	// the previous day with a single daily-average row. Runs at 01:00 AM UTC-4
	// (1 hour after the BCV fetch window closes).
	taskWg := worker.StartTaskWorker(ctx, []worker.TaskJob{
		{
			Name:    "binance-consolidation",
			DailyAt: worker.TimeOfDay{Hour: 1, Minute: 0, Location: utcMinus4},
			Run: func(taskCtx context.Context) {
				now := time.Now().In(utcMinus4)
				yesterday := now.AddDate(0, 0, -1)
				from := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, utcMinus4)
				to := from.AddDate(0, 0, 1)

				currencies := []string{
					string(api.UsdtBinance),
					string(api.UsdtBinanceBuy),
					string(api.UsdtBinanceSell),
				}
				for _, ccy := range currencies {
					if err := dbStore.ConsolidateDay(taskCtx, ccy, from, to); err != nil {
						slog.Error("binance consolidation failed",
							"currency", ccy,
							"day", yesterday.Format("2006-01-02"), "error", err)
					} else {
						slog.Info("binance consolidation done",
							"currency", ccy,
							"day", yesterday.Format("2006-01-02"))
					}
				}
			},
		},
	})

	mux := http.NewServeMux()

	api.HandlerFromMux(handler.NewServerWithStore(appState, dbStore), mux)

	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi.yaml")
	})
	mux.Handle("/docs/", v3.NewHandler("ve-xchange-api", "/openapi.yaml", "/docs/"))
	mux.Handle("GET /metrics", metrics.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", appConfig.AppPort),
		Handler: metrics.Middleware(mux),
	}

	serverErrCh := make(chan error, 1)
	go func() {
		slog.Info("server started", "port", appConfig.AppPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	select {
	case err := <-serverErrCh:
		slog.Error("HTTP server failed", "error", err)
		stop()
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
	}

	workerWg.Wait()
	taskWg.Wait()
	slog.Info("shutdown complete")
}
