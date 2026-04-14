package provider

import (
	"fmt"
	"net/http"
)

type DolarVzlaProvider struct {
	baseURL string
	apiKey  string
	client  HTTPDoer
}

func NewDolarVzlaProvider(client HTTPDoer, apiKey string) *DolarVzlaProvider {
	return &DolarVzlaProvider{
		baseURL: "https://api.dolarvzla.com/public/bcv/exchange-rate",
		apiKey:  apiKey,
		client:  client,
	}
}

type DataDolarVzlaBCV struct {
	USD float64 `json:"usd"`
	EUR float64 `json:"eur"`
	// FIXME: Using string by now, but we should implement an
	// UnmarshalJSON a custom type for us, the api return just
	// a plain string with `2026-03-28 16:00:20.091Z` format.
	//Date time.Time `json:"date"`
	Date string `json:"date"`
}

type JsonResponseDolarVzla struct {
	Current DataDolarVzlaBCV `json:"current"`
}

func (p *DolarVzlaProvider) GetPrices() (PriceResponse, error) {
	// Generate request
	req, err := http.NewRequest(http.MethodGet, p.baseURL, nil)
	if err != nil {
		return nil, err
	}

	// Set api key
	req.Header.Set("x-dolarvzla-key", p.apiKey)

	// Fetch JSON
	data, err := fetchJson[JsonResponseDolarVzla](p.client, req)

	if err != nil {
		return nil, fmt.Errorf("DolarVzla prices - Error  %w", err)
	}

	return PriceResponse{
		"USD_BCV": data.Current.USD,
		"EUR_BCV": data.Current.EUR,
	}, nil
}

func (p *DolarVzlaProvider) GetName() string {
	return "DolarVzla"
}
