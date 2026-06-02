package worker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type MockProvider struct {
	mu         sync.RWMutex
	prices     rates.PriceResponse
	priceError error
	called     bool
}

func (p *MockProvider) isCalled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.called
}

func (p *MockProvider) GetPrices() (rates.PriceResponse, error) {
	p.mu.Lock()
	p.called = true
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

	var applyCalled atomic.Bool

	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Millisecond,
		Apply: func(rates.PriceResponse) {
			applyCalled.Store(true)
		},
	}

	StartPriceWorker([]ProviderJob{job})

	time.Sleep(50 * time.Millisecond)

	if !mockProvider.isCalled() {
		t.Fatalf("Expected Provider to be called")
	}

	if !applyCalled.Load() {
		t.Fatalf("Expected Apply function to be called")
	}
}

func TestWorkerProviderError(t *testing.T) {
	mockProvider := &MockProvider{
		priceError: errors.New("error to get prices"),
	}

	var applyCalled atomic.Bool

	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Millisecond,
		Apply: func(rates.PriceResponse) {
			applyCalled.Store(true)
		},
	}

	StartPriceWorker([]ProviderJob{job})

	time.Sleep(10 * time.Millisecond)

	if !mockProvider.isCalled() {
		t.Fatalf("Expected Provider GetPrices to be called")
	}

	if applyCalled.Load() {
		t.Fatalf("Apply functon should not be called on get price error")
	}
}

func TestWorkerProviderEmptyPrices(t *testing.T) {
	mockProvider := &MockProvider{
		prices: rates.PriceResponse{},
	}

	var applyCalled atomic.Bool

	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Millisecond,
		Apply: func(rates.PriceResponse) {
			applyCalled.Store(true)
		},
	}

	StartPriceWorker([]ProviderJob{job})

	time.Sleep(50 * time.Millisecond)

	if !mockProvider.isCalled() {
		t.Fatalf("Expected Provider to be called")
	}

	if !applyCalled.Load() {
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
	// TODO(tech-debt): This test relies on wall-clock timing. The worker
	// performs 1 immediate call + N ticks; with a 5ms ticker and a 25ms
	// sleep, scheduler jitter (especially under -race) can deliver anywhere
	// from ~4 to ~7 invocations. We assert a tolerant range to keep the
	// test non-flaky, but a tolerant range hides real regressions.
	// Proper fix: make the tick source injectable in StartPriceWorker (or
	// extract a pure processTick function) so tests can drive an exact
	// number of ticks synchronously. Tracked in ROADMAP.md § 6.4.
	if count < 3 || count > 8 {
		t.Fatalf("Expected between 3 and 8 apply calls, got %d", count)
	}

}
