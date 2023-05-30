package client

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/hashflow"
	"github.com/KyberNetwork/logger"
)

type redisCacheClient struct {
	config         *RedisCacheConfig
	redisClient    redis.UniversalClient
	fallbackClient IClient
}

func NewRedisCacheClient(
	config *RedisCacheConfig,
	redisClient redis.UniversalClient,
	fallbackClient IClient,
) *redisCacheClient {
	return &redisCacheClient{
		config:         config,
		redisClient:    redisClient,
		fallbackClient: fallbackClient,
	}
}

func (c *redisCacheClient) ListMarketMakers(ctx context.Context) ([]string, error) {
	return c.fallbackClient.ListMarketMakers(ctx)
}

func (c *redisCacheClient) ListPriceLevels(ctx context.Context, marketMakers []string) ([]hashflow.Pair, error) {
	pairs, err := c.listPairsFromCache(ctx, marketMakers)
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
		if err = c.savePairsByMarketMakerToCache(ctx, marketMaker, pairs); err != nil {
			logger.
				WithFields(logger.Fields{"error": err, "marketMaker": marketMaker}).
				Warn("cache failed")
		}
	}

	return pairs, err
}

// listPairsFromCache only returns if all market makers' pairs are able to fetch from cache
func (c *redisCacheClient) listPairsFromCache(ctx context.Context, marketMakers []string) ([]hashflow.Pair, error) {
	var pairs []hashflow.Pair

	for _, marketMaker := range marketMakers {
		cacheKey := c.getCacheKey(marketMaker)
		cacheData, err := c.redisClient.Get(ctx, cacheKey).Result()
		if err != nil {
			return []hashflow.Pair{}, err
		}

		var marketMakerPairs []hashflow.Pair
		err = json.Unmarshal([]byte(cacheData), &marketMakerPairs)
		if err != nil {
			return []hashflow.Pair{}, err
		}

		pairs = append(pairs, marketMakerPairs...)
	}

	return pairs, nil
}

func (c *redisCacheClient) savePairsByMarketMakerToCache(ctx context.Context, marketMaker string, pairs []hashflow.Pair) error {
	cacheKey := c.getCacheKey(marketMaker)
	cacheData, err := json.Marshal(pairs)
	if err != nil {
		return err
	}
	return c.redisClient.Set(ctx, cacheKey, string(cacheData), c.config.TTL.Duration).Err()
}

func (c *redisCacheClient) getCacheKey(marketMarker string) string {
	return strings.Join([]string{
		c.config.Prefix,
		marketMarker,
	}, c.config.Separator)
}
