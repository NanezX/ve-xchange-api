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
	UpdateBinancePrice(s, rates.PriceResponse{KeyUsdtBinance: 100.0})
	after := time.Now()

	r := s.GetRates()
	if r.UsdtBinance.Value != 100.0 {
		t.Fatalf("expected UsdtBinance=100.0, got %v", r.UsdtBinance.Value)
	}
	if r.UsdtBinance.LastUpdated == nil {
		t.Fatal("expected UsdtBinance.LastUpdated to be set")
	}
	if r.UsdtBinance.LastUpdated.Before(before) || r.UsdtBinance.LastUpdated.After(after) {
		t.Fatalf("UsdtBinance.LastUpdated %v outside window [%v, %v]",
			r.UsdtBinance.LastUpdated, before, after)
	}
}

func TestUpdateBinancePriceSkipsMissingKey(t *testing.T) {
	s := NewState()

	UpdateBinancePrice(s, rates.PriceResponse{KeyUsdtBinance: 99.0})
	UpdateBinancePrice(s, rates.PriceResponse{})

	r := s.GetRates()
	if r.UsdtBinance.Value != 99.0 {
		t.Fatalf("expected UsdtBinance=99.0 (preserved), got %v", r.UsdtBinance.Value)
	}
}

// --- UpdateRates / GetRates ---

func TestUpdateRatesReplacesAll(t *testing.T) {
	s := NewState()

	now := time.Now()
	s.UpdateRates(StateRates{
		UsdBcv:      RateData{Value: 1.0, LastUpdated: &now},
		EurBcv:      RateData{Value: 2.0, LastUpdated: &now},
		UsdtBinance: RateData{Value: 3.0, LastUpdated: &now},
	})

	r := s.GetRates()
	if r.UsdBcv.Value != 1.0 || r.EurBcv.Value != 2.0 || r.UsdtBinance.Value != 3.0 {
		t.Fatalf("unexpected rates: %+v", r)
	}
}

func TestGetRatesOnFreshStateReturnsZeroValues(t *testing.T) {
	s := NewState()
	r := s.GetRates()

	if r.UsdBcv.Value != 0 || r.EurBcv.Value != 0 || r.UsdtBinance.Value != 0 {
		t.Fatalf("expected zero values, got %+v", r)
	}
	if r.UsdBcv.LastUpdated != nil || r.EurBcv.LastUpdated != nil || r.UsdtBinance.LastUpdated != nil {
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
			UpdateBinancePrice(s, rates.PriceResponse{KeyUsdtBinance: 100.0})
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
			UpdateBinancePrice(s, rates.PriceResponse{KeyUsdtBinance: 20.0})
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
	if r.UsdtBinance.ProviderFailing {
		t.Fatal("expected UsdtBinance.ProviderFailing=false (unaffected)")
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
	if !r.UsdtBinance.ProviderFailing {
		t.Fatal("expected UsdtBinance.ProviderFailing=true")
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
	if r.UsdtBinance.ProviderFailing {
		t.Fatal("expected ProviderFailing=false after ClearBinanceFailing")
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

