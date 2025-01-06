package http

import (
	"net/http"
)

type RoundTripFunc func(req *http.Request) (*http.Response, error)

type RoundTripperInterceptor func(req *http.Request, fn RoundTripFunc) (*http.Response, error)

type TransportWithOptions struct {
	Transport http.RoundTripper

	chainedRoundTripperInt       RoundTripperInterceptor
	chainRoundTripperInterceptor []RoundTripperInterceptor
}

func NewTransportWithOptions(transport http.RoundTripper, opts ...RoundTripperOption) *TransportWithOptions {
	t := &TransportWithOptions{
		Transport: transport,
	}
	for _, opt := range opts {
		opt(t)
	}
	chainRoundTripperInterceptors(t)
	return t
}

func (t TransportWithOptions) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.chainedRoundTripperInt != nil {
		return t.chainedRoundTripperInt(req, t.Transport.RoundTrip)
	}
	return t.Transport.RoundTrip(req)
}

func WithChainRoundTripperInterceptor(interceptors ...RoundTripperInterceptor) RoundTripperOption {
	return func(transport *TransportWithOptions) {
		transport.chainRoundTripperInterceptor = append(transport.chainRoundTripperInterceptor, interceptors...)
	}
}

// chainRoundTripperInterceptors chains the interceptors to the transport
func chainRoundTripperInterceptors(transport *TransportWithOptions) {
	interceptors := transport.chainRoundTripperInterceptor

	if transport.chainedRoundTripperInt != nil {
		interceptors = append([]RoundTripperInterceptor{transport.chainedRoundTripperInt}, interceptors...)
	}

	var chainedInt RoundTripperInterceptor
	if len(interceptors) == 0 {
		chainedInt = nil
	} else if len(interceptors) == 1 {
		chainedInt = interceptors[0]
	} else {
		chainedInt = func(req *http.Request, fn RoundTripFunc) (*http.Response, error) {
			return interceptors[0](req, getChainRoundTripFunc(interceptors, 0, fn))
		}
	}
	transport.chainedRoundTripperInt = chainedInt
}

// getChainRoundTripFunc recursively generate chained RoundTripFunc
func getChainRoundTripFunc(interceptors []RoundTripperInterceptor, curr int, finalFn RoundTripFunc) RoundTripFunc {
	if curr == len(interceptors)-1 {
		return finalFn
	}
	return func(req *http.Request) (*http.Response, error) {
		return interceptors[curr+1](req, getChainRoundTripFunc(interceptors, curr+1, finalFn))
	}
}

type RoundTripperOption func(*TransportWithOptions)
