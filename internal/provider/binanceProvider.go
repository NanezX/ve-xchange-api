package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type BinanceProvider struct {
	baseURL string
	client  HTTPDoer
}

func NewBinanceProvider(client HTTPDoer) *BinanceProvider {
	return &BinanceProvider{
		baseURL: "https://p2p.binance.com/bapi/c2c/v2/friendly/c2c/adv/search",
		client:  client,
	}
}

type TradeType string

const (
	TypeBuy  TradeType = "BUY"
	TypeSell TradeType = "SELL"
)

func (t *TradeType) isValid() bool {
	if *t == TypeBuy || *t == TypeSell {
		return true
	}

	return false
}

type BodyRequestP2P struct {
	Asset         string    `json:"asset"`
	Fiat          string    `json:"fiat"`
	TradeType     TradeType `json:"tradeType"`
	PublisherType string    `json:"publisherType"`
	Page          uint      `json:"page"`
	Rows          uint      `json:"rows"`
	PayTypes      []string  `json:"payTypes"`
}

// Advertisement
type DataAdv struct {
	TradeType TradeType `json:"tradeType"`
	Asset     string    `json:"asset"`
	FiatUnit  string    `json:"fiatUnit"`
	Price     string    `json:"price"`
}

type DataP2P struct {
	Adv DataAdv `json:"adv"`
}

type JsonResponseP2P struct {
	Total   uint      `json:"total"`
	Success bool      `json:"success"`
	Data    []DataP2P `json:"data"`
}

// Generate only USDT-VES
func (p *BinanceProvider) generateBodyP2P(tradeType TradeType, page uint) (BodyRequestP2P, error) {
	if !tradeType.isValid() {
		return BodyRequestP2P{}, errors.New("Invalid trade type")
	}

	return BodyRequestP2P{
		Asset:         "USDT",
		Fiat:          "VES",
		TradeType:     tradeType,
		PublisherType: "merchant",
		Page:          page,
		Rows:          20,
	}, nil
}

func (p *BinanceProvider) getOrders(tradeType TradeType, page uint) ([]float64, error) {
	// Get the basic body for the P2P Asset/Fiat page 1
	bodyData, err := p.generateBodyP2P(tradeType, page)
	if err != nil {
		return nil, err
	}

	// Convert from struct to json
	jsonBody, err := json.Marshal(bodyData)
	if err != nil {
		return nil, err
	}

	bufferBody := bytes.NewBuffer(jsonBody)

	// Generate the request
	req, err := http.NewRequest(http.MethodPost, p.baseURL, bufferBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Fetch JSON
	data, err := fetchJson[JsonResponseP2P](p.client, req)

	if err != nil {
		return nil, fmt.Errorf("P2P [%s] prices - Error %w", tradeType, err)
	}

	if !data.Success {
		return nil, fmt.Errorf("P2P [%s] prices - Error No success Data: %v", tradeType, data)

	}

	var response []float64

	for _, v := range data.Data {
		val, err := strconv.ParseFloat(v.Adv.Price, 64)

		if err != nil {
			continue
		}

		response = append(response, val)
	}

	if len(response) == 0 {
		return nil, fmt.Errorf("P2P [%s] prices - No prices found", tradeType)
	}

	return response, nil

}

func (p *BinanceProvider) getAllOrders(tradeType TradeType) ([]float64, error) {
	collectedPrices := []float64{}

	// FIXME: Use goroutines to improve speed here
	for page := uint(1); page <= 5; page++ {
		// Get the basic body for the P2P Asset/Fiat page
		prices, err := p.getOrders(tradeType, page)

		if err != nil {
			return nil, err
		}

		collectedPrices = slices.Concat(collectedPrices, prices)
	}

	return collectedPrices, nil
}

func (p *BinanceProvider) GetPrices() (rates.PriceResponse, error) {
	sellPrices, err := p.getAllOrders(TypeSell)
	if err != nil {
		return nil, err
	}

	buyPrices, err := p.getAllOrders(TypeBuy)
	if err != nil {
		return nil, err
	}

	combined := slices.Concat(sellPrices, buyPrices)

	if len(combined) == 0 {
		return nil, errors.New("No prices found")
	}

	var acc float64

	for _, val := range combined {

		acc += val
	}

	return rates.PriceResponse{"USDT_BINANCE": acc / float64(len(combined))}, nil

}

func (p *BinanceProvider) GetName() string {
	return "USDT"
}
