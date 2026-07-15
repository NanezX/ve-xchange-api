package rates

import "time"

// PriceResponse contains prices returned by a provider. EffectiveDate is set
// only when a provider publishes the date for which its rates are valid.
type PriceResponse struct {
	Values        map[string]float64
	EffectiveDate *time.Time
}
