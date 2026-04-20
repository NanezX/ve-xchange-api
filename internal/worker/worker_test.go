package worker

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type MockProvider struct {
	mu         sync.Mutex
	prices     rates.PriceResponse
	priceError error
	isCalled   bool
}

func (p *MockProvider) GetPrices() (rates.PriceResponse, error) {
	p.mu.Lock()
	p.isCalled = true
	p.mu.Unlock()
	if p.priceError != nil {
		return nil, p.priceError
	}
	return p.prices, nil
}

func (p *MockProvider) GetName() string {
	return "Mock"
}

func TestWorkerApplyData(t *testing.T) {
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

func TestWorkerProviderError(t *testing.T) {
	mockProvider := &MockProvider{
		priceError: errors.New("error to get prices"),
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

	time.Sleep(10 * time.Millisecond)

	if !mockProvider.isCalled {
		t.Fatalf("Expected Provider GetPrices to be called")
	}

	if applyCalled {
		t.Fatalf("Apply functon should not be called on get price error")
	}
}
