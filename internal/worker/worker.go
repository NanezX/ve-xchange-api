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

// TimeOfDay represents a wall-clock time in a specific timezone.
// Used to schedule a provider job at a fixed time each day.
type TimeOfDay struct {
	Hour     int
	Minute   int
	Location *time.Location // nil means UTC
}

type ProviderJob struct {
	Provider PriceProvider
	// Every drives an interval-based schedule (e.g. every 5 min for Binance).
	// Mutually exclusive with DailyAt.
	Every time.Duration
	// DailyAt drives a once-per-day schedule at a fixed wall-clock time.
	// When set, Every is ignored. An initial fetch still runs at startup.
	// Mutually exclusive with Every.
	DailyAt *TimeOfDay
	Apply   func(rates.PriceResponse)
	// OnFail is called on every fetch failure once consecutiveFails reaches 3.
	// Useful for marking provider state as degraded. Optional.
	OnFail func(consecutiveFails int64)
	// OnRecover is called on the first successful fetch after a streak of ≥3
	// failures. Useful for clearing the degraded flag. Optional.
	OnRecover func()
	// AfterFetch is called after every fetch attempt. consecutiveFails is the
	// updated streak count (0 after a successful fetch). Optional.
	AfterFetch func(consecutiveFails int64, success bool)
}

// nextDaily returns the duration until the next occurrence of tod after now.
// If the target time today has already passed (or equals now), the next
// occurrence is scheduled for tomorrow.
func nextDaily(tod TimeOfDay, now time.Time) time.Duration {
	loc := tod.Location
	if loc == nil {
		loc = time.UTC
	}
	nowInLoc := now.In(loc)
	candidate := time.Date(nowInLoc.Year(), nowInLoc.Month(), nowInLoc.Day(),
		tod.Hour, tod.Minute, 0, 0, loc)
	if !candidate.After(nowInLoc) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate.Sub(now)
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
					if currentJob.AfterFetch != nil {
						currentJob.AfterFetch(consecutiveFails, false)
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
				if currentJob.AfterFetch != nil {
					currentJob.AfterFetch(consecutiveFails, true)
				}
			}

			fetch()

			if currentJob.DailyAt != nil {
				for {
					timer := time.NewTimer(nextDaily(*currentJob.DailyAt, time.Now()))
					select {
					case <-ctx.Done():
						timer.Stop()
						return
					case <-timer.C:
						fetch()
						// On failure, retry every 30 min up to 10 times before
						// giving up until the next daily window.
						for retries := 0; retries < 10 && consecutiveFails > 0; retries++ {
							slog.Warn("daily fetch failed, retrying",
								"provider", currentJob.Provider.GetName(),
								"retry", retries+1,
								"max_retries", 10,
								"next_retry_in", "30m")
							retryTimer := time.NewTimer(30 * time.Minute)
							select {
							case <-ctx.Done():
								retryTimer.Stop()
								return
							case <-retryTimer.C:
								fetch()
							}
						}
					}
				}
			}

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

// TaskJob is a scheduled background task that does not fetch from an external
// provider (e.g. a nightly DB consolidation). It fires once per day at DailyAt.
// Unlike ProviderJob, there is no initial execution at startup.
type TaskJob struct {
	// Name is used in log messages.
	Name string
	// DailyAt schedules the task at a fixed wall-clock time each day.
	DailyAt TimeOfDay
	// Run is called at each scheduled occurrence. The context is cancelled
	// when the worker is shutting down.
	Run func(ctx context.Context)
}

// StartTaskWorker launches one goroutine per job and returns a WaitGroup that
// completes once all goroutines have exited.
func StartTaskWorker(ctx context.Context, jobs []TaskJob) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, job := range jobs {
		j := job
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				timer := time.NewTimer(nextDaily(j.DailyAt, time.Now()))
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
					slog.Info("running scheduled task", "task", j.Name)
					j.Run(ctx)
					slog.Info("scheduled task completed", "task", j.Name)
				}
			}
		}()
	}

	return &wg
}
