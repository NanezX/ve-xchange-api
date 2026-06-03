package state

import (
	"sync"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

// Map keys produced by each provider. Kept here as the single source of
// truth so writers and readers cannot drift out of sync silently.
const (
	KeyUsdBcv      = "USD_BCV"
	KeyEurBcv      = "EUR_BCV"
	KeyUsdtBinance = "USDT_BINANCE"
)

type RateData struct {
	Value           float64
	LastUpdated     *time.Time
	ProviderFailing bool
}

type StateRates struct {
	UsdBcv      RateData
	EurBcv      RateData
	UsdtBinance RateData
}

type State struct {
	mu    sync.RWMutex
	rates StateRates
}

func NewState() *State {
	return &State{}
}

func (s *State) UpdateRates(newRates StateRates) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rates = newRates
}

func (s *State) GetRates() StateRates {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rates
}

// MarkBcvFailing marks BCV rates (USD and EUR) as coming from a failing
// provider. The handler will report them as stale regardless of timestamp.
func MarkBcvFailing(s *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates.UsdBcv.ProviderFailing = true
	s.rates.EurBcv.ProviderFailing = true
}

// ClearBcvFailing clears the failing flag for BCV rates.
func ClearBcvFailing(s *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates.UsdBcv.ProviderFailing = false
	s.rates.EurBcv.ProviderFailing = false
}

// MarkBinanceFailing marks the Binance rate as coming from a failing provider.
func MarkBinanceFailing(s *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates.UsdtBinance.ProviderFailing = true
}

// ClearBinanceFailing clears the failing flag for the Binance rate.
func ClearBinanceFailing(s *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates.UsdtBinance.ProviderFailing = false
}

// Missing keys are skipped (previous value preserved) so a partial provider
// failure cannot silently overwrite valid data with zeros.
func UpdateBcvPrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	if v, ok := data[KeyUsdBcv]; ok {
		state.rates.UsdBcv.Value = v
		state.rates.UsdBcv.LastUpdated = &now
	}
	if v, ok := data[KeyEurBcv]; ok {
		state.rates.EurBcv.Value = v
		state.rates.EurBcv.LastUpdated = &now
	}
}

// UpdateBinancePrice writes the USDT/Binance value from a provider response.
// Missing key is skipped (previous value preserved).
func UpdateBinancePrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	if v, ok := data[KeyUsdtBinance]; ok {
		state.rates.UsdtBinance.Value = v
		state.rates.UsdtBinance.LastUpdated = &now
	}
}
