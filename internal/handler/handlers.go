package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/api"
	"github.com/nanezx/ve-xchange-api/internal/db"
	"github.com/nanezx/ve-xchange-api/internal/state"
)

// Staleness thresholds per currency. A rate older than its threshold is
// reported with `is_stale = true` and forces /health into "degraded" status.
// BCV publishes once per day (~6 PM VZ time); the worker fetches at midnight
// VZ, so data can legitimately be up to 24h old. 26h gives a safe margin.
const (
	stalenessBcv     = 26 * time.Hour
	stalenessBinance = 15 * time.Minute
)

type Server struct {
	appState *state.State
	store    db.Store
	now      func() time.Time
}

// NewServer creates a Server with no database store.
// Use NewServerWithStore when historical data persistence is available.
func NewServer(appState *state.State) Server {
	return Server{appState: appState, now: time.Now}
}

// NewServerWithStore creates a Server backed by a database store for
// historical rate queries.
func NewServerWithStore(appState *state.State, store db.Store) Server {
	return Server{appState: appState, store: store, now: time.Now}
}

func (h Server) GetRates(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Error{Error: "database not configured"})
		return
	}

	latest, err := h.store.GetLatestRates(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Error{Error: "failed to query latest rates"})
		return
	}

	snapshot := h.appState.GetRates()
	now := h.now()

	usdBcv, hasUsdBcv := latest[string(api.UsdBcv)]
	eurBcv, hasEurBcv := latest[string(api.EurBcv)]
	usdt, hasUsdt := latest[string(api.Usdt)]
	usdtCompra, hasUsdtCompra := latest[string(api.UsdtCompra)]
	usdtVenta, hasUsdtVenta := latest[string(api.UsdtVenta)]

	body := api.AllRates{
		UsdBcv:     toStoredRateEntry(usdBcv, hasUsdBcv, snapshot.UsdBcv.ProviderFailing, stalenessBcv, now),
		EurBcv:     toStoredRateEntry(eurBcv, hasEurBcv, snapshot.EurBcv.ProviderFailing, stalenessBcv, now),
		Usdt:       toStoredRateEntry(usdt, hasUsdt, snapshot.Usdt.ProviderFailing, stalenessBinance, now),
		UsdtCompra: toStoredRateEntry(usdtCompra, hasUsdtCompra, snapshot.UsdtCompra.ProviderFailing, stalenessBinance, now),
		UsdtVenta:  toStoredRateEntry(usdtVenta, hasUsdtVenta, snapshot.UsdtVenta.ProviderFailing, stalenessBinance, now),
	}

	writeJSON(w, http.StatusOK, body)
}

func (h Server) GetRatesCurrency(w http.ResponseWriter, r *http.Request, currency api.Currency) {
	if !currency.Valid() {
		writeJSON(w, http.StatusNotFound, api.Error{Error: "unknown currency"})
		return
	}
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Error{Error: "database not configured"})
		return
	}

	latest, err := h.store.GetLatestRates(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Error{Error: "failed to query latest rates"})
		return
	}

	snapshot := h.appState.GetRates()
	now := h.now()

	var entry api.RateEntry
	switch currency {
	case api.UsdBcv:
		stored, ok := latest[string(api.UsdBcv)]
		entry = toStoredRateEntry(stored, ok, snapshot.UsdBcv.ProviderFailing, stalenessBcv, now)
	case api.EurBcv:
		stored, ok := latest[string(api.EurBcv)]
		entry = toStoredRateEntry(stored, ok, snapshot.EurBcv.ProviderFailing, stalenessBcv, now)
	case api.Usdt:
		stored, ok := latest[string(api.Usdt)]
		entry = toStoredRateEntry(stored, ok, snapshot.Usdt.ProviderFailing, stalenessBinance, now)
	case api.UsdtCompra:
		stored, ok := latest[string(api.UsdtCompra)]
		entry = toStoredRateEntry(stored, ok, snapshot.UsdtCompra.ProviderFailing, stalenessBinance, now)
	case api.UsdtVenta:
		stored, ok := latest[string(api.UsdtVenta)]
		entry = toStoredRateEntry(stored, ok, snapshot.UsdtVenta.ProviderFailing, stalenessBinance, now)
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	snapshot := h.appState.GetRates()
	now := h.now()

	stale := make([]api.Currency, 0, 3)
	if isStale(snapshot.UsdBcv, stalenessBcv, now) {
		stale = append(stale, api.UsdBcv)
	}
	if isStale(snapshot.EurBcv, stalenessBcv, now) {
		stale = append(stale, api.EurBcv)
	}
	if isStale(snapshot.Usdt, stalenessBinance, now) {
		stale = append(stale, api.Usdt)
	}
	if isStale(snapshot.UsdtCompra, stalenessBinance, now) {
		stale = append(stale, api.UsdtCompra)
	}
	if isStale(snapshot.UsdtVenta, stalenessBinance, now) {
		stale = append(stale, api.UsdtVenta)
	}

	body := api.Health{Status: api.Ok}
	status := http.StatusOK
	if len(stale) > 0 {
		body.Status = api.Degraded
		body.Stale = &stale
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, body)
}

func toStoredRateEntry(entry db.HistoryEntry, found, providerFailing bool, threshold time.Duration, now time.Time) api.RateEntry {
	if !found {
		return api.RateEntry{IsStale: true}
	}

	recordedAt := entry.RecordedAt
	age := max(now.Sub(recordedAt), 0)
	return api.RateEntry{
		Value:          entry.Value,
		LastUpdated:    &recordedAt,
		DataAgeSeconds: int(age.Seconds()),
		IsStale:        providerFailing || age > threshold,
	}
}

func isStale(d state.RateData, threshold time.Duration, now time.Time) bool {
	if d.ProviderFailing {
		return true
	}
	if d.LastUpdated == nil {
		return true
	}
	return now.Sub(*d.LastUpdated) > threshold
}

func (h Server) GetRatesCurrencyHistory(w http.ResponseWriter, r *http.Request, currency api.Currency, params api.GetRatesCurrencyHistoryParams) {
	if !currency.Valid() {
		writeJSON(w, http.StatusNotFound, api.Error{Error: "unknown currency"})
		return
	}
	if h.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Error{Error: "database not configured"})
		return
	}

	from, err := time.Parse(time.DateOnly, params.FromDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, api.Error{Error: "invalid fromDate: expected YYYY-MM-DD"})
		return
	}
	to, err := time.Parse(time.DateOnly, params.ToDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, api.Error{Error: "invalid toDate: expected YYYY-MM-DD"})
		return
	}
	// Make 'to' exclusive-end: include the full last day.
	to = to.AddDate(0, 0, 1)

	entries, err := h.store.GetHistory(r.Context(), string(currency), from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, api.Error{Error: "failed to query history"})
		return
	}

	apiEntries := make([]api.HistoryEntry, len(entries))
	for i, e := range entries {
		apiEntries[i] = api.HistoryEntry{
			Value:      e.Value,
			RecordedAt: e.RecordedAt,
			IsAverage:  e.IsAverage,
		}
	}

	writeJSON(w, http.StatusOK, api.RateHistory{
		Currency: currency,
		Entries:  apiEntries,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
