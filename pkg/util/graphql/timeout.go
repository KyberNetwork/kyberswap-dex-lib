package graphql

import (
	"net/http"
	"time"

	"github.com/machinebox/graphql"
	"github.com/samber/lo"
)

const (
	DefaultGraphQLRequestTimeout = 20 * time.Second
)

func NewWithTimeout(url string, timeout time.Duration) *graphql.Client {
	// Initialize graphql client with custom HTTP client (use custom timeout instead of 0)
	// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	httpClient := &http.Client{
		Timeout: lo.Ternary[time.Duration](timeout != 0, timeout, DefaultGraphQLRequestTimeout),
	}

	return graphql.NewClient(url, graphql.WithHTTPClient(httpClient))
}
