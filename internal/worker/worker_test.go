package worker

import (
	"context"
	"errors"
	"sync"
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

func TestWorkerApplyData(t *testing.T) {
	mockProvider := &MockProvider{
		prices: rates.PriceResponse{"USD": 543.21},
	}

	calls := make(chan struct{}, 1)
	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Hour,
		Apply:    func(rates.PriceResponse) { calls <- struct{}{} },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatalf("Expected Apply to be called")
	}

	cancel()
	wg.Wait()
}

func TestWorkerProviderError(t *testing.T) {
	mockProvider := &MockProvider{
		priceError: errors.New("error to get prices"),
	}

	applyCalled := make(chan struct{}, 1)
	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Hour,
		Apply:    func(rates.PriceResponse) { applyCalled <- struct{}{} },
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	// Give the worker time to run the initial fetch, then verify Apply was not called.
	time.Sleep(20 * time.Millisecond)

	select {
	case <-applyCalled:
		t.Fatalf("Apply should not be called on provider error")
	default:
	}

	cancel()
	wg.Wait()
}

func TestWorkerProviderEmptyPrices(t *testing.T) {
	mockProvider := &MockProvider{
		prices: rates.PriceResponse{},
	}

	calls := make(chan struct{}, 1)
	job := ProviderJob{
		Provider: mockProvider,
		Every:    1 * time.Hour,
		Apply:    func(rates.PriceResponse) { calls <- struct{}{} },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatalf("Expected Apply to be called even with empty prices")
	}

	cancel()
	wg.Wait()
}

func TestWorkerTicksExecution(t *testing.T) {
	const wantApplies = 5

	// Buffered so the worker goroutine never blocks on Apply.
	calls := make(chan struct{}, wantApplies*2)

	job := ProviderJob{
		Provider: &MockProvider{prices: rates.PriceResponse{"USD": 543.21}},
		Every:    time.Millisecond,
		Apply:    func(rates.PriceResponse) { calls <- struct{}{} },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	// Collect exactly wantApplies signals. Each receive is event-driven —
	// no sleep, no tolerance range.
	for i := 0; i < wantApplies; i++ {
		select {
		case <-calls:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for apply call %d/%d", i+1, wantApplies)
		}
	}

	cancel()
	wg.Wait()
}

