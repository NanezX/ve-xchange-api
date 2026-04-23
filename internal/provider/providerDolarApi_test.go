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

	if prices["USD_BCV"] != usdPrice {
		t.Fatalf("Expected USD Price '%v', got '%v'", usdPrice, prices["USD_BCV"])
	}

	if prices["EUR_BCV"] != eurPrice {
		t.Fatalf("Expected EUR Price '%v', got '%v'", eurPrice, prices["EUR_BCV"])
	}

}

func TestGetPriceDolazApiServerInternalError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 500}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiServerResponseError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 404}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiNetworkError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{Error: errors.New("connection timeout")}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiInvalidJSON(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200, Body: "not json"}

	provider := NewDolarDolarApiProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazApiEmptyResponse(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200}

	provider := NewDolarDolarApiProvider(fakeClient)

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
