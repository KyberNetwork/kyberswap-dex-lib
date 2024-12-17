package aevmclient

import (
	"context"
	"fmt"
	"sync/atomic"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	aevmtypes "github.com/KyberNetwork/aevm/types"
)

type FixedPooledClient struct {
	pool     []aevmclient.Client
	curIndex atomic.Uint64
}

func NewFixedPooledClient(n int, serverURL string, makeClientFunc MakeClient) (*FixedPooledClient, error) {
	var pool []aevmclient.Client
	for i := 0; i < n; i++ {
		client, err := makeClientFunc(serverURL)
		if err != nil {
			return nil, fmt.Errorf("could not make client: %w", err)
		}
		pool = append(pool, client)
	}
	return &FixedPooledClient{pool: pool}, nil
}

func (c *FixedPooledClient) nextClient() aevmclient.Client {
	index := int((c.curIndex.Add(1) - 1) % uint64(len(c.pool)))
	return c.pool[index]
}

func (c *FixedPooledClient) LatestStateRoot(ctx context.Context) (aevmcommon.Hash, error) {
	return c.nextClient().LatestStateRoot(ctx)
}

func (c *FixedPooledClient) SingleCall(ctx context.Context, req *aevmtypes.SingleCallParams) (*aevmtypes.CallResult, error) {
	return c.nextClient().SingleCall(ctx, req)
}

func (c *FixedPooledClient) MultipleCall(ctx context.Context, req *aevmtypes.MultipleCallParams) (*aevmtypes.MultipleCallResult, error) {
	return c.nextClient().MultipleCall(ctx, req)
}

func (c *FixedPooledClient) StorePreparedPools(ctx context.Context, req *aevmtypes.StorePreparedPoolsParams) (*aevmtypes.StorePreparedPoolsResult, error) {
	return c.nextClient().StorePreparedPools(ctx, req)
}

func (c *FixedPooledClient) FindRoute(ctx context.Context, req *aevmtypes.FindRouteParams) (*aevmtypes.FindRouteResult, error) {
	return c.nextClient().FindRoute(ctx, req)
}

func (c *FixedPooledClient) Close() {
	for _, client := range c.pool {
		if closer, ok := client.(Closer); ok {
			closer.Close()
		}
	}
}
