package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"
	"github.com/dgraph-io/ristretto"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/hashflow"
)

const (
	defaultNumCounts   = 5000
	defaultMaxCost     = 500
	defaultBufferItems = 64

	defaultSingleItemCost = 1
)

type memoryCacheClient struct {
	config         *MemoryCacheConfig
	cache          *ristretto.Cache
	fallbackClient IClient
}

func NewMemoryCacheClient(
	config *MemoryCacheConfig,
	fallbackClient IClient,
) *memoryCacheClient {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: defaultNumCounts,
		MaxCost:     defaultMaxCost,
		BufferItems: defaultBufferItems,
	})
	if err != nil {
		logger.Errorf("failed to init memory cache. err %v", err.Error())
	}

	return &memoryCacheClient{
		config:         config,
		cache:          cache,
		fallbackClient: fallbackClient,
	}
}

func (c *memoryCacheClient) ListMarketMakers(ctx context.Context) ([]string, error) {
	return c.fallbackClient.ListMarketMakers(ctx)
}

func (c *memoryCacheClient) ListPriceLevels(ctx context.Context, marketMakers []string) ([]hashflow.Pair, error) {
	pairs, err := c.listPairsFromCache(marketMakers)
	if err == nil {
		return pairs, nil
	}

	// Cache missed. Using fallbackClient
	pairs, err = c.fallbackClient.ListPriceLevels(ctx, marketMakers)

	// Save pairs data by market makers
	pairsByMarketMaker := map[string][]hashflow.Pair{}
	for _, pair := range pairs {
		if _, ok := pairsByMarketMaker[pair.MarketMaker]; !ok {
			pairsByMarketMaker[pair.MarketMaker] = []hashflow.Pair{}
		}
		pairsByMarketMaker[pair.MarketMaker] = append(pairsByMarketMaker[pair.MarketMaker], pair)
	}
	for marketMaker, pairs := range pairsByMarketMaker {
		if err = c.savePairsByMarketMakerToCache(marketMaker, pairs); err != nil {
			logger.
				WithFields(logger.Fields{"error": err, "marketMaker": marketMaker}).
				Warn("memory cache failed")
		}
	}

	return pairs, err
}

// listPairsFromCache only returns if all market makers' pairs are able to fetch from cache
func (c *memoryCacheClient) listPairsFromCache(marketMakers []string) ([]hashflow.Pair, error) {
	var pairs []hashflow.Pair

	for _, marketMaker := range marketMakers {
		marketMakerPairs, found := c.cache.Get(marketMaker)
		if !found {
			return []hashflow.Pair{}, errors.New("no data in cache")
		}

		pairs = append(pairs, marketMakerPairs.([]hashflow.Pair)...)
	}

	return pairs, nil
}

func (c *memoryCacheClient) savePairsByMarketMakerToCache(marketMaker string, pairs []hashflow.Pair) error {
	c.cache.SetWithTTL(marketMaker, pairs, defaultSingleItemCost, c.config.TTL.Duration)
	c.cache.Wait()

	return nil
}
