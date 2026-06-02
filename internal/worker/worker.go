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
