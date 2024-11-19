package aevmclient

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	aevmtypes "github.com/KyberNetwork/aevm/types"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Closer interface {
	Close()
}

type MakeClient = func(url string) (aevmclient.Client, error)

type Client struct {
	cfg               Config
	makeClientFunc    MakeClient
	clients           []aevmclient.Client // clients[i]'s URL = cfg.ServerURLs[i]
	clientWg          []*sync.WaitGroup   // len(clients) = len(clientWg)
	publishingClients []aevmclient.Client // publishingClients[i]'s URL = cfg.PublishingURLs[i]
	curIndex          atomic.Uint64
	lock              sync.RWMutex // lock for mutating clients list

	retryOnTimeout          time.Duration
	findrouteRetryOnTimeout time.Duration
}

func NewClient(cfg Config, makeClientFunc MakeClient) (*Client, error) {
	// unique ServerURLs
	cfg.ServerURLs = sets.NewString(cfg.ServerURLs...).List()
	clients := make([]aevmclient.Client, len(cfg.ServerURLs))
	clientWg := make([]*sync.WaitGroup, len(cfg.ServerURLs))
	for i, serverURL := range cfg.ServerURLs {
		client, err := makeClientFunc(serverURL)
		if err != nil {
			return nil, fmt.Errorf("could not make client: %w", err)
		}
		clients[i] = client
		clientWg[i] = new(sync.WaitGroup)
	}

	// unique PublishingPoolsURLs
	cfg.PublishingPoolsURLs = sets.NewString(cfg.PublishingPoolsURLs...).List()
	publishingClients := make([]aevmclient.Client, len(cfg.PublishingPoolsURLs))
	for i, serverURL := range cfg.PublishingPoolsURLs {
		client, err := makeClientFunc(serverURL)
		if err != nil {
			return nil, fmt.Errorf("could not make publishing client: %w", err)
		}
		publishingClients[i] = client
	}

	return &Client{
		cfg:               cfg,
		makeClientFunc:    makeClientFunc,
		clients:           clients,
		clientWg:          clientWg,
		publishingClients: publishingClients,

		retryOnTimeout:          time.Duration(cfg.RetryOnTimeoutMs) * time.Millisecond,
		findrouteRetryOnTimeout: time.Duration(cfg.FindrouteRetryOnTimeoutMs) * time.Millisecond,
	}, nil
}

func (c *Client) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, client := range c.clients {
		// we don't wait for client
		if closer, ok := client.(Closer); ok {
			closer.Close()
		}
	}
	for _, client := range c.publishingClients {
		if closer, ok := client.(Closer); ok {
			closer.Close()
		}
	}
}

func (c *Client) ApplyConfig(cfg Config) {
	if len(cfg.ServerURLs) == 0 {
		return
	}

	var (
		oldUrls          = sets.NewString(c.cfg.ServerURLs...)
		newUrls          = sets.NewString(cfg.ServerURLs...)
		removingClients  []aevmclient.Client
		removingClientWg []*sync.WaitGroup
	)

	c.lock.Lock()
	for i := 0; i < len(c.cfg.ServerURLs); {
		url := c.cfg.ServerURLs[i]
		if !newUrls.Has(url) {
			removingClients = append(removingClients, c.clients[i])
			removingClientWg = append(removingClientWg, c.clientWg[i])
			// remove clients[i] from clients
			c.clients = append(c.clients[:i], c.clients[i+1:]...)
			c.clientWg = append(c.clientWg[:i], c.clientWg[i+1:]...)
			c.cfg.ServerURLs = append(c.cfg.ServerURLs[:i], c.cfg.ServerURLs[i+1:]...)
		} else {
			i++
		}
	}
	for _, url := range cfg.ServerURLs {
		if !oldUrls.Has(url) {
			client, err := c.makeClientFunc(url)
			if err != nil {
				logger.Warnf(context.Background(), "could not make client: %s", err)
				continue
			}
			c.clients = append(c.clients, client)
			c.clientWg = append(c.clientWg, new(sync.WaitGroup))
			c.cfg.ServerURLs = append(c.cfg.ServerURLs, url)
		}
	}
	c.lock.Unlock()

	// now it is safe to close removingClients
	// we closing removingClients asynchronously to make ApplyConfig return fast
	go func() {
		for i, client := range removingClients {
			// wait for the client no longer being used
			removingClientWg[i].Wait()
			if closer, ok := client.(Closer); ok {
				closer.Close()
			}
		}
		logger.Infof(context.Background(), "[AEVMClientUsecase] Closed all unused clients")
	}()
}

func (c *Client) accquireNextClient() (client aevmclient.Client, done func()) {
	var wg *sync.WaitGroup

	// accquire read lock to safely get next index
	c.lock.RLock()
	nextIndex := c.curIndex.Add(1) - 1
	wrappedIndex := int(nextIndex) % len(c.clients)
	client, wg = c.clients[wrappedIndex], c.clientWg[wrappedIndex]
	// add to client's waitgroup to make sure client is safe from closing at ApplyConfig
	wg.Add(1)
	c.lock.RUnlock()

	done = func() { wg.Done() }
	return
}

func withRetry[R any](ctx context.Context, c *Client, onTimeout time.Duration, op func(context.Context, aevmclient.Client) (R, error)) (R, error) {
	var (
		result R
		err    error
		N      = c.cfg.MaxRetry + 1
	)
	for i := uint64(0); i < N; i++ {
		func() {
			client, done := c.accquireNextClient()
			defer done()

			subCtx := ctx
			if onTimeout != 0 {
				var cancel context.CancelFunc
				subCtx, cancel = context.WithTimeout(ctx, onTimeout)
				defer cancel()
			}
			result, err = op(subCtx, client)
		}()
		if err == nil {
			break
		}
	}
	return result, err
}

// LatestStateRoot returns the latest state root hash from AEVM
// It returns empty hash if error, so the consumer of this function should handle it accordingly
func (c *Client) LatestStateRoot(ctx context.Context) (aevmcommon.Hash, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[aevmclient] LatestStateRoot")
	defer span.End()

	hash, err := withRetry(ctx, c, c.retryOnTimeout, func(ctx context.Context, client aevmclient.Client) (aevmcommon.Hash, error) {
		return client.LatestStateRoot(ctx)
	})
	if err != nil {
		// return empty hash if error
		return aevmcommon.Hash{}, nil
	}

	return hash, nil
}

func (c *Client) SingleCall(ctx context.Context, req *aevmtypes.SingleCallParams) (*aevmtypes.CallResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[aevmclient] SingleCall")
	defer span.End()

	return withRetry(ctx, c, c.retryOnTimeout, func(ctx context.Context, client aevmclient.Client) (*aevmtypes.CallResult, error) {
		return client.SingleCall(ctx, req)
	})
}

func (c *Client) MultipleCall(ctx context.Context, req *aevmtypes.MultipleCallParams) (*aevmtypes.MultipleCallResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[aevmclient] MultipleCall")
	defer span.End()

	return withRetry(ctx, c, c.retryOnTimeout, func(ctx context.Context, client aevmclient.Client) (*aevmtypes.MultipleCallResult, error) {
		return client.MultipleCall(ctx, req)
	})
}

func (c *Client) StorePreparedPools(ctx context.Context, req *aevmtypes.StorePreparedPoolsParams) (*aevmtypes.StorePreparedPoolsResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[aevmclient] StorePreparedPools")
	defer span.End()

	var (
		wg         errgroup.Group
		storageIDs = make([]string, len(c.publishingClients))
	)
	for _i, _client := range c.publishingClients {
		i, client := _i, _client
		wg.Go(func() error {
			start := time.Now()
			result, err := client.StorePreparedPools(ctx, &aevmtypes.StorePreparedPoolsParams{
				EncodedPools: req.EncodedPools,
			})
			if err != nil {
				return fmt.Errorf("[publishing client] could not StorePreparedPools to client %s: %s", c.cfg.PublishingPoolsURLs[i], err)
			}
			logger.Infof(ctx, "[publishing client] StorePreparedPools to client %s took = %s", c.cfg.PublishingPoolsURLs[i], time.Since(start).String())
			storageIDs[i] = result.StorageID
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return nil, err
	}
	if len(storageIDs) != 0 {
		for _, id := range storageIDs {
			if storageIDs[0] != id {
				return nil, fmt.Errorf("storageIDs must be the same")
			}
		}
	}
	return &aevmtypes.StorePreparedPoolsResult{StorageID: storageIDs[0]}, nil
}

func (c *Client) FindRoute(ctx context.Context, req *aevmtypes.FindRouteParams) (*aevmtypes.FindRouteResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[aevmclient] FindRoute")
	defer span.End()

	return withRetry(ctx, c, c.findrouteRetryOnTimeout, func(ctx context.Context, client aevmclient.Client) (*aevmtypes.FindRouteResult, error) {
		return client.FindRoute(ctx, req)
	})
}
