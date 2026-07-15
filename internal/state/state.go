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
	KeyUsdtBinance     = "USDT"
	KeyUsdtBinanceBuy  = "USDT_BINANCE_SELL"
	KeyUsdtBinanceSell = "USDT_BINANCE_BUY"
)

type RateData struct {
	Value           float64
	LastUpdated     *time.Time
	ProviderFailing bool
}

type StateRates struct {
	UsdBcv     RateData
	EurBcv     RateData
	Usdt       RateData
	UsdtCompra RateData
	UsdtVenta  RateData
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
	s.rates.Usdt.ProviderFailing = true
	s.rates.UsdtCompra.ProviderFailing = true
	s.rates.UsdtVenta.ProviderFailing = true
}

// ClearBinanceFailing clears the failing flag for the Binance rate.
func ClearBinanceFailing(s *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates.Usdt.ProviderFailing = false
	s.rates.UsdtCompra.ProviderFailing = false
	s.rates.UsdtVenta.ProviderFailing = false
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
		s.rates.Usdt.Value = e.Value
		s.rates.Usdt.LastUpdated = &t
	}
	if e, ok := entries[KeyUsdtBinanceBuy]; ok {
		t := e.RecordedAt
		s.rates.UsdtCompra.Value = e.Value
		s.rates.UsdtCompra.LastUpdated = &t
	}
	if e, ok := entries[KeyUsdtBinanceSell]; ok {
		t := e.RecordedAt
		s.rates.UsdtVenta.Value = e.Value
		s.rates.UsdtVenta.LastUpdated = &t
	}
}

// Missing keys are skipped (previous value preserved) so a partial provider
// failure cannot silently overwrite valid data with zeros.
func UpdateBcvPrice(state *State, data rates.PriceResponse) {
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	if v, ok := data.Values[KeyUsdBcv]; ok {
		state.rates.UsdBcv.Value = v
		state.rates.UsdBcv.LastUpdated = &now
	}
	if v, ok := data.Values[KeyEurBcv]; ok {
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
	if v, ok := data.Values[KeyUsdtBinance]; ok {
		state.rates.Usdt.Value = v
		state.rates.Usdt.LastUpdated = &now
	}
	if v, ok := data.Values[KeyUsdtBinanceBuy]; ok {
		state.rates.UsdtCompra.Value = v
		state.rates.UsdtCompra.LastUpdated = &now
	}
	if v, ok := data.Values[KeyUsdtBinanceSell]; ok {
		state.rates.UsdtVenta.Value = v
		state.rates.UsdtVenta.LastUpdated = &now
	}
}
