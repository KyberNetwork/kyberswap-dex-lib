package graphql

import (
	"net/http"
)

// TransportWithDefaultHeaders adds default headers to all HTTP requests.
type TransportWithDefaultHeaders struct {
	Transport http.RoundTripper
	Headers   http.Header
}

// RoundTrip adds default headers and executes a single HTTP transaction.
// The headers are only added if not already set in the *http.Request.
func (t *TransportWithDefaultHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	for key, values := range t.Headers {
		if len(req.Header.Values(key)) == 0 {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	return t.Transport.RoundTrip(req)
}
