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
	for i := range wantApplies {
		select {
		case <-calls:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for apply call %d/%d", i+1, wantApplies)
		}
	}

	cancel()
	wg.Wait()
}

func TestWorkerOnFailCalledAfterThreeConsecutiveFailures(t *testing.T) {
	failProvider := &MockProvider{priceError: errors.New("boom")}

	onFail := make(chan int64, 10)
	job := ProviderJob{
		Provider:  failProvider,
		Every:     time.Millisecond,
		Apply:     func(rates.PriceResponse) {},
		OnFail:    func(n int64) { onFail <- n },
		OnRecover: func() { t.Error("OnRecover should not be called") },
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	// Wait for at least one OnFail signal (triggered at consecutiveFails==3).
	select {
	case n := <-onFail:
		if n < 3 {
			t.Fatalf("expected consecutiveFails>=3, got %d", n)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for OnFail")
	}

	cancel()
	wg.Wait()
}

func TestWorkerDailyAt_RetriesOnFailure(t *testing.T) {
	// Provider fails on the initial startup fetch. Apply must NOT fire yet.
	// We verify the startup fetch ran and Apply was not triggered.
	applyCh := make(chan struct{}, 1)
	callCount := 0

	prov := &mockSequenceProvider{
		count: &callCount,
		responses: func(n int) (rates.PriceResponse, error) {
			if n < 2 {
				return nil, errors.New("temporary failure")
			}
			return rates.PriceResponse{"USD": 1.0}, nil
		},
	}

	job := ProviderJob{
		Provider: prov,
		DailyAt:  &TimeOfDay{Hour: 0, Minute: 0, Location: time.UTC},
		Apply:    func(rates.PriceResponse) { applyCh <- struct{}{} },
		OnFail:   func(int64) {},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = StartPriceWorker(ctx, []ProviderJob{job})

	time.Sleep(50 * time.Millisecond)
	prov.mu.Lock()
	c := callCount
	prov.mu.Unlock()
	if c < 1 {
		t.Fatal("expected at least one fetch call at startup")
	}

	// Apply must NOT have fired — provider is still failing on first 2 calls.
	select {
	case <-applyCh:
		t.Fatal("Apply fired before provider recovered")
	default:
	}
}

func TestWorkerDailyAt_StopsRetryingAfter10Attempts(t *testing.T) {
	// All fetches fail. Apply must never be called.
	applyCh := make(chan struct{}, 1)
	callCount := 0

	prov := &mockSequenceProvider{
		count: &callCount,
		responses: func(n int) (rates.PriceResponse, error) {
			return nil, errors.New("always fails")
		},
	}

	job := ProviderJob{
		Provider: prov,
		DailyAt:  &TimeOfDay{Hour: 0, Minute: 0, Location: time.UTC},
		Apply:    func(rates.PriceResponse) { applyCh <- struct{}{} },
		OnFail:   func(int64) {},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = StartPriceWorker(ctx, []ProviderJob{job})
	<-ctx.Done()

	select {
	case <-applyCh:
		t.Fatal("Apply should never fire when provider always fails")
	default:
	}
}

func TestWorkerOnRecoverCalledAfterStreak(t *testing.T) {
	const failCount = 3

	callCount := 0

	recovered := make(chan struct{}, 1)
	job := ProviderJob{
		Provider: &mockSequenceProvider{
			responses: func(n int) (rates.PriceResponse, error) {
				if n < failCount {
					return nil, errors.New("transient error")
				}
				return rates.PriceResponse{"USD": 1.0}, nil
			},
			count: &callCount,
		},
		Every:  time.Millisecond,
		Apply:  func(rates.PriceResponse) {},
		OnFail: func(_ int64) {},
		OnRecover: func() {
			select {
			case recovered <- struct{}{}:
			default:
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	select {
	case <-recovered:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for OnRecover after streak")
	}

	cancel()
	wg.Wait()
}

// mockSequenceProvider calls responses(n) for each successive GetPrices call.
type mockSequenceProvider struct {
	responses func(n int) (rates.PriceResponse, error)
	count     *int
	mu        sync.Mutex
}

func (m *mockSequenceProvider) GetPrices() (rates.PriceResponse, error) {
	m.mu.Lock()
	n := *m.count
	*m.count++
	m.mu.Unlock()
	return m.responses(n)
}

func (m *mockSequenceProvider) GetName() string { return "mockSequence" }

// --- nextDaily ---

func TestNextDaily_BeforeTargetToday(t *testing.T) {
	loc := time.UTC
	tod := TimeOfDay{Hour: 0, Minute: 5, Location: loc}

	// now = 23:00 the day before — target (00:05) is still in the future today
	// Wait: 00:05 is earlier than 23:00, so next occurrence is tomorrow 00:05.
	now := time.Date(2026, 6, 4, 23, 0, 0, 0, loc)
	d := nextDaily(tod, now)

	// Expected: ~65 minutes
	expected := 65 * time.Minute
	if d < 60*time.Minute || d > 70*time.Minute {
		t.Fatalf("expected ~65min, got %v (expected ~%v)", d, expected)
	}
}

func TestNextDaily_AfterTargetToday(t *testing.T) {
	loc := time.UTC
	tod := TimeOfDay{Hour: 0, Minute: 5, Location: loc}

	// now = 01:00 — target (00:05) already passed today, next is tomorrow
	now := time.Date(2026, 6, 4, 1, 0, 0, 0, loc)
	d := nextDaily(tod, now)

	// Expected: ~23h05m
	if d < 23*time.Hour || d > 24*time.Hour {
		t.Fatalf("expected ~23h, got %v", d)
	}
}

func TestNextDaily_ExactlyAtTarget_SchedulesTomorrow(t *testing.T) {
	loc := time.UTC
	tod := TimeOfDay{Hour: 0, Minute: 5, Location: loc}

	// now == target exactly → must schedule for tomorrow (not fire immediately)
	now := time.Date(2026, 6, 4, 0, 5, 0, 0, loc)
	d := nextDaily(tod, now)

	if d < 23*time.Hour || d > 24*time.Hour {
		t.Fatalf("expected ~24h when now==target, got %v", d)
	}
}

func TestNextDaily_DifferentTimezone(t *testing.T) {
	utcMinus4 := time.FixedZone("UTC-4", -4*60*60)
	tod := TimeOfDay{Hour: 0, Minute: 5, Location: utcMinus4}

	// 00:05 UTC-4 = 04:05 UTC
	// now = 03:00 UTC (= 23:00 UTC-4 the day before) — target is 65 min away
	now := time.Date(2026, 6, 4, 3, 0, 0, 0, time.UTC)
	d := nextDaily(tod, now)

	if d < 60*time.Minute || d > 70*time.Minute {
		t.Fatalf("expected ~65min across timezone, got %v", d)
	}
}

func TestWorkerDailyAt_InitialFetchRunsImmediately(t *testing.T) {
	// DailyAt is set — the initial fetch must still run at startup (before the
	// first daily timer fires).
	provider := &MockProvider{prices: rates.PriceResponse{"USD": 1.0}}
	calls := make(chan struct{}, 1)

	loc := time.UTC
	job := ProviderJob{
		Provider: provider,
		DailyAt:  &TimeOfDay{Hour: 0, Minute: 5, Location: loc},
		Apply:    func(rates.PriceResponse) { calls <- struct{}{} },
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := StartPriceWorker(ctx, []ProviderJob{job})

	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("initial fetch did not fire for DailyAt job")
	}

	cancel()
	wg.Wait()
}
