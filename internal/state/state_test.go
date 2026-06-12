package state

import (
	"sync"
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

func TestUpdateBcvPriceSetsValues(t *testing.T) {
	s := NewState()

	before := time.Now()
	UpdateBcvPrice(s, rates.PriceResponse{
		KeyUsdBcv: 50.5,
		KeyEurBcv: 60.0,
	})
	after := time.Now()

	r := s.GetRates()

	if r.UsdBcv.Value != 50.5 {
		t.Fatalf("expected UsdBcv=50.5, got %v", r.UsdBcv.Value)
	}
	if r.EurBcv.Value != 60.0 {
		t.Fatalf("expected EurBcv=60.0, got %v", r.EurBcv.Value)
	}
	if r.UsdBcv.LastUpdated == nil {
		t.Fatal("expected UsdBcv.LastUpdated to be set")
	}
	if r.UsdBcv.LastUpdated.Before(before) || r.UsdBcv.LastUpdated.After(after) {
		t.Fatalf("UsdBcv.LastUpdated %v outside window [%v, %v]",
			r.UsdBcv.LastUpdated, before, after)
	}
}

func TestUpdateBcvPriceSkipsMissingKeys(t *testing.T) {
	s := NewState()

	// Seed initial values
	UpdateBcvPrice(s, rates.PriceResponse{
		KeyUsdBcv: 50.0,
		KeyEurBcv: 60.0,
	})

	// Update only USD — EUR must be preserved
	UpdateBcvPrice(s, rates.PriceResponse{
		KeyUsdBcv: 55.0,
	})

	r := s.GetRates()
	if r.UsdBcv.Value != 55.0 {
		t.Fatalf("expected UsdBcv=55.0, got %v", r.UsdBcv.Value)
	}
	if r.EurBcv.Value != 60.0 {
		t.Fatalf("expected EurBcv=60.0 (preserved), got %v", r.EurBcv.Value)
	}
}

func TestUpdateBcvPriceEmptyResponsePreservesState(t *testing.T) {
	s := NewState()

	UpdateBcvPrice(s, rates.PriceResponse{KeyUsdBcv: 42.0, KeyEurBcv: 43.0})
	UpdateBcvPrice(s, rates.PriceResponse{})

	r := s.GetRates()
	if r.UsdBcv.Value != 42.0 {
		t.Fatalf("expected UsdBcv=42.0, got %v", r.UsdBcv.Value)
	}
}

// --- UpdateBinancePrice ---

func TestUpdateBinancePriceSetsValue(t *testing.T) {
	s := NewState()

	before := time.Now()
	UpdateBinancePrice(s, rates.PriceResponse{
		KeyUsdtBinance:     100.0,
		KeyUsdtBinanceBuy:  110.0,
		KeyUsdtBinanceSell: 90.0,
	})
	after := time.Now()

	r := s.GetRates()
	if r.Usdt.Value != 100.0 {
		t.Fatalf("expected Usdt=100.0, got %v", r.Usdt.Value)
	}
	if r.UsdtCompra.Value != 110.0 {
		t.Fatalf("expected UsdtCompra=110.0, got %v", r.UsdtCompra.Value)
	}
	if r.UsdtVenta.Value != 90.0 {
		t.Fatalf("expected UsdtVenta=90.0, got %v", r.UsdtVenta.Value)
	}
	if r.Usdt.LastUpdated == nil {
		t.Fatal("expected Usdt.LastUpdated to be set")
	}
	if r.Usdt.LastUpdated.Before(before) || r.Usdt.LastUpdated.After(after) {
		t.Fatalf("Usdt.LastUpdated %v outside window [%v, %v]",
			r.Usdt.LastUpdated, before, after)
	}
}

func TestUpdateBinancePriceSkipsMissingKey(t *testing.T) {
	s := NewState()

	UpdateBinancePrice(s, rates.PriceResponse{
		KeyUsdtBinance:     99.0,
		KeyUsdtBinanceBuy:  88.0,
		KeyUsdtBinanceSell: 77.0,
	})
	UpdateBinancePrice(s, rates.PriceResponse{})

	r := s.GetRates()
	if r.Usdt.Value != 99.0 {
		t.Fatalf("expected Usdt=99.0 (preserved), got %v", r.Usdt.Value)
	}
	if r.UsdtCompra.Value != 88.0 {
		t.Fatalf("expected UsdtCompra=88.0 (preserved), got %v", r.UsdtCompra.Value)
	}
	if r.UsdtVenta.Value != 77.0 {
		t.Fatalf("expected UsdtVenta=77.0 (preserved), got %v", r.UsdtVenta.Value)
	}
}

// --- UpdateRates / GetRates ---

func TestUpdateRatesReplacesAll(t *testing.T) {
	s := NewState()

	now := time.Now()
	s.UpdateRates(StateRates{
		UsdBcv:     RateData{Value: 1.0, LastUpdated: &now},
		EurBcv:     RateData{Value: 2.0, LastUpdated: &now},
		Usdt:       RateData{Value: 3.0, LastUpdated: &now},
		UsdtCompra: RateData{Value: 4.0, LastUpdated: &now},
		UsdtVenta:  RateData{Value: 5.0, LastUpdated: &now},
	})

	r := s.GetRates()
	if r.UsdBcv.Value != 1.0 || r.EurBcv.Value != 2.0 || r.Usdt.Value != 3.0 ||
		r.UsdtCompra.Value != 4.0 || r.UsdtVenta.Value != 5.0 {
		t.Fatalf("unexpected rates: %+v", r)
	}
}

func TestGetRatesOnFreshStateReturnsZeroValues(t *testing.T) {
	s := NewState()
	r := s.GetRates()

	if r.UsdBcv.Value != 0 || r.EurBcv.Value != 0 || r.Usdt.Value != 0 ||
		r.UsdtCompra.Value != 0 || r.UsdtVenta.Value != 0 {
		t.Fatalf("expected zero values, got %+v", r)
	}
	if r.UsdBcv.LastUpdated != nil || r.EurBcv.LastUpdated != nil || r.Usdt.LastUpdated != nil ||
		r.UsdtCompra.LastUpdated != nil || r.UsdtVenta.LastUpdated != nil {
		t.Fatal("expected nil LastUpdated on fresh state")
	}
}

// --- Thread safety (requires -race) ---

func TestConcurrentBcvAndBinanceUpdates(t *testing.T) {
	s := NewState()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			UpdateBcvPrice(s, rates.PriceResponse{KeyUsdBcv: 50.0, KeyEurBcv: 60.0})
		}()
		go func() {
			defer wg.Done()
			_ = s.GetRates()
		}()
	}

	wg.Wait()
}

func TestConcurrentBinanceUpdates(t *testing.T) {
	s := NewState()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			UpdateBinancePrice(s, rates.PriceResponse{
				KeyUsdtBinance:     100.0,
				KeyUsdtBinanceBuy:  110.0,
				KeyUsdtBinanceSell: 90.0,
			})
		}()
		go func() {
			defer wg.Done()
			_ = s.GetRates()
		}()
	}

	wg.Wait()
}

func TestConcurrentMixedWriters(t *testing.T) {
	s := NewState()

	const goroutines = 30
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			UpdateBcvPrice(s, rates.PriceResponse{KeyUsdBcv: 10.0, KeyEurBcv: 11.0})
		}()
		go func() {
			defer wg.Done()
			UpdateBinancePrice(s, rates.PriceResponse{
				KeyUsdtBinance:     20.0,
				KeyUsdtBinanceBuy:  22.0,
				KeyUsdtBinanceSell: 18.0,
			})
		}()
		go func() {
			defer wg.Done()
			_ = s.GetRates()
		}()
	}

	wg.Wait()
}

// --- ProviderFailing flag ---

func TestMarkBcvFailingSetsFlag(t *testing.T) {
	s := NewState()
	MarkBcvFailing(s)

	r := s.GetRates()
	if !r.UsdBcv.ProviderFailing {
		t.Fatal("expected UsdBcv.ProviderFailing=true")
	}
	if !r.EurBcv.ProviderFailing {
		t.Fatal("expected EurBcv.ProviderFailing=true")
	}
	if r.Usdt.ProviderFailing {
		t.Fatal("expected Usdt.ProviderFailing=false (unaffected)")
	}
}

func TestClearBcvFailingClearsFlag(t *testing.T) {
	s := NewState()
	MarkBcvFailing(s)
	ClearBcvFailing(s)

	r := s.GetRates()
	if r.UsdBcv.ProviderFailing || r.EurBcv.ProviderFailing {
		t.Fatal("expected ProviderFailing=false after ClearBcvFailing")
	}
}

func TestMarkBinanceFailingSetsFlag(t *testing.T) {
	s := NewState()
	MarkBinanceFailing(s)

	r := s.GetRates()
	if !r.Usdt.ProviderFailing {
		t.Fatal("expected Usdt.ProviderFailing=true")
	}
	if !r.UsdtCompra.ProviderFailing {
		t.Fatal("expected UsdtCompra.ProviderFailing=true")
	}
	if !r.UsdtVenta.ProviderFailing {
		t.Fatal("expected UsdtVenta.ProviderFailing=true")
	}
	if r.UsdBcv.ProviderFailing || r.EurBcv.ProviderFailing {
		t.Fatal("expected BCV flags unaffected by MarkBinanceFailing")
	}
}

func TestClearBinanceFailingClearsFlag(t *testing.T) {
	s := NewState()
	MarkBinanceFailing(s)
	ClearBinanceFailing(s)

	r := s.GetRates()
	if r.Usdt.ProviderFailing {
		t.Fatal("expected Usdt.ProviderFailing=false after ClearBinanceFailing")
	}
	if r.UsdtCompra.ProviderFailing {
		t.Fatal("expected UsdtCompra.ProviderFailing=false after ClearBinanceFailing")
	}
	if r.UsdtVenta.ProviderFailing {
		t.Fatal("expected UsdtVenta.ProviderFailing=false after ClearBinanceFailing")
	}
}

func TestConcurrentFailingFlagUpdates(t *testing.T) {
	s := NewState()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			MarkBcvFailing(s)
			ClearBcvFailing(s)
		}()
		go func() {
			defer wg.Done()
			_ = s.GetRates()
		}()
	}

	wg.Wait()
}

// --- WarmUp ---

func TestWarmUpSetsAllCurrencies(t *testing.T) {
	s := NewState()
	ts := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	WarmUp(s, map[string]WarmEntry{
		KeyUsdBcv:          {Value: 480.5, RecordedAt: ts},
		KeyEurBcv:          {Value: 520.0, RecordedAt: ts},
		KeyUsdtBinance:     {Value: 530.1, RecordedAt: ts},
		KeyUsdtBinanceBuy:  {Value: 540.0, RecordedAt: ts},
		KeyUsdtBinanceSell: {Value: 520.0, RecordedAt: ts},
	})

	r := s.GetRates()
	if r.UsdBcv.Value != 480.5 {
		t.Fatalf("expected UsdBcv=480.5, got %f", r.UsdBcv.Value)
	}
	if r.EurBcv.Value != 520.0 {
		t.Fatalf("expected EurBcv=520.0, got %f", r.EurBcv.Value)
	}
	if r.Usdt.Value != 530.1 {
		t.Fatalf("expected Usdt=530.1, got %f", r.Usdt.Value)
	}
	if r.UsdtCompra.Value != 540.0 {
		t.Fatalf("expected UsdtCompra=540.0, got %f", r.UsdtCompra.Value)
	}
	if r.UsdtVenta.Value != 520.0 {
		t.Fatalf("expected UsdtVenta=520.0, got %f", r.UsdtVenta.Value)
	}
	if r.UsdBcv.LastUpdated == nil || !r.UsdBcv.LastUpdated.Equal(ts) {
		t.Fatalf("expected UsdBcv.LastUpdated=%v, got %v", ts, r.UsdBcv.LastUpdated)
	}
}

func TestWarmUpPartialMapOnlyUpdatesPresent(t *testing.T) {
	s := NewState()
	ts := time.Now()

	// Pre-set a known Binance value.
	binanceTs := ts.Add(-1 * time.Minute)
	WarmUp(s, map[string]WarmEntry{
		KeyUsdtBinance:     {Value: 99.9, RecordedAt: binanceTs},
		KeyUsdtBinanceBuy:  {Value: 88.8, RecordedAt: binanceTs},
		KeyUsdtBinanceSell: {Value: 77.7, RecordedAt: binanceTs},
	})

	// Now warm up only BCV — Binance should stay untouched.
	WarmUp(s, map[string]WarmEntry{
		KeyUsdBcv: {Value: 480.0, RecordedAt: ts},
		KeyEurBcv: {Value: 520.0, RecordedAt: ts},
	})

	r := s.GetRates()
	if r.UsdBcv.Value != 480.0 {
		t.Fatalf("expected UsdBcv=480.0, got %f", r.UsdBcv.Value)
	}
	if r.Usdt.Value != 99.9 {
		t.Fatalf("expected Usdt unchanged=99.9, got %f", r.Usdt.Value)
	}
	if r.UsdtCompra.Value != 88.8 {
		t.Fatalf("expected UsdtCompra unchanged=88.8, got %f", r.UsdtCompra.Value)
	}
	if r.UsdtVenta.Value != 77.7 {
		t.Fatalf("expected UsdtVenta unchanged=77.7, got %f", r.UsdtVenta.Value)
	}
}

func TestWarmUpEmptyMapIsNoop(t *testing.T) {
	s := NewState()
	WarmUp(s, map[string]WarmEntry{})

	r := s.GetRates()
	if r.UsdBcv.Value != 0 || r.UsdBcv.LastUpdated != nil {
		t.Fatal("expected zero state after empty WarmUp")
	}
}

func TestWarmUpDoesNotSetProviderFailing(t *testing.T) {
	s := NewState()
	ts := time.Now()

	WarmUp(s, map[string]WarmEntry{
		KeyUsdBcv: {Value: 480.0, RecordedAt: ts},
	})

	r := s.GetRates()
	if r.UsdBcv.ProviderFailing {
		t.Fatal("WarmUp must not set ProviderFailing")
	}
}
