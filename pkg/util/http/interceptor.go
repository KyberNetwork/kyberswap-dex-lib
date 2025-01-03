package http

import (
	"net/http"
)

// RoundTripperFunc implement http.RoundTripper for convenient usage.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

// Interceptor is interceptor that can do more work before/after a request
type Interceptor interface {
	Next(fn http.RoundTripper) http.RoundTripper
}

// InterceptorFunc implement Interceptor for convenient usage.
type InterceptorFunc func(rt http.RoundTripper) http.RoundTripper

func (fn InterceptorFunc) Next(rt http.RoundTripper) http.RoundTripper { return fn(rt) }

// ComposeInterceptor compose interceptors to given http.RoundTripper
func ComposeInterceptor(rt http.RoundTripper, interceptors ...Interceptor) http.RoundTripper {
	for _, interceptor := range interceptors {
		rt = interceptor.Next(rt)
	}
	return rt
}
