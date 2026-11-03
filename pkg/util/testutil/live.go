package testutil

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// Multicall3 is Multicall3's address, the same on every chain it is deployed on.
var Multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// LiveURL returns the endpoint env names for a test that reads a live chain, and skips t when CI
// is set or env is not.
func LiveURL(t testing.TB, env string) string {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	url := os.Getenv(env)
	if url == "" {
		t.Skipf("set %s to run this", env)
	}
	return url
}

// LiveClient is a live test's connection to an endpoint, which counts the requests it sends.
type LiveClient struct {
	RPC    *rpc.Client
	Eth    *ethclient.Client
	Ethrpc *ethrpc.Client // with Multicall3 as its multicall contract

	requests atomic.Int64
}

// Requests is the number of HTTP requests sent so far, each once however often a rate limit made
// it go again.
func (c *LiveClient) Requests() int64 {
	return c.requests.Load()
}

// DialLive connects a live test to url, sending again a request a rate limit refused (see
// RetryOn429).
func DialLive(t testing.TB, url string) *LiveClient {
	t.Helper()
	return DialLiveVia(t, url, RetryOn429{})
}

// DialLiveVia is DialLive over transport, which sends each request the count passes on.
func DialLiveVia(t testing.TB, url string, transport http.RoundTripper) *LiveClient {
	t.Helper()
	c := &LiveClient{}
	count := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		c.requests.Add(1)
		return transport.RoundTrip(req)
	})
	var err error
	c.RPC, err = rpc.DialOptions(t.Context(), url, rpc.WithHTTPClient(&http.Client{Transport: count}))
	require.NoError(t, err)
	t.Cleanup(c.RPC.Close)
	c.Eth = ethclient.NewClient(c.RPC)
	c.Ethrpc = ethrpc.NewWithClient(c.Eth).SetMulticallContract(Multicall3)
	return c
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// RetryOn429 sends again, after a pause that doubles from 250ms to 8s, a request refused with 429,
// for up to a minute: a shared endpoint's rate limit would otherwise fail a live test.
type RetryOn429 struct{}

func (r RetryOn429) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
	}
	deadline := time.Now().Add(time.Minute)
	for wait := 250 * time.Millisecond; ; wait = min(2*wait, 8*time.Second) {
		req.Body = io.NopCloser(bytes.NewReader(body))
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || time.Now().Add(wait).After(deadline) {
			return resp, err
		}
		_ = resp.Body.Close()
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}
}
