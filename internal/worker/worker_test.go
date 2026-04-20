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

type CountingProvider struct {
	mu             sync.Mutex
	getPricesCalls int
	applyCallCount int
	prices         rates.PriceResponse
}

func (p *CountingProvider) GetPrices() (rates.PriceResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.getPricesCalls++
	return rates.PriceResponse{}, nil
}
func (p *CountingProvider) GetName() string {
	return "MockCounting"
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

func TestWorkerProviderEmptyPrices(t *testing.T) {
	mockProvider := &MockProvider{
		prices: rates.PriceResponse{},
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

func TestWorkerTicksExecution(t *testing.T) {
	mockProvider := &CountingProvider{
		prices: rates.PriceResponse{"USD": 543.21},
	}

	job := ProviderJob{
		Provider: mockProvider,
		Every:    5 * time.Millisecond,
		Apply: func(rates.PriceResponse) {
			mockProvider.mu.Lock()
			mockProvider.applyCallCount++
			mockProvider.mu.Unlock()
		},
	}

	StartPriceWorker([]ProviderJob{job})

	time.Sleep(25 * time.Millisecond)

	mockProvider.mu.Lock()
	count := mockProvider.applyCallCount
	mockProvider.mu.Unlock()
	if count != 5 {
		t.Fatalf("Expected 5, got %d", count)
	}

}
