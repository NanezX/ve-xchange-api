package state

import (
	"sync"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type ExchangeRates struct {
	UsdBCV      float64   `json:"usd_bcv"`
	EurBCV      float64   `json:"eur_bcv"`
	UsdtBinance float64   `json:"usdt_binance"`
	LastUpdate  time.Time `json:"last_update"`
}

type State struct {
	mu sync.RWMutex
	Rates ExchangeRates
}

func NewState() *State {
	return &State{}
}

func (s *State) UpdateRates(newRates ExchangeRates) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Rates = newRates
}

func (s *State) GetRates() ExchangeRates {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Rates
}

// FIXME: Add safety checks that exists those values on the maps
func UpdateBcvPrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.Rates.UsdBCV = data["USD_BCV"]
	state.Rates.EurBCV = data["EUR_BCV"]
	state.Rates.LastUpdate = time.Now()
}

// FIXME: Add safety checks that exists those values on the maps
func UpdateBinancePrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.Rates.UsdtBinance = data["USDT_BINANCE"]
	state.Rates.LastUpdate = time.Now()
}
