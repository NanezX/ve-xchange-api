package main

import (
	"fmt"
	"github.com/nanezx/ve-xchange-api/internal/provider"
	"time"
)

type ProviderJob struct {
	Provider PriceProvider
	Every    time.Duration
	Apply    func(provider.PriceResponse)
}

func StartPriceWorker(jobs []ProviderJob) {

	for _, job := range jobs {
		currentJob := job

		go func() {
			resp, err := currentJob.Provider.GetPrices()
			if err != nil {
				fmt.Printf("Error initializing %s: %v\n", currentJob.Provider.GetName(), err)
			} else {
				currentJob.Apply(resp)
			}

			ticker := time.NewTicker(currentJob.Every)
			defer ticker.Stop()

			for range ticker.C {
				resp, err := currentJob.Provider.GetPrices()
				if err != nil {
					fmt.Printf("Error updating %s: %v\n", currentJob.Provider.GetName(), err)
					continue
				}

				currentJob.Apply(resp)
				fmt.Printf("Updated %s price\n", currentJob.Provider.GetName())
			}

		}()

	}
}
