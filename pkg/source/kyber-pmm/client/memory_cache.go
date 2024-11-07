package client

import (
	"context"
	"errors"

	"github.com/KyberNetwork/logger"
	"github.com/dgraph-io/ristretto"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
)

const (
	defaultNumCounts   = 5000
	defaultMaxCost     = 500
	defaultBufferItems = 64

	defaultSingleItemCost = 1

	cacheKeyTokens      = "tokens"
	cacheKeyPairs       = "pairs"
	cacheKeyPriceLevels = "price-levels"
)

type memoryCacheClient struct {
	config         *kyberpmm.MemoryCacheConfig
	cache          *ristretto.Cache
	fallbackClient kyberpmm.IClient
}

func NewMemoryCacheClient(
	config *kyberpmm.MemoryCacheConfig,
	fallbackClient kyberpmm.IClient,
) *memoryCacheClient {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: defaultNumCounts,
		MaxCost:     defaultMaxCost,
		BufferItems: defaultBufferItems,
	})
	if err != nil {
		logger.Errorf("failed to init memory cache, err %v", err.Error())
	}

	return &memoryCacheClient{
		config:         config,
		cache:          cache,
		fallbackClient: fallbackClient,
	}
}

func (c *memoryCacheClient) ListTokens(ctx context.Context) (map[string]kyberpmm.TokenItem, error) {
	cachedTokens, err := c.listTokensFromCache()
	if err == nil {
		return cachedTokens, nil
	}

	// Cache missed. Using fallbackClient
	tokens, err := c.fallbackClient.ListTokens(ctx)
	if err != nil {
		return nil, err
	}

	if err = c.saveTokensToCache(tokens); err != nil {
		logger.
			WithFields(logger.Fields{"error": err}).
			Warn("memory cache failed")
	}

	return tokens, err
}

// listTokensFromCache only returns if tokens are able to fetch from cache
func (c *memoryCacheClient) listTokensFromCache() (map[string]kyberpmm.TokenItem, error) {
	cachedTokens, found := c.cache.Get(cacheKeyTokens)
	if !found {
		return nil, errors.New("no tokens data in cache")
	}

	return cachedTokens.(map[string]kyberpmm.TokenItem), nil
}

func (c *memoryCacheClient) saveTokensToCache(tokens map[string]kyberpmm.TokenItem) error {
	c.cache.SetWithTTL(cacheKeyTokens, tokens, defaultSingleItemCost, c.config.TTL.Tokens.Duration)
	c.cache.Wait()

	return nil
}

func (c *memoryCacheClient) ListPairs(ctx context.Context) (map[string]kyberpmm.PairItem, error) {
	cachedPairs, err := c.listPairsFromCache()
	if err == nil {
		return cachedPairs, nil
	}

	// Cache missed. Using fallbackClient
	pairs, err := c.fallbackClient.ListPairs(ctx)
	if err != nil {
		return nil, err
	}

	if err = c.savePairsToCache(pairs); err != nil {
		logger.
			WithFields(logger.Fields{"error": err}).
			Warn("memory cache failed")
	}

	return pairs, err
}

// listPairsFromCache only returns if pairs are able to fetch from cache
func (c *memoryCacheClient) listPairsFromCache() (map[string]kyberpmm.PairItem, error) {
	cachedPairs, found := c.cache.Get(cacheKeyPairs)
	if !found {
		return nil, errors.New("no pairs data in cache")
	}

	return cachedPairs.(map[string]kyberpmm.PairItem), nil
}

func (c *memoryCacheClient) savePairsToCache(tokens map[string]kyberpmm.PairItem) error {
	c.cache.SetWithTTL(cacheKeyPairs, tokens, defaultSingleItemCost, c.config.TTL.Pairs.Duration)
	c.cache.Wait()

	return nil
}

func (c *memoryCacheClient) ListPriceLevels(ctx context.Context) (kyberpmm.ListPriceLevelsResult, error) {
	cachedPriceLevels, err := c.listPriceLevelsFromCache()
	if err == nil {
		return cachedPriceLevels, nil
	}

	// Cache missed. Using fallbackClient
	priceLevels, err := c.fallbackClient.ListPriceLevels(ctx)
	if err != nil {
		return kyberpmm.ListPriceLevelsResult{}, err
	}

	if err = c.savePriceLevelsToCache(priceLevels); err != nil {
		logger.
			WithFields(logger.Fields{"error": err}).
			Warn("memory cache failed")
	}

	return priceLevels, err
}

// listPriceLevelsFromCache only returns if price levels are able to fetch from cache
func (c *memoryCacheClient) listPriceLevelsFromCache() (kyberpmm.ListPriceLevelsResult, error) {
	cachedPriceLevels, found := c.cache.Get(cacheKeyPriceLevels)
	if !found {
		return kyberpmm.ListPriceLevelsResult{}, errors.New("no price levels data in cache")
	}

	return cachedPriceLevels.(kyberpmm.ListPriceLevelsResult), nil
}

func (c *memoryCacheClient) savePriceLevelsToCache(priceLevelsAndInventory kyberpmm.ListPriceLevelsResult) error {
	c.cache.SetWithTTL(cacheKeyPriceLevels, priceLevelsAndInventory, defaultSingleItemCost, c.config.TTL.PriceLevels.Duration)
	c.cache.Wait()

	return nil
}

func (c *memoryCacheClient) Firm(ctx context.Context, params kyberpmm.FirmRequestParams) (kyberpmm.FirmResult, error) {
	return c.fallbackClient.Firm(ctx, params)
}
