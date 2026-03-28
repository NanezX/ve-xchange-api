package main

import "time"

// TODO: Separate each "provider" with his own mutex (r-w) so we can lock individual provider instead of whole rates

type ExchageRates struct {
	BCV        float64   `json:"bcv"`
	Binance    float64   `json:"binance"`
	Paralelo   float64   `json:"paralelo"`
	LastUpdate time.Time `json:"last_update"`
}

type AppState struct {
	rates ExchageRates
}
