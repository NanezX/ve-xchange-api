package provider

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type FakeHTTPDoer struct {
	StatusCode int
	Body       string
	Error      error
}

func (f *FakeHTTPDoer) Do(*http.Request) (*http.Response, error) {
	if f.Error != nil {
		return nil, f.Error
	}

	resp := &http.Response{
		StatusCode: f.StatusCode,
		Body:       io.NopCloser(strings.NewReader(f.Body)),
	}

	return resp, nil
}

func TestGetPriceSuccess(t *testing.T) {
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

func TestGetPriceServerInternalError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 500}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceServerResponseError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 404}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceNetworkError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{Error: errors.New("connection timeout")}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceInvalidJSON(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200, Body: "not json"}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceEmptyResponse(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200}

	provider := NewDolarVzlaProvider(fakeClient, "")

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}
