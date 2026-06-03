package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/api"
	"github.com/nanezx/ve-xchange-api/internal/state"
)

// Staleness thresholds per currency. A rate older than its threshold is
// reported with `is_stale = true` and forces /health into "degraded" status.
const (
	stalenessBcv     = 12 * time.Hour
	stalenessBinance = 15 * time.Minute
)

type Server struct {
	appState *state.State
	now      func() time.Time
}

func NewServer(appState *state.State) Server {
	return Server{appState: appState, now: time.Now}
}

func (h Server) GetRates(w http.ResponseWriter, r *http.Request) {
	snapshot := h.appState.GetRates()
	now := h.now()

	body := api.AllRates{
		UsdBcv:      toRateEntry(snapshot.UsdBcv, stalenessBcv, now),
		EurBcv:      toRateEntry(snapshot.EurBcv, stalenessBcv, now),
		UsdtBinance: toRateEntry(snapshot.UsdtBinance, stalenessBinance, now),
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
