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
	providers := []PriceProvider{dolarVzlaProvider, binanceProvider}

	// Start worker
	go StartPriceWorker(providers)

	mux := http.NewServeMux()

	mux.Handle("/hello", HelloWorldHandler{})
	mux.Handle("/rates", RatesHandler{})

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", AppConfig.AppPort)
	http.ListenAndServe(fmt.Sprintf(":%d", AppConfig.AppPort), mux)
}
