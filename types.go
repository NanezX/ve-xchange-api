package main

import (
	"github.com/nanezx/ve-xchange-api/internal/provider"
	"time"
)

// TODO: Separate each "provider" with his own mutex (r-w) so we can lock individual provider instead of whole rates

type ExchangeRates struct {
	UsdBCV      float64   `json:"usd_bcv"`
	EurBCV      float64   `json:"eur_bcv"`
	UsdtBinance float64   `json:"usdt_binance"`
	LastUpdate  time.Time `json:"last_update"`
}

type PriceProvider interface {
	GetPrices() (provider.PriceResponse, error)
	GetName() string
}
