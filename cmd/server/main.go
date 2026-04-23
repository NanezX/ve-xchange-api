package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/config"
	"github.com/nanezx/ve-xchange-api/internal/handler"
	"github.com/nanezx/ve-xchange-api/internal/provider"
	"github.com/nanezx/ve-xchange-api/internal/rates"
	"github.com/nanezx/ve-xchange-api/internal/state"
	"github.com/nanezx/ve-xchange-api/internal/worker"
)

func main() {
	appConfig, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load app configuration [Error]: %v", err)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	appState := state.NewState()

	// Add provider to lists
	providerJobs := []worker.ProviderJob{
		// DolarVzla API
		{
			Provider: provider.NewDolarDolarApiProvider(client),
			Every:    6 * time.Hour,
			Apply:    func(pr rates.PriceResponse) { state.UpdateBcvPrice(appState, pr) },
		},
		// P2P Binance API
		{
			Provider: provider.NewBinanceProvider(client),
			Every:    5 * time.Minute,
			Apply:    func(pr rates.PriceResponse) { state.UpdateBinancePrice(appState, pr) },
		},
	}

	// Start worker
	go worker.StartPriceWorker(providerJobs)

	mux := http.NewServeMux()

	mux.Handle("/hello", handler.HelloWorldHandler{})
	mux.Handle("/rates", handler.NewRatesHandler(appState))

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", appConfig.AppPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", appConfig.AppPort), mux)
	if err != nil {
		fmt.Printf("Failed to serve the API [Error]: %v", err)
		return
	}
}
