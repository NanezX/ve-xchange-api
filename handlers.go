package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HelloWorldHandler struct{}

func (HelloWorldHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world!")
}

type RatesHandler struct{}

func (RatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	AppState.RLock()
	defer AppState.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(AppState.Rates)
}
