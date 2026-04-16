package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type FakeHTTPDoer struct {
	jsonBody string
}

func NewFakeClient(jsonBody string) *FakeHTTPDoer {
	return &FakeHTTPDoer{jsonBody}
}

func (f *FakeHTTPDoer) Do(*http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(f.jsonBody)),
	}

	return resp, nil
}

func TestGetPriceSuccess(t *testing.T) {
	usdPrice := 50.5
	eurPrice := 2.1

	fakeData := JsonResponseDolarVzla{
		Current: DataDolarVzlaBCV{
			USD:  usdPrice,
			EUR:  eurPrice,
			Date: "2026-03-28 16:00:20.091Z",
		},
	}
	jsonBytes, _ := json.Marshal(fakeData)

	fakeClient := NewFakeClient(string(jsonBytes))

	provider := NewDolarVzlaProvider(fakeClient, "")

	prices, err := provider.GetPrices()

	if err != nil {
		t.Fatalf("Expected succes, got %v", err)
	}


	if prices["USD_BCV"] != usdPrice {
		t.Fatalf("Expected USD Price '%v', got '%v'", usdPrice, prices["USD_BCV"])
	}

	if prices["EUR_BCV"] != eurPrice {
		t.Fatalf("Expected EUR Price '%v', got '%v'", eurPrice, prices["EUR_BCV"])
	}

}
