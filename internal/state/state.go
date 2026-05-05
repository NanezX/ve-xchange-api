package state

import (
	"github.com/nanezx/ve-xchange-api/internal/api"
	"github.com/nanezx/ve-xchange-api/internal/rates"
	"sync"
	"time"
)

type State struct {
	mu    sync.RWMutex
	rates api.AllRates
}

func NewState() *State {
	return &State{}
}

func (s *State) UpdateRates(newRates api.AllRates) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates = newRates
}

func (s *State) GetRates() api.AllRates {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rates
}

// FIXME: Add safety checks that exists those values on the maps
func UpdateBcvPrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	// TODO: Add/fill all the values on each entry
	now := time.Now()
	state.rates.UsdBcv.Value = data["USD_BCV"]
	state.rates.UsdBcv.LastUpdated = &now
	state.rates.EurBcv.Value = data["EUR_BCV"]
	state.rates.EurBcv.LastUpdated = &now

}

// FIXME: Add safety checks that exists those values on the maps
func UpdateBinancePrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	// TODO: Add/fill all the values on each entry
	now := time.Now()
	state.rates.UsdtBinance.Value = data["USDT_BINANCE"]
	state.rates.UsdtBinance.LastUpdated = &now

}
