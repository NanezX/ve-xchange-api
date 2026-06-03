package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

type PriceProvider interface {
	GetPrices() (rates.PriceResponse, error)
	GetName() string
}

type ProviderJob struct {
	Provider PriceProvider
	Every    time.Duration
	Apply    func(rates.PriceResponse)
	// OnFail is called on every fetch failure once consecutiveFails reaches 3.
	// Useful for marking provider state as degraded. Optional.
	OnFail func(consecutiveFails int64)
	// OnRecover is called on the first successful fetch after a streak of ≥3
	// failures. Useful for clearing the degraded flag. Optional.
	OnRecover func()
}

// StartPriceWorker launches one goroutine per job. Each goroutine performs an
// initial fetch and then loops until ctx is cancelled. The returned WaitGroup
// completes once every goroutine has exited.
func StartPriceWorker(ctx context.Context, jobs []ProviderJob) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, job := range jobs {
		currentJob := job
		wg.Add(1)

		go func() {
			defer wg.Done()

			var consecutiveFails int64

			fetch := func() {
				resp, err := currentJob.Provider.GetPrices()
				if err != nil {
					consecutiveFails++
					if consecutiveFails >= 3 {
						slog.Error("consecutive provider failures",
							"provider", currentJob.Provider.GetName(),
							"consecutive_failures", consecutiveFails,
							"error", err)
						if currentJob.OnFail != nil {
							currentJob.OnFail(consecutiveFails)
						}
					} else {
						slog.Warn("provider fetch failed",
							"provider", currentJob.Provider.GetName(),
							"failure_number", consecutiveFails,
							"error", err)
					}
					return
				}
				if consecutiveFails > 0 {
					slog.Info("provider recovered",
						"provider", currentJob.Provider.GetName(),
						"after_failures", consecutiveFails)
					if consecutiveFails >= 3 && currentJob.OnRecover != nil {
						currentJob.OnRecover()
					}
					consecutiveFails = 0
				}
				currentJob.Apply(resp)
				slog.Info("provider updated", "provider", currentJob.Provider.GetName())
			}

			fetch()

			ticker := time.NewTicker(currentJob.Every)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					fetch()
				}
			}
		}()
	}

	return &wg
}
