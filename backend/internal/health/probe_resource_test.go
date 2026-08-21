package health_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"minigate/internal/health"
)

type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

type responseTransport struct {
	body *trackingBody
}

func (t *responseTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Header:     make(http.Header),
		Body:       t.body,
	}, nil
}

func TestProbeClosesResponseBodyOnUnexpectedStatus(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("temporarily unavailable")}
	client := &http.Client{Transport: &responseTransport{body: body}}

	err := health.Probe(client, "http://upstream.example", "/health", http.StatusOK)
	if err == nil {
		t.Fatal("Probe returned nil error for an unavailable upstream")
	}
	if !body.closed {
		t.Fatal("Probe did not close the response body after an unexpected status")
	}
}
