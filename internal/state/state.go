package state

import (
	"github.com/nanezx/ve-xchange-api/internal/rates"
	"sync"
	"time"
)

type ExchangeRates struct {
	UsdBCV      float64   `json:"usd_bcv"`
	EurBCV      float64   `json:"eur_bcv"`
	UsdtBinance float64   `json:"usdt_binance"`
	LastUpdate  time.Time `json:"last_update"`
}

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

// FIXME: Add safety checks that exists those values on the maps
func UpdateBcvPrice(data rates.PriceResponse) {
	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdBCV = data["USD_BCV"]
	AppState.Rates.EurBCV = data["EUR_BCV"]
	AppState.Rates.LastUpdate = time.Now()
}

// FIXME: Add safety checks that exists those values on the maps
func UpdateBinancePrice(data rates.PriceResponse) {
	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdtBinance = data["USDT_BINANCE"]
	AppState.Rates.LastUpdate = time.Now()
}
