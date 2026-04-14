package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPDoer interface {
    Do(*http.Request) (*http.Response, error)
}

func fetchJson[T any](client HTTPDoer, req *http.Request) (T, error) {
	var result T

	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("Failed to fetch json. Status code: %d - Error: %s", resp.StatusCode, string(errorBody))
	}

	// Write the response
	err = json.NewDecoder(resp.Body).Decode(&result)

	if err != nil {
		return result, err
	}

	return result, nil
}
