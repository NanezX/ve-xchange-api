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
	snapshot := h.appState.GetRates()
	now := h.now()

	body := api.AllRates{
		UsdBcv:          toRateEntry(snapshot.UsdBcv, stalenessBcv, now),
		EurBcv:          toRateEntry(snapshot.EurBcv, stalenessBcv, now),
		UsdtBinance:     toRateEntry(snapshot.UsdtBinance, stalenessBinance, now),
		UsdtBinanceBuy:  toRateEntry(snapshot.UsdtBinanceBuy, stalenessBinance, now),
		UsdtBinanceSell: toRateEntry(snapshot.UsdtBinanceSell, stalenessBinance, now),
	}

	writeJSON(w, http.StatusOK, body)
}

func (h Server) GetRatesCurrency(w http.ResponseWriter, r *http.Request, currency api.Currency) {
	if !currency.Valid() {
		writeJSON(w, http.StatusNotFound, api.Error{Error: "unknown currency"})
		return
	}

	snapshot := h.appState.GetRates()
	now := h.now()

	var entry api.RateEntry
	switch currency {
	case api.UsdBcv:
		entry = toRateEntry(snapshot.UsdBcv, stalenessBcv, now)
	case api.EurBcv:
		entry = toRateEntry(snapshot.EurBcv, stalenessBcv, now)
	case api.UsdtBinance:
		entry = toRateEntry(snapshot.UsdtBinance, stalenessBinance, now)
	case api.UsdtBinanceBuy:
		entry = toRateEntry(snapshot.UsdtBinanceBuy, stalenessBinance, now)
	case api.UsdtBinanceSell:
		entry = toRateEntry(snapshot.UsdtBinanceSell, stalenessBinance, now)
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
	if isStale(snapshot.UsdtBinance, stalenessBinance, now) {
		stale = append(stale, api.UsdtBinance)
	}
	if isStale(snapshot.UsdtBinanceBuy, stalenessBinance, now) {
		stale = append(stale, api.UsdtBinanceBuy)
	}
	if isStale(snapshot.UsdtBinanceSell, stalenessBinance, now) {
		stale = append(stale, api.UsdtBinanceSell)
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

// toRateEntry projects a state.RateData onto the api.RateEntry contract,
// computing age and staleness against the provided threshold.
func toRateEntry(d state.RateData, threshold time.Duration, now time.Time) api.RateEntry {
	if d.LastUpdated == nil || d.ProviderFailing {
		return api.RateEntry{
			Value:          d.Value,
			LastUpdated:    d.LastUpdated,
			DataAgeSeconds: 0,
			IsStale:        true,
		}
	}

	age := max(now.Sub(*d.LastUpdated), 0)

	return api.RateEntry{
		Value:          d.Value,
		LastUpdated:    d.LastUpdated,
		DataAgeSeconds: int(age.Seconds()),
		IsStale:        age > threshold,
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
