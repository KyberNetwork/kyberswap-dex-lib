package graphql

import (
	httppkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/http"
	"github.com/machinebox/graphql"
	"net/http"
	"time"
)

const (
	DefaultGraphQLRequestTimeout = 20 * time.Second // default graphql client's timeout if not configured
)

// Config specifies config for creating a new graphql Client. See New.
type Config struct {
	Url     string
	Header  map[string][]string
	Timeout time.Duration
}

// New creates a graphql Client with provided config, allowing for adding timeout, default headers...
func New(cfg Config) *graphql.Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultGraphQLRequestTimeout
	}

	// Initialize graphql client with custom HTTP client (use custom timeout instead of 0)
	// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	if len(cfg.Header) > 0 {
		httpClient.Transport = &TransportWithDefaultHeaders{
			Transport: http.DefaultTransport,
			Headers:   cfg.Header,
		}
	}

	return graphql.NewClient(cfg.Url, graphql.WithHTTPClient(httpClient))
}

// NewWithTimeout creates a graphql Client with provided url and timeout.
// Deprecated: use New instead
func NewWithTimeout(url string, timeout time.Duration) *graphql.Client {
	return New(Config{
		Url:     url,
		Timeout: timeout,
	})
}

func NewGraphQLClient(cfg Config, interceptors ...any) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultGraphQLRequestTimeout
	}

	httpInterceptors, graphqlInterceptors := filterInterceptors(interceptors)

	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &TransportWithDefaultHeaders{
			Transport: http.DefaultTransport,
			Headers:   cfg.Header,
		},
	}
	httpClient.Transport = httppkg.ComposeInterceptor(httpClient.Transport, httpInterceptors...)

	return NewClient(
		cfg.Url,
		WithHTTPClient(httpClient), WithRunInterceptors(graphqlInterceptors...),
	)
}

func filterInterceptors(interceptors []interface{}) ([]httppkg.Interceptor, []Interceptor) {
	httpInterceptors := make([]httppkg.Interceptor, 0)
	graphqlInterceptors := make([]Interceptor, 0)
	for _, interceptor := range interceptors {
		switch i := interceptor.(type) {
		case httppkg.Interceptor:
			httpInterceptors = append(httpInterceptors, i)
		case Interceptor:
			graphqlInterceptors = append(graphqlInterceptors, i)
		}
	}
	return httpInterceptors, graphqlInterceptors
}
