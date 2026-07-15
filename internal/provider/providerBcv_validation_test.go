package provider

import (
	"testing"
	"time"

	"github.com/nanezx/ve-xchange-api/internal/rates"
)

func TestNextBusinessDate(t *testing.T) {
	loc := time.FixedZone("UTC-4", -4*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "monday → tuesday",
			now:  time.Date(2026, 7, 13, 17, 0, 0, 0, loc),
			want: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "friday → monday",
			now:  time.Date(2026, 7, 17, 17, 0, 0, 0, loc),
			want: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "saturday → monday",
			now:  time.Date(2026, 7, 18, 10, 0, 0, 0, loc),
			want: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "sunday → monday",
			now:  time.Date(2026, 7, 19, 10, 0, 0, 0, loc),
			want: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "tuesday → wednesday",
			now:  time.Date(2026, 7, 14, 17, 0, 0, 0, loc),
			want: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextBusinessDate(tt.now, loc)
			if !got.Equal(tt.want) {
				t.Fatalf("nextBusinessDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateBCVPublication(t *testing.T) {
	loc := time.FixedZone("UTC-4", -4*60*60)
	tue := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	mon20 := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		response  rates.PriceResponse
		now       time.Time
		wantError bool
	}{
		{
			name:      "nil effective date",
			response:  rates.PriceResponse{EffectiveDate: nil},
			now:       time.Date(2026, 7, 13, 17, 0, 0, 0, loc),
			wantError: true,
		},
		{
			name: "correct next day (monday → tuesday)",
			response: rates.PriceResponse{
				Values:        map[string]float64{"USD_BCV": 100},
				EffectiveDate: &tue,
			},
			now:       time.Date(2026, 7, 13, 17, 0, 0, 0, loc),
			wantError: false,
		},
		{
			name: "stale date rejects (monday expects tuesday, has monday)",
			response: rates.PriceResponse{
				Values:        map[string]float64{"USD_BCV": 100},
				EffectiveDate: &mon,
			},
			now:       time.Date(2026, 7, 13, 17, 0, 0, 0, loc),
			wantError: true,
		},
		{
			name: "friday expects monday",
			response: rates.PriceResponse{
				Values:        map[string]float64{"USD_BCV": 100},
				EffectiveDate: &mon20,
			},
			now:       time.Date(2026, 7, 17, 17, 0, 0, 0, loc),
			wantError: false,
		},
		{
			name: "saturday rejects stale monday (expects monday, has tuesday)",
			response: rates.PriceResponse{
				Values:        map[string]float64{"USD_BCV": 100},
				EffectiveDate: &tue,
			},
			now:       time.Date(2026, 7, 18, 10, 0, 0, 0, loc),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBCVPublication(tt.response, tt.now, loc)
			if tt.wantError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
