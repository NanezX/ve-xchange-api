package provider

import (
	"errors"
	"fmt"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

// ValidateBCVPublication checks that the response's effective date corresponds
// to the next business day in the given location. It is used by the worker's
// BusinessWindow to reject stale BCV pages.
func ValidateBCVPublication(response rates.PriceResponse, now time.Time, loc *time.Location) error {
	if response.EffectiveDate == nil {
		return errors.New("BCV response is missing its effective date")
	}

	expected := nextBusinessDate(now, loc)
	effectiveDate := *response.EffectiveDate
	if effectiveDate.Year() != expected.Year() ||
		effectiveDate.Month() != expected.Month() ||
		effectiveDate.Day() != expected.Day() {
		return fmt.Errorf("BCV effective date %s does not match expected %s",
			effectiveDate.Format(time.DateOnly), expected.Format(time.DateOnly))
	}
	return nil
}

func nextBusinessDate(now time.Time, loc *time.Location) time.Time {
	date := now.In(loc).AddDate(0, 0, 1)
	for date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		date = date.AddDate(0, 0, 1)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
