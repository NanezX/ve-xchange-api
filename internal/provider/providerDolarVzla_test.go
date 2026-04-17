package provider

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestGetPriceDolazVzlaSuccess(t *testing.T) {
	usdPrice := 50.5
	eurPrice := 56.1

	// Data
	fakeData := JsonResponseDolarVzla{
		Current: DataDolarVzlaBCV{
			USD:  usdPrice,
			EUR:  eurPrice,
			Date: "2026-03-28 16:00:20.091Z",
		},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	fakeClient := &FakeHTTPDoer{Body: string(jsonBytes), StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

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

func TestGetPriceDolazVzlaServerInternalError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 500}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazVzlaServerResponseError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 404}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazVzlaNetworkError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{Error: errors.New("connection timeout")}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazVzlaInvalidJSON(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200, Body: "not json"}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazVzlaEmptyResponse(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceDolazVzlaMissingEUR(t *testing.T) {
	// Body with missing EUR
	jsonBody := `{"current": {"usd": 50.5}}`

	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetPriceDolazVzlaWrongType(t *testing.T) {
	jsonBody := `{"current": {"usd": 50.5, "eur": "eur price"}}`

	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetPriceDolazVzlaUSDZero(t *testing.T) {
	jsonBody := `{"current": {"usd": 0, "eur": 2.1}}`

	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetPriceDolazVzlaEURZero(t *testing.T) {
	jsonBody := `{"current": {"usd": 2.1, "eur": 0}}`

	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}
