package main

import (
	"fmt"
	"net/http"
)

func main() {
	err := LoadConfig()

	if err != nil {
		fmt.Printf("Failed to load env file... [Error]: %v", err)
		return
	}

	go StartPriceWorker()

	mux := http.NewServeMux()

	mux.Handle("/hello", HelloWorldHandler{})
	mux.Handle("/rates", RatesHandler{})

	fmt.Printf("Servidor corriendo en http://localhost:%d\n", AppConfig.AppPort)
	http.ListenAndServe(fmt.Sprintf(":%d", AppConfig.AppPort), mux)
}
