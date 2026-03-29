package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DolarVzlaProvider struct {
	baseURL string
	apiKey  string
}

func NewDolarVzlaProvider() *DolarVzlaProvider {
	return &DolarVzlaProvider{
		baseURL: "https://api.dolarvzla.com/public/bcv/exchange-rate",
		apiKey:  AppConfig.DolarVzlaApiKey,
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
	req, err := http.NewRequest(http.MethodGet, p.baseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-dolarvzla-key", p.apiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Leemos el cuerpo del error para ver qué dice Binance
		errorBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error de DolarVzla API: %s\n", string(errorBody))
		return nil, fmt.Errorf("error %d", resp.StatusCode)
	}

	var data JsonResponseDolarVzla
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return PriceResponse{
		"USD_BCV": data.Current.USD,
		"EUR_BCV": data.Current.EUR,
	}, nil
}

func (p *DolarVzlaProvider) GetName() string {
	return "DolarVzla"
}
