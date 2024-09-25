package aevmclient

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/aevm/common"
	"github.com/KyberNetwork/aevm/types"
	aevmtypes "github.com/KyberNetwork/aevm/types"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/sets"
)

type testClient struct {
	t           *testing.T
	url         string
	reqDuration time.Duration
	closeCh     chan struct{}

	closedURLs chan string
}

func newTestClient(t *testing.T, url string, reqDuration time.Duration, closedURLs chan string) *testClient {
	return &testClient{
		t:           t,
		url:         url,
		reqDuration: reqDuration,
		closeCh:     make(chan struct{}),
		closedURLs:  closedURLs,
	}
}

func (c *testClient) LatestStateRoot(context.Context) (common.Hash, error) { panic("unreachable") }

func (c *testClient) SingleCall(context.Context, *types.SingleCallParams) (*types.CallResult, error) {
	panic("unreachable")
}

func (c *testClient) StorePreparedPools(context.Context,
	*aevmtypes.StorePreparedPoolsParams) (*aevmtypes.StorePreparedPoolsResult, error) {
	panic("unreachable")
}

func (c *testClient) FindRoute(context.Context, *aevmtypes.FindRouteParams) (*aevmtypes.FindRouteResult, error) {
	panic("unreachable")
}

func (c *testClient) MultipleCall(context.Context, *types.MultipleCallParams) (*types.MultipleCallResult, error) {
	select {
	case <-time.NewTimer(c.reqDuration).C:
	case <-c.closeCh:
		c.t.Fatalf("client must not be closed while using")
	}
	return nil, nil
}

func (c *testClient) Close() {
	close(c.closeCh)
	c.closedURLs <- c.url
}

func TestApplyConfig(t *testing.T) {
	var (
		closedURLsCh = make(chan string, 1000)
		reqDuration  = 1 * time.Second
	)

	c, err := NewClient(Config{
		ServerURLs: []string{"c0", "c1", "c2"},
	}, func(url string) (aevmclient.Client, error) {
		return newTestClient(t, url, reqDuration, closedURLsCh), nil
	})
	require.NoError(t, err)

	var (
		firstCalled      atomic.Bool
		waitForFirstCall = make(chan struct{})
	)
	for i := 0; i < 100; i++ {
		go func() {
			if firstCalled.CompareAndSwap(false, true) {
				close(waitForFirstCall)
			}
			c.MultipleCall(context.TODO(), nil)
		}()
	}

	// wait for fist call to make sure client's goroutine is scheduled
	<-waitForFirstCall

	c.ApplyConfig(Config{ServerURLs: []string{"c2", "c3", "c4"}})

	require.EqualValues(t, []string{"c2", "c3", "c4"}, c.cfg.ServerURLs)
	require.Equal(t, len(c.cfg.ServerURLs), len(c.clients))
	require.Equal(t, len(c.cfg.ServerURLs), len(c.clientWg))
	require.True(t, sets.NewString("c0", "c1").
		Equal(sets.NewString(<-closedURLsCh, <-closedURLsCh)))

	c.ApplyConfig(Config{ServerURLs: []string{"c2", "c3", "c4", "c99", "c100"}})
	require.EqualValues(t, []string{"c2", "c3", "c4", "c99", "c100"}, c.cfg.ServerURLs)
	require.Equal(t, len(c.cfg.ServerURLs), len(c.clients))
	require.Equal(t, len(c.cfg.ServerURLs), len(c.clientWg))

	select {
	case <-time.NewTimer(2 * reqDuration).C:
	case <-closedURLsCh:
		require.FailNow(t, "there must not be any closed client")
	}
}

func TestClientWithRetry(t *testing.T) {
	t.Skip()

	c, err := NewClient(Config{
		ServerURLs: []string{
			"localhost:8247",
		},
		RetryOnTimeoutMs: 100,
		MaxRetry:         3,
	}, func(url string) (aevmclient.Client, error) {
		return aevmclient.NewGRPCClient(url)
	})
	require.NoError(t, err)

	stateRoot, err := c.LatestStateRoot(context.Background())
	require.NoError(t, err)
	fmt.Printf("%s\n", stateRoot)
}
