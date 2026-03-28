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
	// TODO: Read from stored or something - The values should be updated independently
	dolarVzlaData, err := fetchDolarVzlaBcv()
	if err != nil {
		http.Error(w, "Error obteniendo datos del BCV", http.StatusInternalServerError)
		return
	}

	binaProvider := NewBinanceProvider()

	bnbPrices, err := binaProvider.GetPrices()
	if err != nil {
		fmt.Printf("error bnb: %v", err)
		http.Error(w, "Error obteniendo datos del Binance p2p", http.StatusInternalServerError)
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
