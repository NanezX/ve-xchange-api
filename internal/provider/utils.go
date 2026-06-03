package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// withRetry calls fn up to maxAttempts times. On failure it waits baseDelay,
// then doubles the delay before the next attempt. Pass baseDelay=0 to disable
// sleeping (useful in tests).
func withRetry[T any](maxAttempts int, baseDelay time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	delay := baseDelay
	var lastErr error
	for i := range maxAttempts {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if i < maxAttempts-1 && delay > 0 {
			time.Sleep(delay)
			delay *= 2
		}
	}
	return zero, fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

func fetchJson[T any](client HTTPDoer, req *http.Request) (T, error) {
	var result T

	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer func() { _ = resp.Body.Close() }()

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
