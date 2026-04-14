package main

import (
	"encoding/json"
	"fmt"
	"github.com/nanezx/ve-xchange-api/internal/state"
	"net/http"
)

type HelloWorldHandler struct{}

func (HelloWorldHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world!")
}

type RatesHandler struct{}

func (RatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state.AppState.RLock()
	defer state.AppState.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(state.AppState.Rates)
}
