package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nanezx/ve-xchange-api/internal/state"
)

type InfoHandler struct{}

func (InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world!")
}

type RatesHandler struct {
	appState *state.State
}

func NewRatesHandler(appState *state.State) RatesHandler {
	return RatesHandler{appState: appState}
}

func (handler RatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handler.appState.GetRates())
}
