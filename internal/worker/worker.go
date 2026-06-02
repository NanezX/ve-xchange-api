package worker

import (
	"context"
	"fmt"
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

			resp, err := currentJob.Provider.GetPrices()
			if err != nil {
				fmt.Printf("Error initializing %s: %v\n", currentJob.Provider.GetName(), err)
			} else {
				currentJob.Apply(resp)
			}

			ticker := time.NewTicker(currentJob.Every)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					resp, err := currentJob.Provider.GetPrices()
					if err != nil {
						fmt.Printf("Error updating %s: %v\n", currentJob.Provider.GetName(), err)
						continue
					}
					currentJob.Apply(resp)
					fmt.Printf("Updated %s price\n", currentJob.Provider.GetName())
				}
			}
		}()
	}

	return &wg
}
