package graphql

import (
	"context"
	"testing"
)

func Test_graphqlInterceptors(t *testing.T) {
	expectedLogs := []string{
		"Before interceptor 3",

		"Before interceptor 2",
		"Before interceptor 1",
		"After interceptor 1",
		"After interceptor 2",

		"Before interceptor 2",
		"Before interceptor 1",
		"After interceptor 1",
		"After interceptor 2",

		"Before interceptor 2",
		"Before interceptor 1",
		"After interceptor 1",
		"After interceptor 2",

		"After interceptor 3",
	}

	var logs []string
	interceptors := []Interceptor{
		InterceptorFunc(func(client IClient) IClient {
			return ClientFunc(func(ctx context.Context, req *Request, resp interface{}) (err error, md *Metadata) {
				logs = append(logs, "Before interceptor 1")
				err, md = client.Invoke(ctx, req, resp)
				logs = append(logs, "After interceptor 1")
				return
			})
		}),
		InterceptorFunc(func(client IClient) IClient {
			return ClientFunc(func(ctx context.Context, req *Request, resp interface{}) (err error, md *Metadata) {
				for i := 0; i < 3; i++ {
					logs = append(logs, "Before interceptor 2")
					err, md = client.Invoke(ctx, req, resp)
					logs = append(logs, "After interceptor 2")
				}
				return
			})
		}),
		InterceptorFunc(func(client IClient) IClient {
			return ClientFunc(func(ctx context.Context, req *Request, resp interface{}) (err error, md *Metadata) {
				logs = append(logs, "Before interceptor 3")
				err, md = client.Invoke(ctx, req, resp)
				logs = append(logs, "After interceptor 3")
				return
			})
		}),
	}

	client := NewClient("", WithRunInterceptors(interceptors...))
	_, _ = client.Run(context.Background(), &Request{}, nil)
	for i, log := range logs {
		if log != expectedLogs[i] {
			t.Errorf("expected %s, got %s", expectedLogs[i], log)
		}
	}

}
