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
		store, err := db.New(ctx, appConfig.DatabaseURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		if err := store.CreateSchema(ctx); err != nil {
			slog.Error("failed to create schema", "error", err)
			os.Exit(1)
		}
		dbStore = store
		defer dbStore.Close()
		slog.Info("database connected")
	} else {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	// Add provider to lists
	providerJobs := []worker.ProviderJob{
		// DolarAPI
		{
			Provider: provider.NewDolarDolarApiProvider(client),
			Every:    6 * time.Hour,
			Apply: func(pr rates.PriceResponse) {
				state.UpdateBcvPrice(appState, pr)
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
		},
		// P2P Binance API
		{
			Provider: provider.NewBinanceProvider(client),
			Every:    5 * time.Minute,
			Apply: func(pr rates.PriceResponse) {
				state.UpdateBinancePrice(appState, pr)
				if dbStore != nil {
					if v, ok := pr[state.KeyUsdtBinance]; ok {
						if err := dbStore.InsertRate(ctx, string(api.UsdtBinance), v, time.Now()); err != nil {
							slog.Warn("failed to persist usdt_binance rate", "error", err)
						}
					}
				}
			},
			OnFail:    func(_ int64) { state.MarkBinanceFailing(appState) },
			OnRecover: func() { state.ClearBinanceFailing(appState) },
		},
	}

	workerWg := worker.StartPriceWorker(ctx, providerJobs)

	mux := http.NewServeMux()

	api.HandlerFromMux(handler.NewServerWithStore(appState, dbStore), mux)

	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi.yaml")
	})
	mux.Handle("/docs/", v3.NewHandler("ve-xchange-api", "/openapi.yaml", "/docs/"))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", appConfig.AppPort),
		Handler: mux,
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
	slog.Info("shutdown complete")
}
