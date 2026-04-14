package main

import (
	"sync"
	"time"
)

type State struct {
	sync.RWMutex
	Rates ExchangeRates
}

var AppState = &State{}

func (s *State) UpdateRates(newRates ExchangeRates) {
	s.Lock()
	defer s.Unlock()
	s.Rates = newRates
}

func (s *State) GetRates() ExchangeRates {
	s.RLock()
	defer s.RUnlock()
	return s.Rates
}

func UpdateBcvPrice(data PriceResponse) {
	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdBCV = data["USD_BCV"]
	AppState.Rates.EurBCV = data["EUR_BCV"]
	AppState.Rates.LastUpdate = time.Now()
}

func UpdateBinancePrice(data PriceResponse) {
	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdtBinance = data["USDT_BINANCE"]
	AppState.Rates.LastUpdate = time.Now()
}
