package main

import (
	"fmt"
	"github.com/nanezx/ve-xchange-api/internal/config"
	"github.com/nanezx/ve-xchange-api/internal/handler"
	"github.com/nanezx/ve-xchange-api/internal/provider"
	"github.com/nanezx/ve-xchange-api/internal/state"
	"github.com/nanezx/ve-xchange-api/internal/worker"
	"net/http"
	"time"
)

func main() {
	err := config.LoadConfig()

	if err != nil {
		fmt.Printf("Failed to load env file... [Error]: %v", err)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Add provider to lists
	providerJobs := []worker.ProviderJob{
		// DolarVzla API
		{
			Provider: provider.NewDolarVzlaProvider(client, config.AppConfig.DolarVzlaApiKey),
			Every:    6 * time.Hour,
			Apply:    state.UpdateBcvPrice,
		},
		// P2P Binance API
		{
			Provider: provider.NewBinanceProvider(client),
			Every:    5 * time.Minute,
			Apply:    state.UpdateBinancePrice,
		},
	}

	// Start worker
	go worker.StartPriceWorker(providerJobs)

	mux := http.NewServeMux()

	mux.Handle("/hello", handler.HelloWorldHandler{})
	mux.Handle("/rates", handler.RatesHandler{})

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", config.AppConfig.AppPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.AppConfig.AppPort), mux)
	if err != nil {
		fmt.Printf("Failed to serve the API [Error]: %v", err)
		return
	}
}
