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
}

func (f *FakeHTTPDoer) Do(*http.Request) (*http.Response, error) {
	if f.Error != nil {
		return nil, f.Error
	}

	resp := &http.Response{
		StatusCode: f.StatusCode,
		Body:       io.NopCloser(strings.NewReader(f.Body)),
	}

	return resp, nil
}
