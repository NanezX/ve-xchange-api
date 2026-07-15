package provider

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestGetPriceDolazApiSuccess(t *testing.T) {
	usdPrice := 483.3379
	eurPrice := 567.0133

	// Data
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "USD", Promedio: usdPrice},
		{Moneda: "EUR", Promedio: eurPrice},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	fakeClient := &FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)

	prices, err := provider.GetPrices()

	if err != nil {
		t.Fatalf("Expected success, got %v", err)
	}

	if prices.Values["USD_BCV"] != usdPrice {
		t.Fatalf("Expected USD Price '%v', got '%v'", usdPrice, prices.Values["USD_BCV"])
	}

	if prices.Values["EUR_BCV"] != eurPrice {
		t.Fatalf("Expected EUR Price '%v', got '%v'", eurPrice, prices.Values["EUR_BCV"])
	}

}

func TestGetPriceDolazApiServerInternalError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 500}

	provider := NewDolarDolarApiProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiServerResponseError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 404}

	provider := NewDolarDolarApiProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiNetworkError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{Error: errors.New("connection timeout")}

	provider := NewDolarDolarApiProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiInvalidJSON(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200, Body: "not json"}

	provider := NewDolarDolarApiProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiEmptyResponse(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiMissingEUR(t *testing.T) {
	usdPrice := 483.3379

	// Data with missing EUR
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "USD", Promedio: usdPrice},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	fakeClient := &FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetPriceDolazApiMissingUSD(t *testing.T) {
	eurPrice := 483.3379

	// Data with missing USD
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "EUR", Promedio: eurPrice},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	fakeClient := &FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetPriceDolazApiWrongType(t *testing.T) {
	jsonBody := `{"current": {"usd": 50.5, "eur": "eur price"}}`

	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetPriceDolazApiUSDZero(t *testing.T) {
	usdPrice := 0.0
	eurPrice := 0.0

	// Data
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "USD", Promedio: usdPrice},
		{Moneda: "EUR", Promedio: eurPrice},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	fakeClient := &FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

// --- Boundary / edge-value tests ---

func TestGetPriceDolazApiNegativeUSD(t *testing.T) {
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "USD", Promedio: -100.0},
		{Moneda: "EUR", Promedio: 50.0},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	provider := NewDolarDolarApiProvider(&FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200})
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()
	if err == nil {
		t.Fatal("expected error for negative USD price, got nil")
	}
}

func TestGetPriceDolazApiNegativeEUR(t *testing.T) {
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "USD", Promedio: 50.0},
		{Moneda: "EUR", Promedio: -1.0},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	provider := NewDolarDolarApiProvider(&FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200})
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()
	if err == nil {
		t.Fatal("expected error for negative EUR price, got nil")
	}
}

func TestGetPriceDolazApiExtremelyLargeValues(t *testing.T) {
	// 1e308 is a valid float64 but financially nonsensical — provider currently
	// accepts it (only validates > 0). This test documents the current behavior.
	fakeData := []DolarApiCurrencyItem{
		{Moneda: "USD", Promedio: 1e308},
		{Moneda: "EUR", Promedio: 1e308},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	provider := NewDolarDolarApiProvider(&FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200})

	prices, err := provider.GetPrices()
	if err != nil {
		t.Fatalf("unexpected error for large values: %v", err)
	}
	if prices.Values["USD_BCV"] != 1e308 {
		t.Fatalf("expected 1e308, got %v", prices.Values["USD_BCV"])
	}
}

func TestGetPriceDolazApiNullPromedioTreatedAsZero(t *testing.T) {
	// When "promedio" is null in JSON, Go decodes it as 0.0.
	// The provider's > 0 check must catch it.
	body := `[{"moneda":"USD","promedio":null},{"moneda":"EUR","promedio":50.0}]`

	provider := NewDolarDolarApiProvider(&FakeHTTPDoer{Body: body, StatusCode: 200})
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()
	if err == nil {
		t.Fatal("expected error when USD promedio is null (decoded as 0), got nil")
	}
}
