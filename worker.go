package main

import (
	"fmt"
	"net/http"
	"time"
)

func StartPriceWorker(client *http.Client) {
	// DolarVzla API
	dolarVzlaProvider := NewDolarVzlaProvider(client, AppConfig)

	// P2P Binance API
	binanceProvider := NewBinanceProvider(client)

	// // Timers/Tickers
	bcvTicker := time.NewTicker(6 * time.Hour)
	binanceTicker := time.NewTicker(5 * time.Minute)
	// Defer the tickers
	defer bcvTicker.Stop()
	defer binanceTicker.Stop()

	// Start prices
	updateBcv(dolarVzlaProvider)
	updateBinance(binanceProvider)

	for {
		select {
		case <-bcvTicker.C:
			fmt.Println("Updating BCV price with DolarVzla Provider")
			updateBcv(dolarVzlaProvider)
		case <-binanceTicker.C:
			fmt.Println("Updating USDT price with Binance P2P Provider")
			updateBinance(binanceProvider)
		}

	}
}

func updateBinance(p *BinanceProvider) {
	data, err := p.GetPrices()

	if err != nil {
		fmt.Printf("Error Binance P2P: %v", err)
		return
	}

	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdtBinance = data["USDT_BINANCE"]
	AppState.Rates.LastUpdate = time.Now()
}

func updateBcv(p *DolarVzlaProvider) {
	data, err := p.GetPrices()

	if err != nil {
		fmt.Printf("Error Dolar Vzla: %v", err)
		return
	}

	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdBCV = data["USD_BCV"]
	AppState.Rates.EurBCV = data["EUR_BCV"]
	AppState.Rates.LastUpdate = time.Now()
}
