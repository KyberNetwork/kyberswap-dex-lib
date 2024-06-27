package aevmclient

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	aevmcommon "github.com/KyberNetwork/aevm/common"
	aevmtypes "github.com/KyberNetwork/aevm/types"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Closer interface {
	Close()
}

type MakeClient = func(url string) (aevmclient.Client, error)

type Client struct {
	cfg                     Config
	makeClientFunc          MakeClient
	clients                 []aevmclient.Client // clients[i]'s URL = cfg.ServerURLs[i]
	clientWg                []*sync.WaitGroup   // len(clients) = len(clientWg)
	publishingClientIndexes []int               // 0 <= publishingClientIndexes[i] < len(clients)
	curIndex                atomic.Uint64
	lock                    sync.RWMutex // lock for mutating clients list
}

func NewClient(cfg Config, makeClientFunc MakeClient) (*Client, error) {
	// unique ServerURLs
	serverURLsSet := sets.NewString(cfg.ServerURLs...)
	cfg.ServerURLs = serverURLsSet.List()
	var publishingIndexes []int
	if len(cfg.PublishingPoolsURLs) == 0 {
		publishingIndexes = make([]int, serverURLsSet.Len())
		for i := 0; i < serverURLsSet.Len(); i++ {
			publishingIndexes[i] = i
		}
	} else {
		for _, url := range cfg.PublishingPoolsURLs {
			if !serverURLsSet.Has(url) {
				continue
			}
			publishingIndexes = append(publishingIndexes, slices.Index(cfg.ServerURLs, url))
		}
	}
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
	return &Client{
		cfg:                     cfg,
		makeClientFunc:          makeClientFunc,
		clients:                 clients,
		clientWg:                clientWg,
		publishingClientIndexes: publishingIndexes,
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

func (c *Client) LatestStateRoot(ctx context.Context) (aevmcommon.Hash, error) {
	client, done := c.accquireNextClient()
	defer done()

	return client.LatestStateRoot(ctx)
}

func (c *Client) SingleCall(ctx context.Context, req *aevmtypes.SingleCallParams) (*aevmtypes.CallResult, error) {
	client, done := c.accquireNextClient()
	defer done()

	return client.SingleCall(ctx, req)
}

func (c *Client) MultipleCall(ctx context.Context, req *aevmtypes.MultipleCallParams) (*aevmtypes.MultipleCallResult, error) {
	client, done := c.accquireNextClient()
	defer done()

	return client.MultipleCall(ctx, req)
}

func (c *Client) StorePreparedPools(ctx context.Context, req *aevmtypes.StorePreparedPoolsParams) (*aevmtypes.StorePreparedPoolsResult, error) {
	var (
		wg         errgroup.Group
		storageIDs = make([]string, len(c.publishingClientIndexes))
	)
	for _i, _index := range c.publishingClientIndexes {
		i, index, client := _i, _index, c.clients[_index]
		wg.Go(func() error {
			start := time.Now()
			result, err := client.StorePreparedPools(ctx, &aevmtypes.StorePreparedPoolsParams{
				EncodedPools: req.EncodedPools,
			})
			took := time.Since(start)
			if err != nil {
				logger.Errorf(ctx, "[client %d] could not StorePreparedPools: %s", index, err)
				return err
			}
			logger.Infof(ctx, "[client %d] StorePreparedPools took = %s", index, took.String())
			storageIDs[i] = result.StorageID
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return nil, err
	}
	for _, id := range storageIDs {
		if storageIDs[0] != id {
			return nil, fmt.Errorf("storageIDs must be the same")
		}
	}
	return &aevmtypes.StorePreparedPoolsResult{StorageID: storageIDs[0]}, nil
}

func (c *Client) FindRoute(ctx context.Context, req *aevmtypes.FindRouteParams) (*aevmtypes.FindRouteResult, error) {
	client, done := c.accquireNextClient()
	defer done()

	return client.FindRoute(ctx, req)
}
