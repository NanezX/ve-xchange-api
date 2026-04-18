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

// func TestGetPriceBinanceServerInternalError(t *testing.T) {
// 	fakeClient := &FakeHTTPDoer{StatusCode: 500}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got %v", err)
// 	}
// }

// func TestGetPriceBinanceServerResponseError(t *testing.T) {
// 	fakeClient := &FakeHTTPDoer{StatusCode: 404}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got %v", err)
// 	}
// }

// func TestGetPriceBinanceNetworkError(t *testing.T) {
// 	fakeClient := &FakeHTTPDoer{Error: errors.New("connection timeout")}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got %v", err)
// 	}
// }

// func TestGetPriceBinanceInvalidJSON(t *testing.T) {
// 	fakeClient := &FakeHTTPDoer{StatusCode: 200, Body: "not json"}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got %v", err)
// 	}
// }

// func TestGetPriceBinanceEmptyResponse(t *testing.T) {
// 	fakeClient := &FakeHTTPDoer{StatusCode: 200}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got %v", err)
// 	}
// }

// func TestGetPriceBinanceMissingEUR(t *testing.T) {
// 	// Body with missing EUR
// 	jsonBody := `{"current": {"usd": 50.5}}`

// 	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got nil")
// 	}
// }

// func TestGetPriceBinanceWrongType(t *testing.T) {
// 	jsonBody := `{"current": {"usd": 50.5, "eur": "eur price"}}`

// 	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got nil")
// 	}
// }

// func TestGetPriceBinanceUSDZero(t *testing.T) {
// 	jsonBody := `{"current": {"usd": 0, "eur": 2.1}}`

// 	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got nil")
// 	}
// }

// func TestGetPriceBinanceEURZero(t *testing.T) {
// 	jsonBody := `{"current": {"usd": 2.1, "eur": 0}}`

// 	fakeClient := &FakeHTTPDoer{Body: jsonBody, StatusCode: 200}

// 	provider := NewDolarVzlaProvider(fakeClient, "")

// 	_, err := provider.GetPrices()

// 	if err == nil {
// 		t.Fatalf("Expected error, got nil")
// 	}
// }
