package http

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func Test_httpInterceptors(t *testing.T) {
	type args struct {
		requestURL         string
		interceptors       []Interceptor
		expectedRequestURL string
		expectedHeaders    map[string]string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "it should update request URL and headers by recursive order",
			args: args{
				requestURL: "https://gateway.thegraph.com/api/$API_KEY/id",
				interceptors: []Interceptor{
					InterceptorFunc(func(rt http.RoundTripper) http.RoundTripper {
						return RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
							req.URL, _ = req.URL.Parse("https://gateway.thegraph.com/api/a/id")
							t.Log("Before interceptor 1")
							resp, err = rt.RoundTrip(req)
							t.Log("After interceptor 1")
							resp.Header.Set("X-Test-1", "a")
							return
						})
					}),
					InterceptorFunc(func(rt http.RoundTripper) http.RoundTripper {
						return RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
							req.URL, _ = req.URL.Parse("https://gateway.thegraph.com/api/b/id")
							t.Log("Before interceptor 2")
							resp, err = rt.RoundTrip(req)
							t.Log("After interceptor 2")
							resp.Header.Set("X-Test-2", "b")
							return
						})
					}),
					InterceptorFunc(func(rt http.RoundTripper) http.RoundTripper {
						return RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
							req.URL, _ = req.URL.Parse("https://gateway.thegraph.com/api/c/id")
							t.Log("Before interceptor 3")
							resp, err = rt.RoundTrip(req)
							t.Log("After interceptor 3")
							resp.Header.Set("X-Test-3", "c")
							return
						})
					}),
				},
				expectedRequestURL: "https://gateway.thegraph.com/api/a/id",
				expectedHeaders: map[string]string{
					"X-Test-1": "a",
					"X-Test-2": "b",
					"X-Test-3": "c",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chained := ComposeInterceptor(http.DefaultTransport, tt.args.interceptors...)
			require.NotNil(t, chained)
			req, _ := http.NewRequest(http.MethodGet, tt.args.requestURL, nil)
			resp, _ := chained.RoundTrip(req)
			assert.Equal(t, tt.args.expectedRequestURL, req.URL.String())
			if resp != nil && resp.Header != nil {
				for key, expectedValue := range tt.args.expectedHeaders {
					actualValue := resp.Header.Get(key)
					assert.Equal(t, expectedValue, actualValue)
				}
			}
		})
	}
}
