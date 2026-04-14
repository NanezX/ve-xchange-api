package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	err := LoadConfig()

	if err != nil {
		fmt.Printf("Failed to load env file... [Error]: %v", err)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// DolarVzla API
	dolarVzlaProvider := NewDolarVzlaProvider(client, AppConfig)

	// P2P Binance API
	binanceProvider := NewBinanceProvider(client)

	// Add provider to lists
	providerJobs := []ProviderJob{
		{
			Provider: dolarVzlaProvider,
			Every:    6 * time.Hour,
			Apply:    UpdateBcvPrice,
		},
		{
			Provider: binanceProvider,
			Every:    5 * time.Minute,
			Apply:    UpdateBinancePrice,
		},
	}

	// Start worker
	go StartPriceWorker(providerJobs)

	mux := http.NewServeMux()

	mux.Handle("/hello", HelloWorldHandler{})
	mux.Handle("/rates", RatesHandler{})

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", AppConfig.AppPort)
	http.ListenAndServe(fmt.Sprintf(":%d", AppConfig.AppPort), mux)
}
