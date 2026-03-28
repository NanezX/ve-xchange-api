package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HelloWorldHandler struct{}

func (HelloWorldHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world!")
}

type RatesHandler struct{}

func (RatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dolarVzlaData, err := fetchDolarVzlaBcv()
	if err != nil {
		return
	}

	binaProvider := NewBinanceProvider()

	bnbPrices, err := binaProvider.GetPrices()
	if err != nil {
		fmt.Printf("error bnb: %v", err)
		return
	}

	//
	data := ExchageRates{
		BCV:        dolarVzlaData.Current.USD,
		Binance:    bnbPrices[binaProvider.GetName()],
		LastUpdate: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(data)
}
