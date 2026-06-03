package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
)

func float64ToStr(val float64) string {
	return strconv.FormatFloat(val, 'f', -1, 64)
}

func genMockData(t TradeType) []DataP2P {
	if t == TypeBuy {
		mockPrices := []float64{505.0, 514.0, 523.0, 532.0, 541.0}
		return []DataP2P{
			{Adv: DataAdv{Price: float64ToStr(mockPrices[0])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[1])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[2])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[3])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[4])}},
		}
	} else {
		mockPrices := []float64{515.0, 525.5, 513.8, 545.12, 543.0}
		return []DataP2P{
			{Adv: DataAdv{Price: float64ToStr(mockPrices[0])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[1])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[2])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[3])}},
			{Adv: DataAdv{Price: float64ToStr(mockPrices[4])}},
		}
	}
}

func TestGetPriceBinanceSuccess(t *testing.T) {
	sellData := JsonResponseP2P{Success: true, Data: genMockData(TypeSell)}
	buyData := JsonResponseP2P{Success: true, Data: genMockData(TypeBuy)}

	// To verify provider calls
	callCount := 0
	sellCalled := false
	buyCalled := false

	fakeClient := &FakeHTTPDoer{
		StatusCode: 200,
		DoFunc: func(req *http.Request) (string, error) {
			callCount++

			var body BodyRequestP2P
			json.NewDecoder(req.Body).Decode(&body)

			switch body.TradeType {
			case "SELL":
				bytes, _ := json.Marshal(sellData)
				if !sellCalled {
					sellCalled = true
				}
				return string(bytes), nil
			case "BUY":
				bytes, _ := json.Marshal(buyData)
				if !buyCalled {
					buyCalled = true
				}
				return string(bytes), nil
			}

			return "", errors.New("unknown trade type")
		},
	}

	provider := NewBinanceProvider(fakeClient)

	prices, err := provider.GetPrices()

	if err != nil {
		t.Fatalf("Expected success, got %v", err)
	}

	if !sellCalled {
		t.Fatalf("Expected to also use SELL P2P Data, but never called")
	}
	if !buyCalled {
		t.Fatalf("Expected to also use BUY P2P Data, but never called")
	}

	if callCount != 10 {
		t.Fatalf("Expected 10 calls (5 SELL + 5 BUY), got %d", callCount)
	}

	if prices["USDT_BINANCE"] != 525.742 {
		t.Fatalf("Expected val=525.742, got %v", prices["USDT_BINANCE"])

	}

}

func TestGetPriceBinanceServerInternalError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 500}

	provider := NewBinanceProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceBinanceServerResponseError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 404}

	provider := NewBinanceProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceBinanceNetworkError(t *testing.T) {
	fakeClient := &FakeHTTPDoer{Error: errors.New("connection timeout")}

	provider := NewBinanceProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceBinanceInvalidJSON(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200, Body: "not json"}

	provider := NewBinanceProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceBinanceEmptyResponse(t *testing.T) {
	fakeClient := &FakeHTTPDoer{StatusCode: 200}

	provider := NewBinanceProvider(fakeClient)
	provider.retryBaseDelay = 0

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}
}

func TestGetPriceBinanceNoPricesFound(t *testing.T) {
	emptyData := []DataP2P{}
	sellData := JsonResponseP2P{Success: true, Data: emptyData}
	buyData := JsonResponseP2P{Success: true, Data: emptyData}

	fakeClient := &FakeHTTPDoer{
		StatusCode: 200,
		DoFunc: func(req *http.Request) (string, error) {

			var body BodyRequestP2P
			json.NewDecoder(req.Body).Decode(&body)

			switch body.TradeType {
			case "SELL":
				bytes, _ := json.Marshal(sellData)

				return string(bytes), nil
			case "BUY":
				bytes, _ := json.Marshal(buyData)
				return string(bytes), nil
			}

			return "", errors.New("unknown trade type")
		},
	}

	provider := NewBinanceProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected error no prices found, got nil")
	}
}

func TestGetPriceBinanceNoSuccess(t *testing.T) {
	sellData := JsonResponseP2P{Success: false, Data: genMockData(TypeSell)}
	buyData := JsonResponseP2P{Success: true, Data: genMockData(TypeBuy)}

	fakeClient := &FakeHTTPDoer{
		StatusCode: 200,
		DoFunc: func(req *http.Request) (string, error) {

			var body BodyRequestP2P
			json.NewDecoder(req.Body).Decode(&body)

			switch body.TradeType {
			case "SELL":
				bytes, _ := json.Marshal(sellData)

				return string(bytes), nil
			case "BUY":
				bytes, _ := json.Marshal(buyData)
				return string(bytes), nil
			}

			return "", errors.New("unknown trade type")
		},
	}

	provider := NewBinanceProvider(fakeClient)

	_, err := provider.GetPrices()

	if err == nil {
		t.Fatalf("Expected failure, got %v", err)
	}

}

// --- Boundary / edge-value tests ---

// makeP2PClient returns a FakeHTTPDoer that always responds with the given
// prices (same data for every page of both SELL and BUY).
func makeP2PClient(prices []float64) *FakeHTTPDoer {
	data := make([]DataP2P, len(prices))
	for i, p := range prices {
		data[i] = DataP2P{Adv: DataAdv{Price: float64ToStr(p)}}
	}
	resp := JsonResponseP2P{Success: true, Data: data}
	respBytes, _ := json.Marshal(resp)
	body := string(respBytes)
	return &FakeHTTPDoer{
		StatusCode: 200,
		DoFunc:     func(_ *http.Request) (string, error) { return body, nil },
	}
}

func TestGetPriceBinanceNegativePricesFiltered(t *testing.T) {
	// Mix of valid and negative prices — negatives must be filtered out.
	p := NewBinanceProvider(makeP2PClient([]float64{500.0, -1.0, 600.0}))
	p.retryBaseDelay = 0

	prices, err := p.GetPrices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	avg := prices["USDT_BINANCE"]
	// Only 500 and 600 should count: avg = (500*10 + 600*10) / 20 = 550.
	if avg != 550.0 {
		t.Fatalf("expected avg=550.0 (negatives filtered), got %v", avg)
	}
}

func TestGetPriceBinanceAllNegativePricesReturnsError(t *testing.T) {
	p := NewBinanceProvider(makeP2PClient([]float64{-100.0, -200.0}))
	p.retryBaseDelay = 0

	_, err := p.GetPrices()
	if err == nil {
		t.Fatal("expected error when all prices are negative, got nil")
	}
}

func TestGetPriceBinanceZeroPriceFiltered(t *testing.T) {
	// Zero prices are non-positive and must be filtered.
	p := NewBinanceProvider(makeP2PClient([]float64{0.0, 500.0}))
	p.retryBaseDelay = 0

	prices, err := p.GetPrices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	avg := prices["USDT_BINANCE"]
	// Only 500 should count across 10 pages * 2 types = 20 entries,
	// but 20 zero entries are filtered, leaving 20 valid 500s → avg = 500.
	if avg != 500.0 {
		t.Fatalf("expected avg=500.0 (zeros filtered), got %v", avg)
	}
}
