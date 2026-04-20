package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/state"
)

func TestRatesHandlerSuccess(t *testing.T) {
	appState := state.NewState()
	appState.UpdateRates(state.ExchangeRates{UsdBCV: 480, EurBCV: 510, UsdtBinance: 530, LastUpdate: time.Now()})

	handler := NewRatesHandler(appState)

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Expected JSON content-type")
	}

	var response state.ExchangeRates
	json.NewDecoder(w.Body).Decode(&response)

	if response.UsdBCV != appState.GetRates().UsdBCV {
		t.Fatalf("Expected %v, got %v", appState.GetRates().UsdBCV, response.UsdBCV)
	}

	if response.EurBCV != appState.GetRates().EurBCV {
		t.Fatalf("Expected %v, got %v", appState.GetRates().EurBCV, response.EurBCV)
	}

	if response.UsdtBinance != appState.GetRates().UsdtBinance {
		t.Fatalf("Expected %v, got %v", appState.GetRates().UsdtBinance, response.UsdtBinance)
	}

	if response.LastUpdate.IsZero() {
		t.Fatalf("Expected LastUpdate to be populated")
	}
}

func TestRatesHaTestRatesHandlerEmptyStatendlerSuccess(t *testing.T) {
	appState := state.NewState()

	handler := NewRatesHandler(appState)

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Expected JSON content-type")
	}

	var response state.ExchangeRates
	json.NewDecoder(w.Body).Decode(&response)

	zeroValue := 0.0
	if response.UsdBCV != zeroValue {
		t.Fatalf("Expected %v, got %v", zeroValue, response.UsdBCV)
	}

	if response.EurBCV != zeroValue {
		t.Fatalf("Expected %v, got %v", zeroValue, response.EurBCV)
	}

	if response.UsdtBinance != zeroValue {
		t.Fatalf("Expected %v, got %v", zeroValue, response.UsdtBinance)
	}

	if !response.LastUpdate.IsZero() {
		t.Fatalf("Expected LastUpdate to be zero")
	}
}

func TestRatesHandlerPOST(t *testing.T) {
	appState := state.NewState()
	appState.UpdateRates(state.ExchangeRates{UsdBCV: 480, EurBCV: 510, UsdtBinance: 530, LastUpdate: time.Now()})

	handler := NewRatesHandler(appState)

	req := httptest.NewRequest(http.MethodPost, "/rates", nil)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Expected JSON content-type")
	}

	var response state.ExchangeRates
	json.NewDecoder(w.Body).Decode(&response)

	if response.UsdBCV != appState.GetRates().UsdBCV {
		t.Fatalf("Expected %v, got %v", appState.GetRates().UsdBCV, response.UsdBCV)
	}

	if response.EurBCV != appState.GetRates().EurBCV {
		t.Fatalf("Expected %v, got %v", appState.GetRates().EurBCV, response.EurBCV)
	}

	if response.UsdtBinance != appState.GetRates().UsdtBinance {
		t.Fatalf("Expected %v, got %v", appState.GetRates().UsdtBinance, response.UsdtBinance)
	}

	if response.LastUpdate.IsZero() {
		t.Fatalf("Expected LastUpdate to be populated")
	}
}
