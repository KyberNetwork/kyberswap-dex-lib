package graphql

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
)

type RunFunc func(ctx context.Context, req *Request, resp interface{}) error

type ClientInterceptor func(ctx context.Context, req *Request, resp interface{}, fn RunFunc) error

// Client is a client for interacting with a GraphQL API.
type Client struct {
	endpoint    string
	restyClient *resty.Client

	chainedInt       ClientInterceptor
	chainInterceptor []ClientInterceptor

	Log func(format string, args ...interface{})
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint: endpoint,
		Log:      func(format string, args ...interface{}) {},
	}
	for _, optionFunc := range opts {
		optionFunc(c)
	}
	if c.restyClient == nil {
		c.restyClient = resty.New()
	}
	chainClientInterceptors(c)
	return c
}

func (c *Client) logf(format string, args ...interface{}) {
	c.Log(format, args...)
}

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned.
func (c *Client) Run(ctx context.Context, req *Request, resp interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if c.chainedInt != nil {
		return c.chainedInt(ctx, req, resp, c.run)
	}
	return c.run(ctx, req, resp)
}

func (c *Client) run(ctx context.Context, req *Request, resp interface{}) error {
	return c.runWithJSON(ctx, req, resp)
}

func (c *Client) runWithJSON(ctx context.Context, req *Request, resp interface{}) error {
	var requestBody bytes.Buffer
	requestBodyObj := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     req.q,
		Variables: req.vars,
	}
	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		return errors.Wrap(err, "encode body")
	}
	c.logf(">> variables: %v", req.vars)
	c.logf(">> query: %s", req.q)
	gr := &graphResponse{
		Data: resp,
	}
	endpoint := c.endpoint
	if req.URL != "" {
		endpoint = req.URL
	}
	r := c.restyClient.R().SetContext(ctx).SetBody(requestBodyObj).
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Accept", "application/json; charset=utf-8")
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
	c.logf(">> headers: %v", r.Header)
	res, err := r.Post(endpoint)
	if err != nil {
		return err
	}
	if err = c.restyClient.JSONUnmarshal(res.Body(), gr); err != nil {
		if res.StatusCode() != http.StatusOK {
			return fmt.Errorf("graphql: server returned a non-200 status code: %v", res.StatusCode())
		}
		return errors.Wrap(err, "decoding response")
	}

	if len(gr.Errors) > 0 {
		// return first error
		return gr.Errors[0]
	}
	return nil
}

// chainClientInterceptors chains all client interceptors into one
func chainClientInterceptors(client *Client) {
	interceptors := client.chainInterceptor

	if client.chainedInt != nil {
		interceptors = append([]ClientInterceptor{client.chainedInt}, interceptors...)
	}

	var chainedInt ClientInterceptor
	if len(interceptors) == 0 {
		chainedInt = nil
	} else if len(interceptors) == 1 {
		chainedInt = interceptors[0]
	} else {
		chainedInt = func(ctx context.Context, req *Request, resp interface{}, runFn RunFunc) error {
			return interceptors[0](ctx, req, resp, getChainRunFunc(interceptors, 0, runFn))
		}
	}
	client.chainedInt = chainedInt
}

// getChainRunFunc recursively generate chained RunFunc
func getChainRunFunc(interceptors []ClientInterceptor, curr int, finalFn RunFunc) RunFunc {
	if curr == len(interceptors)-1 {
		return finalFn
	}
	return func(ctx context.Context, req *Request, resp interface{}) error {
		return interceptors[curr+1](ctx, req, resp, getChainRunFunc(interceptors, curr+1, finalFn))
	}
}

func WithChainClientInterceptor(interceptors ...ClientInterceptor) ClientOption {
	return func(client *Client) {
		client.chainInterceptor = append(client.chainInterceptor, interceptors...)
	}
}

// WithRestyClient specifies the underlying resty.Client to use when
// making requests.
//
//	NewClient(endpoint, WithRestyClient(specificRestyClient))
func WithRestyClient(restyClient *resty.Client) ClientOption {
	return func(client *Client) {
		client.restyClient = restyClient
	}
}

// ClientOption are functions that are passed into NewClient to
// modify the behaviour of the Client.
type ClientOption func(*Client)

type graphErr struct {
	Message string
}

func (e graphErr) Error() string {
	return "graphql: " + e.Message
}

type graphResponse struct {
	Data   interface{}
	Errors []graphErr
}

// Request is a GraphQL request.
type Request struct {
	q    string
	vars map[string]interface{}

	// Header represent any request headers that will be set
	// when the request is made.
	Header http.Header

	// If the URL is not empty when the request is made,
	// it will be used instead of the client's endpoint.
	URL string
}

// NewRequest makes a new Request with the specified string.
func NewRequest(q string) *Request {
	req := &Request{
		q:      q,
		Header: make(map[string][]string),
	}
	return req
}

// Var sets a variable.
func (req *Request) Var(key string, value interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}
	req.vars[key] = value
}

// Vars gets the variables for this Request.
func (req *Request) Vars() map[string]interface{} {
	return req.vars
}

// Query gets the query string of this request.
func (req *Request) Query() string {
	return req.q
}
