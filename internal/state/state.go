package state

import (
	"sync"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

// Map keys produced by each provider. Kept here as the single source of
// truth so writers and readers cannot drift out of sync silently.
const (
	KeyUsdBcv          = "USD_BCV"
	KeyEurBcv          = "EUR_BCV"
	KeyUsdtBinance     = "USDT_BINANCE"
	KeyUsdtBinanceBuy  = "USDT_BINANCE_BUY"
	KeyUsdtBinanceSell = "USDT_BINANCE_SELL"
)

type RateData struct {
	Value           float64
	LastUpdated     *time.Time
	ProviderFailing bool
}

type StateRates struct {
	UsdBcv          RateData
	EurBcv          RateData
	UsdtBinance     RateData
	UsdtBinanceBuy  RateData
	UsdtBinanceSell RateData
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
	s.rates.UsdtBinanceBuy.ProviderFailing = true
	s.rates.UsdtBinanceSell.ProviderFailing = true
}

// ClearBinanceFailing clears the failing flag for the Binance rate.
func ClearBinanceFailing(s *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates.UsdtBinance.ProviderFailing = false
	s.rates.UsdtBinanceBuy.ProviderFailing = false
	s.rates.UsdtBinanceSell.ProviderFailing = false
}

// WarmEntry is a single pre-loaded rate value used to warm the in-memory cache.
type WarmEntry struct {
	Value      float64
	RecordedAt time.Time
}

// WarmUp pre-loads the in-memory cache from persisted values so the API
// can serve correct data immediately after a restart, without waiting for
// the first provider tick. Only currencies present in the map are updated;
// missing keys are left at their zero value.
func WarmUp(s *State, entries map[string]WarmEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := entries[KeyUsdBcv]; ok {
		t := e.RecordedAt
		s.rates.UsdBcv.Value = e.Value
		s.rates.UsdBcv.LastUpdated = &t
	}
	if e, ok := entries[KeyEurBcv]; ok {
		t := e.RecordedAt
		s.rates.EurBcv.Value = e.Value
		s.rates.EurBcv.LastUpdated = &t
	}
	if e, ok := entries[KeyUsdtBinance]; ok {
		t := e.RecordedAt
		s.rates.UsdtBinance.Value = e.Value
		s.rates.UsdtBinance.LastUpdated = &t
	}
	if e, ok := entries[KeyUsdtBinanceBuy]; ok {
		t := e.RecordedAt
		s.rates.UsdtBinanceBuy.Value = e.Value
		s.rates.UsdtBinanceBuy.LastUpdated = &t
	}
	if e, ok := entries[KeyUsdtBinanceSell]; ok {
		t := e.RecordedAt
		s.rates.UsdtBinanceSell.Value = e.Value
		s.rates.UsdtBinanceSell.LastUpdated = &t
	}
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

// UpdateBinancePrice writes the USDT/Binance values from a provider response.
// Missing keys are skipped (previous value preserved).
func UpdateBinancePrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	if v, ok := data[KeyUsdtBinance]; ok {
		state.rates.UsdtBinance.Value = v
		state.rates.UsdtBinance.LastUpdated = &now
	}
	if v, ok := data[KeyUsdtBinanceBuy]; ok {
		state.rates.UsdtBinanceBuy.Value = v
		state.rates.UsdtBinanceBuy.LastUpdated = &now
	}
	if v, ok := data[KeyUsdtBinanceSell]; ok {
		state.rates.UsdtBinanceSell.Value = v
		state.rates.UsdtBinanceSell.LastUpdated = &now
	}
}
