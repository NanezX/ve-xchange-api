package provider

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

const validBCVHTML = `
<section id="tasas">
  <div id="dolar"><span>USD</span><strong>1.234,56</strong></div>
  <div id="euro"><span>EUR</span><strong>1 345,67</strong></div>
	<div class="pull-right dinpro"><span class="date-display-single">Martes, 14 Julio 2026</span></div>
</section>`

func TestBCVProviderGetPrices(t *testing.T) {
	provider := NewBCVProvider(&FakeHTTPDoer{StatusCode: http.StatusOK, Body: validBCVHTML})

	response, err := provider.GetPrices()
	if err != nil {
		t.Fatalf("GetPrices() error = %v", err)
	}
	if response.Values["USD_BCV"] != 1234.56 {
		t.Fatalf("USD_BCV = %v, want 1234.56", response.Values["USD_BCV"])
	}
	if response.Values["EUR_BCV"] != 1345.67 {
		t.Fatalf("EUR_BCV = %v, want 1345.67", response.Values["EUR_BCV"])
	}

	wantDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
	if response.EffectiveDate == nil || !response.EffectiveDate.Equal(wantDate) {
		t.Fatalf("EffectiveDate = %v, want %v", response.EffectiveDate, wantDate)
	}
}

func TestBCVProviderRejectsIncompleteOrInvalidHTML(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{
			name: "missing EUR",
			body: `<div id="dolar"><strong>700,22</strong></div><div id="fecha">Fecha Valor: Lunes, 13 Julio 2026</div>`,
		},
		{
			name: "missing effective date",
			body: `<div id="dolar"><strong>700,22</strong></div><div id="euro"><strong>800,33</strong></div>`,
		},
		{
			name: "unexpected HTML",
			body: `<main><p>Sin tasas publicadas</p></main>`,
		},
		{
			name: "zero USD",
			body: `<div id="dolar"><strong>0,00</strong></div><div id="euro"><strong>800,33</strong></div><div id="fecha">Fecha Valor: Lunes, 13 Julio 2026</div>`,
		},
		{
			name: "non-numeric EUR",
			body: `<div id="dolar"><strong>700,22</strong></div><div id="euro"><strong>N/A</strong></div><div id="fecha">Fecha Valor: Lunes, 13 Julio 2026</div>`,
		},
		{
			name: "invalid date",
			body: `<div id="dolar"><strong>700,22</strong></div><div id="euro"><strong>800,33</strong></div><div id="fecha">Fecha Valor: Lunes, 32 Julio 2026</div>`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := NewBCVProvider(&FakeHTTPDoer{StatusCode: http.StatusOK, Body: testCase.body})
			provider.retryBaseDelay = 0

			if _, err := provider.GetPrices(); err == nil {
				t.Fatal("GetPrices() error = nil, want error")
			}
		})
	}
}

func TestBCVProviderRetriesTransportAndHTTPFailures(t *testing.T) {
	testCases := []struct {
		name   string
		client *FakeHTTPDoer
	}{
		{
			name:   "transport error",
			client: &FakeHTTPDoer{Error: errors.New("connection timeout")},
		},
		{
			name:   "HTTP error",
			client: &FakeHTTPDoer{StatusCode: http.StatusBadGateway},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := NewBCVProvider(testCase.client)
			provider.retryBaseDelay = 0

			if _, err := provider.GetPrices(); err == nil {
				t.Fatal("GetPrices() error = nil, want error")
			}
		})
	}
}

func TestParseEffectiveDate(t *testing.T) {
	tests := []struct {
		raw       string
		want      time.Time
		wantError bool
	}{
		{"Martes, 14 Enero 2026", time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 febrero 2026", time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 marzo 2026", time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 abril 2026", time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 mayo 2026", time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 junio 2026", time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 julio 2026", time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 agosto 2026", time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 setiembre 2026", time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 septiembre 2026", time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 octubre 2026", time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 noviembre 2026", time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC), false},
		{"14 diciembre 2026", time.Date(2026, 12, 14, 0, 0, 0, 0, time.UTC), false},
		{"Fecha Valor: lunes, 13 julio 2026", time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC), false},
		{"", time.Time{}, true},
		{"sin fecha", time.Time{}, true},
		{"14 undecimbre 2026", time.Time{}, true},
		{"32 julio 2026", time.Time{}, true},
	}

	for _, tt := range tests {
		name := tt.raw
		if len(name) > 40 {
			name = name[:40]
		}
		t.Run(name, func(t *testing.T) {
			got, err := parseEffectiveDate(tt.raw)
			if tt.wantError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantError && !got.Equal(tt.want) {
				t.Fatalf("parseEffectiveDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLocalizedNumber(t *testing.T) {
	testCases := []struct {
		raw  string
		want float64
	}{
		{raw: "700,22", want: 700.22},
		{raw: "1.234,56", want: 1234.56},
		{raw: "1 234,56", want: 1234.56},
		{raw: "700.22", want: 700.22},
	}

	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			got, err := parseLocalizedNumber(testCase.raw)
			if err != nil {
				t.Fatalf("parseLocalizedNumber() error = %v", err)
			}
			if got != testCase.want {
				t.Fatalf("parseLocalizedNumber() = %v, want %v", got, testCase.want)
			}
		})
	}
}
