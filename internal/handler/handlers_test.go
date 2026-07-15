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
		UsdBcv: state.RateData{Value: 1, LastUpdated: &ago},
	})

	store := &mockStore{latest: map[string]db.HistoryEntry{
		string(api.UsdBcv):     {Value: 480.5, RecordedAt: ago},
		string(api.EurBcv):     {Value: 520.1, RecordedAt: ago},
		string(api.Usdt):       {Value: 535.2, RecordedAt: ago},
		string(api.UsdtCompra): {Value: 540.0, RecordedAt: ago},
		string(api.UsdtVenta):  {Value: 530.0, RecordedAt: ago},
	}}
	srv := withStore(st, store)
	srv.now = func() time.Time { return now }
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

	if resp.UsdBcv.Value != 480.5 || resp.EurBcv.Value != 520.1 || resp.Usdt.Value != 535.2 {
		t.Fatalf("unexpected values: %+v", resp)
	}
	if resp.UsdtCompra.Value != 540.0 {
		t.Fatalf("expected UsdtCompra=540.0, got %v", resp.UsdtCompra.Value)
	}
	if resp.UsdtVenta.Value != 530.0 {
		t.Fatalf("expected UsdtVenta=530.0, got %v", resp.UsdtVenta.Value)
	}
	if resp.UsdBcv.IsStale || resp.EurBcv.IsStale || resp.Usdt.IsStale ||
		resp.UsdtCompra.IsStale || resp.UsdtVenta.IsStale {
		t.Fatalf("expected no stale entries: %+v", resp)
	}
	if resp.UsdBcv.DataAgeSeconds != 30 {
		t.Fatalf("expected age=30s, got %d", resp.UsdBcv.DataAgeSeconds)
	}
	if store.latestCalls != 1 {
		t.Fatalf("expected one latest-rate query, got %d", store.latestCalls)
	}
}

func TestGetRates_EmptyStateIsStale(t *testing.T) {
	st := state.NewState()
	srv := withStore(st, &mockStore{latest: map[string]db.HistoryEntry{}})
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

	if !resp.UsdBcv.IsStale || !resp.EurBcv.IsStale || !resp.Usdt.IsStale ||
		!resp.UsdtCompra.IsStale || !resp.UsdtVenta.IsStale {
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
	store := &mockStore{latest: map[string]db.HistoryEntry{
		string(api.Usdt): {Value: 540.0, RecordedAt: ago},
	}}
	srv := withStore(st, store)
	srv.now = func() time.Time { return now }
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/usdt", nil)
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
	if store.latestCalls != 1 {
		t.Fatalf("expected one latest-rate query, got %d", store.latestCalls)
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
		UsdBcv:     state.RateData{Value: 1, LastUpdated: &bcvRecent},
		EurBcv:     state.RateData{Value: 1, LastUpdated: &bcvRecent},
		Usdt:       state.RateData{Value: 1, LastUpdated: &recent},
		UsdtCompra: state.RateData{Value: 1, LastUpdated: &recent},
		UsdtVenta:  state.RateData{Value: 1, LastUpdated: &recent},
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
	recent := now.Add(-1 * time.Minute)
	bcvOK := now.Add(-1 * time.Hour)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:     state.RateData{Value: 1, LastUpdated: &bcvOK},
		EurBcv:     state.RateData{Value: 1, LastUpdated: &bcvOK},
		Usdt:       state.RateData{Value: 1, LastUpdated: &veryOld},
		UsdtCompra: state.RateData{Value: 1, LastUpdated: &recent},
		UsdtVenta:  state.RateData{Value: 1, LastUpdated: &recent},
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
	if resp.Stale == nil || len(*resp.Stale) != 1 || (*resp.Stale)[0] != api.Usdt {
		t.Fatalf("expected stale=[usdt], got %v", resp.Stale)
	}
}

func TestGetHealth_DegradedWhenProviderFailing(t *testing.T) {
	// Timestamps are all fresh, but the Binance provider is marked as failing.
	// /health must still return 503 and include usdt in the stale list.
	now := time.Now()
	recent := now.Add(-1 * time.Minute)
	bcvRecent := now.Add(-1 * time.Hour)

	st := state.NewState()
	st.UpdateRates(state.StateRates{
		UsdBcv:     state.RateData{Value: 480.0, LastUpdated: &bcvRecent},
		EurBcv:     state.RateData{Value: 520.0, LastUpdated: &bcvRecent},
		Usdt:       state.RateData{Value: 530.0, LastUpdated: &recent},
		UsdtCompra: state.RateData{Value: 530.0, LastUpdated: &recent},
		UsdtVenta:  state.RateData{Value: 530.0, LastUpdated: &recent},
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
	// MarkBinanceFailing sets ProviderFailing on average, buy, and sell — all 3 are stale.
	if resp.Stale == nil || len(*resp.Stale) != 3 {
		t.Fatalf("expected stale=[usdt, usdt_venta, usdt_compra], got %v", resp.Stale)
	}
}

func TestGetRates_ProviderFailingIsStaleEvenIfRecent(t *testing.T) {
	now := time.Now()
	recent := now.Add(-1 * time.Minute)

	st := state.NewState()
	state.MarkBcvFailing(st)

	srv := withStore(st, &mockStore{latest: map[string]db.HistoryEntry{
		string(api.UsdBcv):     {Value: 480.0, RecordedAt: recent},
		string(api.EurBcv):     {Value: 520.0, RecordedAt: recent},
		string(api.Usdt):       {Value: 530.0, RecordedAt: recent},
		string(api.UsdtCompra): {Value: 535.0, RecordedAt: recent},
		string(api.UsdtVenta):  {Value: 525.0, RecordedAt: recent},
	}})
	srv.now = func() time.Time { return now }
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
	if resp.Usdt.IsStale {
		t.Fatal("expected Usdt.is_stale=false (not failing)")
	}
	if resp.UsdtCompra.IsStale {
		t.Fatal("expected UsdtCompra.is_stale=false (not failing)")
	}
	if resp.UsdtVenta.IsStale {
		t.Fatal("expected UsdtVenta.is_stale=false (not failing)")
	}
}

// --- History endpoint ---

// mockStore is an in-memory implementation of db.Store for handler tests.
type mockStore struct {
	entries     []db.HistoryEntry
	lastErr     error
	latest      map[string]db.HistoryEntry
	latestErr   error
	latestCalls int
}

func (m *mockStore) InsertRate(_ context.Context, _ string, _ float64, _ time.Time) error {
	return nil
}

func (m *mockStore) GetHistory(_ context.Context, _ string, _, _ time.Time) ([]db.HistoryEntry, error) {
	return m.entries, m.lastErr
}

func (m *mockStore) GetLatestRates(_ context.Context) (map[string]db.HistoryEntry, error) {
	m.latestCalls++
	return m.latest, m.latestErr
}

func (m *mockStore) ConsolidateDay(_ context.Context, _ string, _, _ time.Time) error {
	return nil
}

func (m *mockStore) Close() {}

func withStore(s *state.State, store db.Store) Server {
	srv := NewServerWithStore(s, store)
	return srv
}

func TestGetRates_NoStore_Returns503(t *testing.T) {
	srv := NewServer(state.NewState())
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when store is nil, got %d", w.Code)
	}
}

func TestGetRatesCurrency_StoreError_Returns503(t *testing.T) {
	srv := withStore(state.NewState(), &mockStore{latestErr: context.DeadlineExceeded})
	mux := mountMux(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/rates/usdt", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when latest-rate query fails, got %d", w.Code)
	}
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
