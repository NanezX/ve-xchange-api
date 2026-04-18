package provider

import (
	"io"
	"net/http"
	"strings"
)

type FakeHTTPDoer struct {
	StatusCode int
	Body       string
	Error      error
	DoFunc     func(*http.Request) (string, error)
}

func (f *FakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if f.DoFunc == nil {
		return &http.Response{StatusCode: f.StatusCode, Body: io.NopCloser(strings.NewReader(f.Body))}, nil
	}

	// Do func is to inject something specific
	body, err := f.DoFunc(req)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: f.StatusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}
