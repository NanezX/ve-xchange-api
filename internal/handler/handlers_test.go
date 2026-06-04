package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/api"
	"github.com/nanezx/ve-xchange-api/internal/db"
	"github.com/nanezx/ve-xchange-api/internal/state"
)

// withFixedNow returns a Server whose internal clock is pinned for
// deterministic age/staleness assertions.
func withFixedNow(s *state.State, now time.Time) Server {
	srv := NewServer(s)
	srv.now = func() time.Time { return now }
	return srv
}

func mountMux(t *testing.T, srv Server) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	api.HandlerFromMux(srv, mux)
	return mux
}

func TestGetRates_AllFresh(t *testing.T) {
	now := time.Now()
	ago := now.Add(-30 * time.Second)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:      state.RateData{Value: 480.5, LastUpdated: &ago},
		EurBcv:      state.RateData{Value: 520.1, LastUpdated: &ago},
		UsdtBinance: state.RateData{Value: 535.2, LastUpdated: &ago},
	})

	srv := withFixedNow(st, now)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content-type, got %q", got)
	}

	var resp api.AllRates
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.UsdBcv.Value != 480.5 || resp.EurBcv.Value != 520.1 || resp.UsdtBinance.Value != 535.2 {
		t.Fatalf("unexpected values: %+v", resp)
	}
	if resp.UsdBcv.IsStale || resp.EurBcv.IsStale || resp.UsdtBinance.IsStale {
		t.Fatalf("expected no stale entries: %+v", resp)
	}
	if resp.UsdBcv.DataAgeSeconds != 30 {
		t.Fatalf("expected age=30s, got %d", resp.UsdBcv.DataAgeSeconds)
	}
}

func TestGetRates_EmptyStateIsStale(t *testing.T) {
	st := state.NewState()
	srv := withFixedNow(st, time.Now())
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp api.AllRates
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !resp.UsdBcv.IsStale || !resp.EurBcv.IsStale || !resp.UsdtBinance.IsStale {
		t.Fatalf("expected every entry to be stale when state is empty: %+v", resp)
	}
	if resp.UsdBcv.LastUpdated != nil {
		t.Fatalf("expected null last_updated, got %v", resp.UsdBcv.LastUpdated)
	}
}

func TestGetRatesCurrency_OK(t *testing.T) {
	now := time.Now()
	ago := now.Add(-1 * time.Minute)
	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdtBinance: state.RateData{Value: 540.0, LastUpdated: &ago},
	})

	srv := withFixedNow(st, now)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/usdt_binance", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp api.RateEntry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Value != 540.0 || resp.IsStale || resp.DataAgeSeconds != 60 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetRatesCurrency_Unknown(t *testing.T) {
	srv := withFixedNow(state.NewState(), time.Now())
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/btc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetHealth_OK(t *testing.T) {
	now := time.Now()
	recent := now.Add(-1 * time.Minute)
	bcvRecent := now.Add(-1 * time.Hour)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:      state.RateData{Value: 1, LastUpdated: &bcvRecent},
		EurBcv:      state.RateData{Value: 1, LastUpdated: &bcvRecent},
		UsdtBinance: state.RateData{Value: 1, LastUpdated: &recent},
	})
	srv := withFixedNow(st, now)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp api.Health
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != api.Ok {
		t.Fatalf("expected status=ok, got %s", resp.Status)
	}
	if resp.Stale != nil && len(*resp.Stale) != 0 {
		t.Fatalf("expected no stale list, got %v", *resp.Stale)
	}
}

func TestGetHealth_DegradedWhenStale(t *testing.T) {
	now := time.Now()
	veryOld := now.Add(-1 * time.Hour) // > 15min Binance threshold
	bcvOK := now.Add(-1 * time.Hour)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:      state.RateData{Value: 1, LastUpdated: &bcvOK},
		EurBcv:      state.RateData{Value: 1, LastUpdated: &bcvOK},
		UsdtBinance: state.RateData{Value: 1, LastUpdated: &veryOld},
	})
	srv := withFixedNow(st, now)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var resp api.Health
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != api.Degraded {
		t.Fatalf("expected status=degraded, got %s", resp.Status)
	}
	if resp.Stale == nil || len(*resp.Stale) != 1 || (*resp.Stale)[0] != api.UsdtBinance {
		t.Fatalf("expected stale=[usdt_binance], got %v", resp.Stale)
	}
}

func TestGetHealth_DegradedWhenProviderFailing(t *testing.T) {
	// Timestamps are all fresh, but the Binance provider is marked as failing.
	// /health must still return 503 and include usdt_binance in the stale list.
	now := time.Now()
	recent := now.Add(-1 * time.Minute)
	bcvRecent := now.Add(-1 * time.Hour)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:      state.RateData{Value: 480.0, LastUpdated: &bcvRecent},
		EurBcv:      state.RateData{Value: 520.0, LastUpdated: &bcvRecent},
		UsdtBinance: state.RateData{Value: 530.0, LastUpdated: &recent},
	})
	state.MarkBinanceFailing(st)

	srv := withFixedNow(st, now)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (provider failing), got %d", w.Code)
	}

	var resp api.Health
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != api.Degraded {
		t.Fatalf("expected status=degraded, got %s", resp.Status)
	}
	if resp.Stale == nil || len(*resp.Stale) != 1 || (*resp.Stale)[0] != api.UsdtBinance {
		t.Fatalf("expected stale=[usdt_binance], got %v", resp.Stale)
	}
}

func TestGetRates_ProviderFailingIsStaleEvenIfRecent(t *testing.T) {
	now := time.Now()
	recent := now.Add(-1 * time.Minute)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:      state.RateData{Value: 480.0, LastUpdated: &recent},
		EurBcv:      state.RateData{Value: 520.0, LastUpdated: &recent},
		UsdtBinance: state.RateData{Value: 530.0, LastUpdated: &recent},
	})
	state.MarkBcvFailing(st)

	srv := withFixedNow(st, now)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp api.AllRates
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.UsdBcv.IsStale {
		t.Fatal("expected UsdBcv.is_stale=true when provider is failing")
	}
	if !resp.EurBcv.IsStale {
		t.Fatal("expected EurBcv.is_stale=true when provider is failing")
	}
	if resp.UsdtBinance.IsStale {
		t.Fatal("expected UsdtBinance.is_stale=false (not failing)")
	}
}

// --- History endpoint ---

// mockStore is an in-memory implementation of db.Store for handler tests.
type mockStore struct {
	entries []db.HistoryEntry
	lastErr error
}

func (m *mockStore) InsertRate(_ context.Context, _ string, _ float64, _ time.Time) error {
	return nil
}

func (m *mockStore) GetHistory(_ context.Context, _ string, _, _ time.Time) ([]db.HistoryEntry, error) {
	return m.entries, m.lastErr
}

func (m *mockStore) GetLatestRates(_ context.Context) (map[string]db.HistoryEntry, error) {
	return nil, nil
}

func (m *mockStore) Close() {}

func withStore(s *state.State, store db.Store) Server {
	srv := NewServerWithStore(s, store)
	return srv
}

func TestGetRatesCurrencyHistory_NoStore_Returns503(t *testing.T) {
	st := state.NewState()
	srv := NewServer(st)
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/usd_bcv/history?fromDate=2026-01-01&toDate=2026-01-31", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when store is nil, got %d", w.Code)
	}
}

func TestGetRatesCurrencyHistory_InvalidDate_Returns400(t *testing.T) {
	st := state.NewState()
	srv := withStore(st, &mockStore{})
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/usd_bcv/history?fromDate=bad&toDate=2026-01-31", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetRatesCurrencyHistory_OK(t *testing.T) {
	ts1 := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	st := state.NewState()
	srv := withStore(st, &mockStore{
		entries: []db.HistoryEntry{
			{Value: 480.5, RecordedAt: ts1},
			{Value: 481.0, RecordedAt: ts2},
		},
	})
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/usd_bcv/history?fromDate=2026-01-01&toDate=2026-01-31", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp api.RateHistory
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Currency != api.UsdBcv {
		t.Fatalf("expected currency=usd_bcv, got %s", resp.Currency)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Value != 480.5 {
		t.Fatalf("expected first value=480.5, got %f", resp.Entries[0].Value)
	}
}

func TestGetRatesCurrencyHistory_UnknownCurrency_Returns404(t *testing.T) {
	st := state.NewState()
	srv := withStore(st, &mockStore{})
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/unknown_xyz/history?fromDate=2026-01-01&toDate=2026-01-31", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
