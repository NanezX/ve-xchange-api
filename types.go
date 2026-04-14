package main

import (
	"github.com/nanezx/ve-xchange-api/internal/provider"
)

type PriceProvider interface {
	GetPrices() (provider.PriceResponse, error)
	GetName() string
}
