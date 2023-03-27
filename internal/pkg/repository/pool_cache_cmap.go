package repository

import (
	"context"
	"errors"
	"fmt"

	cmap "github.com/orcaman/concurrent-map"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

var ErrPoolNotFoundInCache = errors.New("can not get pool in cache")

type PoolCacheCMapRepository struct {
	poolMap cmap.ConcurrentMap

	// poolByExchangeMap looks like: { "dmm": { 0x111...: true, 0x222: true }, "uniswap": { 0x333...: true, 0x444: true } }
	poolByExchangeMap cmap.ConcurrentMap
}

func NewPoolCacheCMapRepository(poolMap cmap.ConcurrentMap, poolByExchangeMap cmap.ConcurrentMap) *PoolCacheCMapRepository {
	return &PoolCacheCMapRepository{
		poolMap:           poolMap,
		poolByExchangeMap: poolByExchangeMap,
	}
}

func (r *PoolCacheCMapRepository) Keys(_ context.Context) []string {
	return r.poolMap.Keys()
}

func (r *PoolCacheCMapRepository) Get(_ context.Context, address string) (entity.Pool, error) {
	pool, ok := r.poolMap.Get(address)
	if !ok {
		return entity.Pool{
			Address: address,
		}, ErrPoolNotFoundInCache
	}

	return pool.(entity.Pool), nil
}

// Set will set the pool with its address as ID.
// Note that this function will also cache the exchange from which the pool belongs to.
func (r *PoolCacheCMapRepository) Set(_ context.Context, address string, pool entity.Pool) error {
	r.poolMap.Set(address, pool)
	exchangePoolMap, ok := r.poolByExchangeMap.Get(pool.Exchange)

	if !ok {
		exchangePoolMap = cmap.New()
		r.poolByExchangeMap.Set(pool.Exchange, exchangePoolMap)

		logger.Infof("poolByExchangeMap added new dex: [%s]", pool.Exchange)
	}

	exchangePoolMap.(cmap.ConcurrentMap).Set(pool.Address, true)

	return nil
}

func (r *PoolCacheCMapRepository) Remove(_ context.Context, address string) error {
	r.poolMap.Remove(address)
	return nil
}

func (r *PoolCacheCMapRepository) Count(_ context.Context) int {
	return r.poolMap.Count()
}

// GetPoolIdsByExchange will get all pool ids belong to an exchange
func (r *PoolCacheCMapRepository) GetPoolIdsByExchange(_ context.Context, id string) []string {
	if poolIds, ok := r.poolByExchangeMap.Get(id); ok {

		if poolIds != nil {
			var ids []string

			poolIds.(cmap.ConcurrentMap).IterCb(
				func(key string, value interface{}) {
					ids = append(ids, key)
				},
			)

			return ids
		}
	}

	return []string{}
}

// GetPoolsByExchange will get all pools belong to an exchange
func (r *PoolCacheCMapRepository) GetPoolsByExchange(_ context.Context, id string) ([]entity.Pool, error) {
	if poolIds, ok := r.poolByExchangeMap.Get(id); ok {
		if poolIds != nil {
			pools := make([]entity.Pool, 0)

			poolIds.(cmap.ConcurrentMap).IterCb(
				func(key string, value interface{}) {
					pool, ok := r.poolMap.Get(key)
					if ok {
						pools = append(pools, pool.(entity.Pool))
					}
				},
			)

			return pools, nil
		}
	}

	return []entity.Pool{}, nil
}

func (r *PoolCacheCMapRepository) GetByAddresses(_ context.Context, ids []string) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0)

	for _, id := range ids {
		pool, ok := r.poolMap.Get(id)
		if !ok {
			return []entity.Pool{}, fmt.Errorf("%w: %s", ErrPoolNotFoundInCache, id)
		}

		pools = append(pools, pool.(entity.Pool))
	}

	return pools, nil
}

func (r *PoolCacheCMapRepository) IsPoolExist(ctx context.Context, address string) bool {
	_, err := r.Get(ctx, address)

	return !errors.Is(err, ErrPoolNotFoundInCache)
}
