package main

import (
	"fmt"
	"net/http"
)

// var appState AppState = AppState{}

func main() {
	port := 8080
	mux := http.NewServeMux()

	mux.Handle("/hello", HelloWorldHandler{})
	mux.Handle("/rates", RatesHandler{})

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
