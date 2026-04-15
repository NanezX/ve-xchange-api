package worker

import (
	"fmt"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
	"github.com/nanezx/ve-xchange-api/internal/state"
)

type PriceProvider interface {
	GetPrices() (rates.PriceResponse, error)
	GetName() string
}

type ProviderJob struct {
	Provider PriceProvider
	Every    time.Duration
	Apply    func(*state.State, rates.PriceResponse)
}

func StartPriceWorker(appState *state.State, jobs []ProviderJob) {

	for _, job := range jobs {
		currentJob := job

		go func() {
			resp, err := currentJob.Provider.GetPrices()
			if err != nil {
				fmt.Printf("Error initializing %s: %v\n", currentJob.Provider.GetName(), err)
			} else {
				currentJob.Apply(appState, resp)
			}

			ticker := time.NewTicker(currentJob.Every)
			defer ticker.Stop()

			for range ticker.C {
				resp, err := currentJob.Provider.GetPrices()
				if err != nil {
					fmt.Printf("Error updating %s: %v\n", currentJob.Provider.GetName(), err)
					continue
				}

				currentJob.Apply(appState, resp)
				fmt.Printf("Updated %s price\n", currentJob.Provider.GetName())
			}

		}()

	}
}
