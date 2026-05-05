package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nanezx/ve-xchange-api/internal/api"
	"github.com/nanezx/ve-xchange-api/internal/state"
)

type Server struct {
	appState *state.State
}

func NewServer(appState *state.State) Server {
	return Server{appState: appState}
}

func (handler Server) GetRates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handler.appState.GetRates())
}

func (Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.Health{Status: "ok"})
}
