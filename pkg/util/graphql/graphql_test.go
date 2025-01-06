package graphql

import (
	"context"
	"net/http"
	"testing"

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
			if req.Header == nil {
				req.Header = make(http.Header)
			}
			req.Header.Set("x-custom-header-1", "a")
			req.Header.Set("x-custom-header-2", "b")
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
			req.URL = "http://example.com"
			err := fn(ctx, req, resp)
			logs = append(logs, "After interceptor 4")
			return err
		},
		func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error {
			logs = append(logs, "Before interceptor 5")
			err := fn(ctx, req, resp)
			req.Header.Set("x-custom-header-3", "c")
			logs = append(logs, "After interceptor 5")
			return err
		},
	}...)

	req := &Request{}
	client := NewClient("", chainClientInterceptorOpt)
	_ = client.Run(context.Background(), req, nil)

	require.Equal(t, "a", req.Header.Get("x-custom-header-1"))
	require.Equal(t, "b", req.Header.Get("x-custom-header-2"))
	require.Equal(t, "c", req.Header.Get("x-custom-header-3"))
	require.Equal(t, "http://example.com", req.URL)
	require.Equal(t, len(expectedLogs), len(logs))
	for i, log := range logs {
		if log != expectedLogs[i] {
			t.Errorf("expected %s, got %s", expectedLogs[i], log)
		}
	}

}
