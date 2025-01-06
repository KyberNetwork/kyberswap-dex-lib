package graphql

import (
	"context"
	"net/http"
	"testing"
	"time"

	httppkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/http"

	"github.com/stretchr/testify/require"
)

func TestChainClientInterceptor(t *testing.T) {
	expectedLogs := []string{
		"Before interceptor 1",

		"Before interceptor 2",
		"Before interceptor 3",
		"Before interceptor 4",
		"Before interceptor 5",
		"After interceptor 5",
		"After interceptor 4",
		"After interceptor 3",
		"After interceptor 2",

		"Before interceptor 2",
		"Before interceptor 3",
		"Before interceptor 4",
		"Before interceptor 5",
		"After interceptor 5",
		"After interceptor 4",
		"After interceptor 3",
		"After interceptor 2",

		"Before interceptor 2",
		"Before interceptor 3",
		"Before interceptor 4",
		"Before interceptor 5",
		"After interceptor 5",
		"After interceptor 4",
		"After interceptor 3",
		"After interceptor 2",

		"After interceptor 1",
	}

	var logs []string
	chainClientInterceptorOpt := WithChainClientInterceptor([]ClientInterceptor{
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 1")
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 1")
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			var err error
			for i := 0; i < 3; i++ {
				logs = append(logs, "Before interceptor 2")
				err = fn(ctx, req, resp)
				logs = append(logs, "After interceptor 2")
			}
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 3")
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 3")
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 4")
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 4")
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 5")
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 5")
			return err
		},
	}...)
	client := NewClient("", chainClientInterceptorOpt)
	_ = client.Run(context.Background(), &Request{}, nil)
	for i, log := range logs {
		if log != expectedLogs[i] {
			t.Errorf("expected %s, got %s", expectedLogs[i], log)
		}
	}
}

func TestWithHttpClientOptions(t *testing.T) {
	expectedLogs := []string{
		"Before interceptor 1",

		"Before interceptor 2",
		"Before interceptor 3",
		"Before http interceptor 1",
		"Before http interceptor 2",
		"After http interceptor 2",
		"After http interceptor 1",
		"After interceptor 3",
		"After interceptor 2",

		"Before interceptor 2",
		"Before interceptor 3",
		"Before http interceptor 1",
		"Before http interceptor 2",
		"After http interceptor 2",
		"After http interceptor 1",
		"After interceptor 3",
		"After interceptor 2",

		"Before interceptor 2",
		"Before interceptor 3",
		"Before http interceptor 1",
		"Before http interceptor 2",
		"After http interceptor 2",
		"After http interceptor 1",
		"After interceptor 3",
		"After interceptor 2",

		"After interceptor 1",
	}

	var logs []string

	httpClient := NewHttpClient(
		10*time.Second,
		httppkg.WithChainRoundTripperInterceptor([]httppkg.RoundTripperInterceptor{
			func(req *http.Request, fn httppkg.RoundTripFunc) (*http.Response, error) {
				logs = append(logs, "Before http interceptor 1")
				resp, err := fn(req)
				req.Header.Set("X-Custom-Header", "456")
				logs = append(logs, "After http interceptor 1")
				return resp, err
			},
			func(req *http.Request, fn httppkg.RoundTripFunc) (*http.Response, error) {
				logs = append(logs, "Before http interceptor 2")
				resp, err := fn(req)
				logs = append(logs, "After http interceptor 2")
				return resp, err
			},
		}...),
	)

	chainClientInterceptorOpt := WithChainClientInterceptor([]ClientInterceptor{
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 1")
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 1")
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			var err error
			for i := 0; i < 3; i++ {
				logs = append(logs, "Before interceptor 2")
				err = fn(ctx, req, resp)
				logs = append(logs, "After interceptor 2")
			}
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 3")
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 3")
			return err
		},
	}...)
	httpClientOpt := WithHTTPClient(httpClient)

	client := NewClient("", chainClientInterceptorOpt, httpClientOpt)
	graphqlRequest := &Request{}
	_ = client.Run(context.Background(), graphqlRequest, nil)
	require.Equal(t, len(logs), len(expectedLogs))
	for i, log := range logs {
		if log != expectedLogs[i] {
			t.Errorf("expected %s, got %s", expectedLogs[i], log)
		}
	}

	// Verify request headers are replaced by actual HTTP request headers after http.Do()
	// In this test case, the X-Custom-Header is set right after RoundTrip() to simulate the header being added,
	// but it is not actually sent
	require.Equal(t, "456", graphqlRequest.Header.Get("X-Custom-Header"))
}
