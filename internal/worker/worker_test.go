package worker

import (
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type MockProvider struct {
	prices   rates.PriceResponse
	isCalled bool
}

func (p *MockProvider) GetPrices() (rates.PriceResponse, error) {
	p.isCalled = true
	return p.prices, nil
}

func (p *MockProvider) GetName() string {
	return "Mock"
}

func TestWorkerApplyData(t *testing.T) {
	//
	mockProvider := &MockProvider{
		prices: rates.PriceResponse{"USD": 543.21},
	}

	applyCalled := false

	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Millisecond,
		Apply: func(rates.PriceResponse) {
			applyCalled = true
		},
	}

	StartPriceWorker([]ProviderJob{job})

	time.Sleep(50 * time.Millisecond)

	if !mockProvider.isCalled {
		t.Fatalf("Expected Provider to be called")
	}

	if !applyCalled {
		t.Fatalf("Expected Apply function to be called")
	}

}
