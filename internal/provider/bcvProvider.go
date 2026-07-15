package provider

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/nanezx/ve-xchange-api/internal/rates"
)

const bcvURL = "https://bcv.org.ve"

var (
	nonNumericCharacters = regexp.MustCompile(`[^0-9,.-]`)
	effectiveDatePattern = regexp.MustCompile(`(?i)(\d{1,2})\s+([[:alpha:]]+)\s+(\d{4})`)
)

// BCVProvider fetches the official USD and EUR reference rates published by
// Banco Central de Venezuela.
type BCVProvider struct {
	baseURL        string
	client         HTTPDoer
	retryBaseDelay time.Duration
}

func NewBCVProvider(client HTTPDoer) *BCVProvider {
	p := &BCVProvider{
		baseURL:        bcvURL,
		retryBaseDelay: time.Second,
	}
	// BCV's server sends an incomplete TLS certificate chain (missing
	// intermediate CA). We must skip verification to scrape the page.
	// Tests pass a fake client (not *http.Client), so they are unaffected.
	if c, ok := client.(*http.Client); ok {
		tr := c.Transport
		if tr == nil {
			tr = http.DefaultTransport
		}
		clone := tr.(*http.Transport).Clone()
		if clone.TLSClientConfig == nil {
			clone.TLSClientConfig = new(tls.Config)
		}
		clone.TLSClientConfig.InsecureSkipVerify = true
		p.client = &http.Client{
			Transport: clone,
			Timeout:   c.Timeout,
		}
	} else {
		p.client = client
	}
	return p
}

func (p *BCVProvider) GetPrices() (rates.PriceResponse, error) {
	return withRetry(3, p.retryBaseDelay, func() (rates.PriceResponse, error) {
		req, err := http.NewRequest(http.MethodGet, p.baseURL, nil)
		if err != nil {
			return rates.PriceResponse{}, err
		}
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("User-Agent", "ve-xchange-api/1.0")

		resp, err := p.client.Do(req)
		if err != nil {
			return rates.PriceResponse{}, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return rates.PriceResponse{}, fmt.Errorf("BCV prices: unexpected HTTP status %d", resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return rates.PriceResponse{}, fmt.Errorf("BCV prices: parse HTML: %w", err)
		}

		return parseBCVRates(doc)
	})
}

func (p *BCVProvider) GetName() string {
	return "BCV"
}

func parseBCVRates(doc *goquery.Document) (rates.PriceResponse, error) {
	usdText, err := rateText(doc, "#dolar")
	if err != nil {
		return rates.PriceResponse{}, fmt.Errorf("BCV prices: USD: %w", err)
	}
	eurText, err := rateText(doc, "#euro")
	if err != nil {
		return rates.PriceResponse{}, fmt.Errorf("BCV prices: EUR: %w", err)
	}
	dateText, err := effectiveDateText(doc)
	if err != nil {
		return rates.PriceResponse{}, fmt.Errorf("BCV prices: effective date: %w", err)
	}

	usd, err := parseLocalizedNumber(usdText)
	if err != nil {
		return rates.PriceResponse{}, fmt.Errorf("BCV prices: USD: %w", err)
	}
	eur, err := parseLocalizedNumber(eurText)
	if err != nil {
		return rates.PriceResponse{}, fmt.Errorf("BCV prices: EUR: %w", err)
	}
	effectiveDate, err := parseEffectiveDate(dateText)
	if err != nil {
		return rates.PriceResponse{}, fmt.Errorf("BCV prices: effective date: %w", err)
	}

	return rates.PriceResponse{
		Values: map[string]float64{
			"USD_BCV": usd,
			"EUR_BCV": eur,
		},
		EffectiveDate: &effectiveDate,
	}, nil
}

func rateText(doc *goquery.Document, selector string) (string, error) {
	container := doc.Find(selector).First()
	if container.Length() == 0 {
		return "", fmt.Errorf("missing %s container", selector)
	}

	value := strings.TrimSpace(container.Find("strong").First().Text())
	if value == "" {
		value = strings.TrimSpace(container.Text())
	}
	if value == "" {
		return "", fmt.Errorf("empty %s container", selector)
	}
	return value, nil
}

func effectiveDateText(doc *goquery.Document) (string, error) {
	dateText := strings.TrimSpace(doc.Find("#dolar ~ .dinpro .date-display-single").First().Text())
	if dateText != "" {
		return dateText, nil
	}
	return rateText(doc, "#fecha")
}

func parseLocalizedNumber(raw string) (float64, error) {
	normalized := nonNumericCharacters.ReplaceAllString(raw, "")
	if normalized == "" {
		return 0, fmt.Errorf("missing numeric value in %q", raw)
	}

	if strings.Contains(normalized, ",") {
		normalized = strings.ReplaceAll(normalized, ".", "")
		normalized = strings.ReplaceAll(normalized, ",", ".")
	} else {
		normalized = strings.ReplaceAll(normalized, ",", "")
	}

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", raw, err)
	}
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid value %q", raw)
	}
	return value, nil
}

func parseEffectiveDate(raw string) (time.Time, error) {
	matches := effectiveDatePattern.FindStringSubmatch(strings.ToLower(raw))
	if len(matches) != 4 {
		return time.Time{}, fmt.Errorf("missing date in %q", raw)
	}

	day, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse day: %w", err)
	}
	year, err := strconv.Atoi(matches[3])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse year: %w", err)
	}

	month, ok := spanishMonths[matches[2]]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown month %q", matches[2])
	}

	effectiveDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if effectiveDate.Year() != year || effectiveDate.Month() != month || effectiveDate.Day() != day {
		return time.Time{}, fmt.Errorf("invalid date %q", raw)
	}
	return effectiveDate, nil
}

var spanishMonths = map[string]time.Month{
	"enero":      time.January,
	"febrero":    time.February,
	"marzo":      time.March,
	"abril":      time.April,
	"mayo":       time.May,
	"junio":      time.June,
	"julio":      time.July,
	"agosto":     time.August,
	"septiembre": time.September,
	"setiembre":  time.September,
	"octubre":    time.October,
	"noviembre":  time.November,
	"diciembre":  time.December,
}
