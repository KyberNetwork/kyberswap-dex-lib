package graphql

import (
	"net/http"
	"time"

	"github.com/machinebox/graphql"
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
