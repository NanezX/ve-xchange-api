package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type DolarVzlaBCVBase struct {
	USD float64 `json:"usd"`
	EUR float64 `json:"eur"`
	// FIXME: Using string by now, but we should implement an
	// UnmarshalJSON a custom type for us, the api return just
	// a plain string with `2026-03-28 16:00:20.091Z` format.
	//Date time.Time `json:"date"`
	Date string `json:"date"`
}

type ChangePerceantage struct {
	USD float64 `json:"usd"`
	EUR float64 `json:"eur"`
}

type DolarVzlaBCVResponse struct {
	Current          DolarVzlaBCVBase  `json:"current"`
	Previous         DolarVzlaBCVBase  `json:"previous"`
	ChangePercentage ChangePerceantage `json:"changePercentage"`
}

const dolarVzlaUrl = "https://api.dolarvzla.com/public/"

// Fetch from `DolarVzla` API
func fetchDolarVzlaBcv() (DolarVzlaBCVResponse, error) {
	// os.Getenv("DOLAR_VZLA_API_KEY")
	url := dolarVzlaUrl + "bcv/exchange-rate"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return DolarVzlaBCVResponse{}, err
	}

	req.Header.Set("x-dolarvzla-key", AppConfig.DolarVzlaApiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		return DolarVzlaBCVResponse{}, err
	}

	defer resp.Body.Close()

	var data DolarVzlaBCVResponse

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&data)
	if err != nil {
		return DolarVzlaBCVResponse{}, err
	}

	return data, nil

}
