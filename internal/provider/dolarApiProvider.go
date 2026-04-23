package provider

import (
	"fmt"
	"net/http"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type DolarApiProvider struct {
	baseURL string
	apiKey  string
	client  HTTPDoer
}

func NewDolarDolarApiProvider(client HTTPDoer) *DolarApiProvider {
	return &DolarApiProvider{
		baseURL: "https://ve.dolarapi.com/v1/cotizaciones",
		client:  client,
	}
}

type DolarApiCurrencyItem struct {
	Moneda             string  `json:"moneda"`
	Fuente             string  `json:"fuente"`
	Nombre             string  `json:"nombre"`
	Promedio           float64 `json:"promedio"`
	FechaActualizacion string  `json:"fechaActualizacion"`
}

func (p *DolarApiProvider) GetPrices() (rates.PriceResponse, error) {
	// Generate request
	req, err := http.NewRequest(http.MethodGet, p.baseURL, nil)
	if err != nil {
		return nil, err
	}

	// Fetch JSON
	data, err := fetchJson[[]DolarApiCurrencyItem](p.client, req)

	if err != nil {
		return nil, fmt.Errorf("DolarAPI prices - Error  %w", err)
	}

	// Init value
	resp := rates.PriceResponse{}

	// TODO: Should we tak care if slice length is different than two? The API returns 2 items (USD and EUR)
	for _, rate := range data {
		switch rate.Moneda {
		case "USD":
			resp["USD_BCV"] = rate.Promedio
		case "EUR":
			resp["EUR_BCV"] = rate.Promedio
		}
	}

	if len(resp) != 2 {
		return nil, fmt.Errorf("DolarAPI prices - invalid rates: USD=%.2f, EUR=%.2f (must be > 0)",
			resp["USD_BCV"], resp["EUR_BCV"])
	}

	if resp["USD_BCV"] <= 0 || resp["EUR_BCV"] <= 0 {
		return nil, fmt.Errorf("DolarAPI prices - invalid rates: USD=%.2f, EUR=%.2f (must be > 0)",
			resp["USD_BCV"], resp["EUR_BCV"])
	}

	return resp, nil
}

func (p *DolarApiProvider) GetName() string {
	return "DolarAPI"
}
