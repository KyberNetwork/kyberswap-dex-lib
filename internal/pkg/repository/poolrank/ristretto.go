package poolrank

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/dgraph-io/ristretto"
)

const Cardinality = "indexCardinality"

type IFallbackRepository interface {
	AddToSortedSet(
		ctx context.Context,
		token0, token1 string,
		isToken0Whitelisted, isToken1Whitelisted bool,
		key string, memberName string, score float64,
		useGlobal bool,
	) error
	RemoveFromSortedSet(
		ctx context.Context,
		token0, token1 string,
		isToken0Whitelisted, isToken1Whitelisted bool,
		key string, memberName string, useGlobal bool,
	) error
	RemoveAddressesFromWhitelistIndex(ctx context.Context, key string, pools []string, removeFromGlobal bool) error
	GetDirectIndexLength(ctx context.Context, key, token0, token1 string) (int64, error)
	AddScoreToSortedSets(ctx context.Context, scores []entity.PoolScore) error
	RemoveScoreToSortedSets(ctx context.Context, scores []entity.PoolScore) error
	ZCard(ctx context.Context, keys []string) map[string]int64
	SaveCorrelatedPair(ctx context.Context, correlatedPairs []entity.CorrelatedPairInfo) error
}

type ristrettoRepository struct {
	cache              *ristretto.Cache
	fallbackRepository IFallbackRepository
	config             RistrettoConfig
}

func NewRistrettoRepository(
	fallbackRepository IFallbackRepository,
	config RistrettoConfig,
) (*ristrettoRepository, error) {

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
	})
	if err != nil {
		return nil, err
	}

	return &ristrettoRepository{
		cache:              cache,
		fallbackRepository: fallbackRepository,
		config:             config,
	}, nil
}

func (r *ristrettoRepository) AddToSortedSet(
	ctx context.Context,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
	key string, memberName string, score float64,
	useGlobal bool,
) error {
	return r.fallbackRepository.AddToSortedSet(ctx, token0, token1, isToken0Whitelisted, isToken1Whitelisted, key, memberName, score, useGlobal)
}

func (r *ristrettoRepository) RemoveFromSortedSet(
	ctx context.Context,
	token0, token1 string,
	isToken0Whitelisted, isToken1Whitelisted bool,
	key string, memberName string, useGlobal bool,
) error {
	return r.fallbackRepository.RemoveFromSortedSet(ctx, token0, token1, isToken0Whitelisted, isToken1Whitelisted, key, memberName, useGlobal)
}

func (r *ristrettoRepository) RemoveAddressesFromWhitelistIndex(ctx context.Context, key string, pools []string, removeFromGlobal bool) error {
	return r.fallbackRepository.RemoveAddressesFromWhitelistIndex(ctx, key, pools, removeFromGlobal)
}

func (r *ristrettoRepository) GetDirectIndexLength(ctx context.Context, key, token0, token1 string) (int64, error) {
	return r.fallbackRepository.GetDirectIndexLength(ctx, key, token0, token1)
}

func (r *ristrettoRepository) AddScoreToSortedSets(ctx context.Context, scores []entity.PoolScore) error {
	return r.fallbackRepository.AddScoreToSortedSets(ctx, scores)
}

func (r *ristrettoRepository) RemoveScoreToSortedSets(ctx context.Context, scores []entity.PoolScore) error {
	return r.fallbackRepository.RemoveScoreToSortedSets(ctx, scores)
}

func (r *ristrettoRepository) SaveCorrelatedPair(ctx context.Context, correlatedPairs []entity.CorrelatedPairInfo) error {
	return r.fallbackRepository.SaveCorrelatedPair(ctx, correlatedPairs)
}

func genCardinalityIndexKey(prefix, key string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, Cardinality, key)
}

func (r *ristrettoRepository) ZCard(ctx context.Context, keys []string) map[string]int64 {
	result := make(map[string]int64, len(keys))
	uncachedKeys := make([]string, 0, len(keys))

	for _, key := range keys {
		cachedCard, found := r.cache.Get(genCardinalityIndexKey(r.config.Prefix, key))
		if !found {
			uncachedKeys = append(uncachedKeys, key)
			continue
		}

		cardinality, ok := cachedCard.(int64)
		if !ok {
			uncachedKeys = append(uncachedKeys, key)
			continue
		}

		result[key] = cardinality
	}

	if len(uncachedKeys) == 0 {
		return result
	}

	uncachedCards := r.fallbackRepository.ZCard(ctx, uncachedKeys)
	if len(uncachedCards) == 0 {
		return result
	}

	for key, card := range uncachedCards {
		r.cache.SetWithTTL(genCardinalityIndexKey(r.config.Prefix, key), card, r.config.IndexCardinality.Cost, r.config.IndexCardinality.TTL)
		result[key] = card
	}

	return result
}
