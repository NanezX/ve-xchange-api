package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"time"
)

type BinanceProvider struct {
	baseURL   string
	client    *http.Client
	appConfig *Config
}

func NewBinanceProvider(client *http.Client) *BinanceProvider {
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

func (t *TradeType) IsValid() bool {
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
func (p *BinanceProvider) generateBodyP2P(tradeType TradeType) (BodyRequestP2P, error) {
	if !tradeType.IsValid() {
		return BodyRequestP2P{}, errors.New("Invalid trade type")
	}

	return BodyRequestP2P{
		Asset:         "USDT",
		Fiat:          "VES",
		TradeType:     tradeType,
		PublisherType: "merchant",
		Page:          1,
		Rows:          20,
	}, nil
}

func (p *BinanceProvider) fetchPrices(tradeType TradeType) ([]float64, error) {
	// Get the basic body for the P2P Asset/Fiat
	bodyData, err := p.generateBodyP2P(tradeType)
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

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Leemos el cuerpo del error para ver qué dice Binance
		errorBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error de Binance: %s\n", string(errorBody))
		return nil, fmt.Errorf("error %d", resp.StatusCode)
	}

	var data JsonResponseP2P

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	if !data.Success {
		return nil, fmt.Errorf("Failed to get the [%s] P2P prices\n", tradeType)

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
		return nil, fmt.Errorf("Failed to get the [%s] P2P prices\n", tradeType)
	}

	return response, nil

}

func (p *BinanceProvider) GetPrices() (PriceResponse, error) {
	sellPrices, err := p.fetchPrices(TypeSell)
	if err != nil {
		return nil, err
	}

	buyPrices, err := p.fetchPrices(TypeBuy)
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

	return PriceResponse{"USDT_BINANCE": acc / float64(len(combined))}, nil

}

func (p *BinanceProvider) GetName() string {
	return "USDT"
}

func (p *BinanceProvider) UpdatePrice() {
	data, err := p.GetPrices()

	if err != nil {
		fmt.Printf("Error Binance P2P: %v", err)
		return
	}

	AppState.Lock()
	defer AppState.Unlock()

	AppState.Rates.UsdtBinance = data["USDT_BINANCE"]
	AppState.Rates.LastUpdate = time.Now()
}


