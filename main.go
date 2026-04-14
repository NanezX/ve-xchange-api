package main

import (
	"fmt"
	"github.com/nanezx/ve-xchange-api/internal/config"
	"github.com/nanezx/ve-xchange-api/internal/provider"
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
	providerJobs := []ProviderJob{
		// DolarVzla API
		{
			Provider: provider.NewDolarVzlaProvider(client, config.AppConfig.DolarVzlaApiKey),
			Every:    6 * time.Hour,
			Apply:    UpdateBcvPrice,
		},
		// P2P Binance API
		{
			Provider: provider.NewBinanceProvider(client),
			Every:    5 * time.Minute,
			Apply:    UpdateBinancePrice,
		},
	}

	// Start worker
	go StartPriceWorker(providerJobs)

	mux := http.NewServeMux()

	mux.Handle("/hello", HelloWorldHandler{})
	mux.Handle("/rates", RatesHandler{})

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", config.AppConfig.AppPort)
	http.ListenAndServe(fmt.Sprintf(":%d", config.AppConfig.AppPort), mux)
}
