package graphql

import (
	"context"
)

type ClientFunc func(ctx context.Context, req *Request, resp interface{}) error

func (fn ClientFunc) Invoke(ctx context.Context, req *Request, resp interface{}) error {
	return fn(ctx, req, resp)
}

type Interceptor interface {
	Next(fn IClient) IClient
}

type InterceptorFunc func(client IClient) IClient

func (fn InterceptorFunc) Next(client IClient) IClient { return fn(client) }

func ComposeInvokeInterceptor(client IClient, interceptors ...Interceptor) IClient {
	for _, interceptor := range interceptors {
		client = interceptor.Next(client)
	}
	return client
}
