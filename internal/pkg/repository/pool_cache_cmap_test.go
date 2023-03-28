package repository

import (
	"context"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/stretchr/testify/assert"
)

func TestPoolCacheCMapRepository_Keys(t *testing.T) {
	t.Parallel()

	t.Run("it should return correct keys", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()
		poolMapCache.Set("key1", 1)
		poolMapCache.Set("key2", 2)

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		assert.ElementsMatch(t, repo.Keys(ctx), []string{"key1", "key2"})
	})
}

func TestPoolCacheCMapRepository_Get(t *testing.T) {
	t.Parallel()

	t.Run("it should return ErrPoolNotFoundInCache when pool not found in cache", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()
		poolMapCache.Set("address1", entity.Pool{Address: "address1", ReserveUsd: 1234})

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		pool, err := repo.Get(ctx, "address2")

		assert.ErrorIs(t, err, ErrPoolNotFoundInCache)
		assert.Equal(t, entity.Pool{Address: "address2"}, pool)
	})

	t.Run("it should return correct pool when pool is found in cache", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()
		poolMapCache.Set("address1", entity.Pool{Address: "address1", ReserveUsd: 1234})

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		pool, err := repo.Get(ctx, "address1")

		assert.Nil(t, err)
		assert.Equal(t, entity.Pool{Address: "address1", ReserveUsd: 1234}, pool)
	})
}

func TestPoolCacheCMapRepository_Set(t *testing.T) {
	t.Parallel()

	t.Run("it should set to the cache correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)
		err := repo.Set(ctx, "address1", entity.Pool{
			Address:    "address1",
			Exchange:   "dmm",
			ReserveUsd: 1234,
		})

		assert.Nil(t, err)

		pool, ok := poolMapCache.Get("address1")

		assert.True(t, ok)
		assert.EqualValues(t, entity.Pool{Address: "address1", Exchange: "dmm", ReserveUsd: 1234}, pool.(entity.Pool))

		dmmPoolMap, ok := poolByExchangeMapCache.Get("dmm")

		assert.Equal(t, true, ok)
		result, ok := dmmPoolMap.(cmap.ConcurrentMap).Get("address1")

		assert.Equal(t, true, ok)
		assert.Equal(t, true, result)
	})
}

func TestPoolCacheCMapRepository_Remove(t *testing.T) {
	t.Parallel()

	t.Run("it should remove correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()
		poolMapCache.Set("address1", entity.Pool{Address: "address1", ReserveUsd: 1234})

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)
		err := repo.Remove(ctx, "address1")

		assert.Nil(t, err)

		_, ok := poolMapCache.Get("address1")

		assert.False(t, ok)
	})
}

func TestPoolCacheCMapRepository_Count(t *testing.T) {
	t.Parallel()

	t.Run("it should return number of keys correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()
		poolMapCache.Set("address1", entity.Pool{Address: "address1", ReserveUsd: 1234})

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)
		count := repo.Count(ctx)

		assert.Equal(t, 1, count)
	})
}

func TestPoolCacheCMapRepository_GetPoolIdsByExchange(t *testing.T) {
	t.Parallel()

	t.Run("it should get pool ids by exchange correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		_ = repo.Set(ctx, "address1", entity.Pool{Address: "address1", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address2", entity.Pool{Address: "address2", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address3", entity.Pool{Address: "address3", Exchange: "dmm", ReserveUsd: 1234})

		poolIds := repo.GetPoolIdsByExchange(ctx, "dmm")

		assert.ElementsMatch(t, []string{"address1", "address2", "address3"}, poolIds)
	})
}

func TestPoolCacheCMapRepository_GetPoolsByExchange(t *testing.T) {
	t.Parallel()

	t.Run("it should get pools by exchange correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		_ = repo.Set(ctx, "address1", entity.Pool{Address: "address1", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address2", entity.Pool{Address: "address2", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address3", entity.Pool{Address: "address3", Exchange: "dmm", ReserveUsd: 1234})

		pools, err := repo.GetPoolsByExchange(ctx, "dmm")

		assert.Nil(t, err)
		assert.Equal(t, 3, len(pools))
		assert.ElementsMatch(t, []entity.Pool{
			{Address: "address1", Exchange: "dmm", ReserveUsd: 1234},
			{Address: "address2", Exchange: "dmm", ReserveUsd: 1234},
			{Address: "address3", Exchange: "dmm", ReserveUsd: 1234},
		}, pools)
	})
}

func TestPoolCacheCMapRepository_GetByAddresses(t *testing.T) {
	t.Parallel()

	t.Run("it should get pools by exchange correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		_ = repo.Set(ctx, "address1", entity.Pool{Address: "address1", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address2", entity.Pool{Address: "address2", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address3", entity.Pool{Address: "address3", Exchange: "dmm", ReserveUsd: 1234})

		pools, err := repo.GetByAddresses(ctx, []string{"address1", "address2"})

		assert.Nil(t, err)
		assert.Equal(t, 2, len(pools))
		assert.ElementsMatch(t, []entity.Pool{
			{Address: "address1", Exchange: "dmm", ReserveUsd: 1234},
			{Address: "address2", Exchange: "dmm", ReserveUsd: 1234},
		}, pools)
	})
}

func TestPoolCacheCMapRepository_IsPoolExist(t *testing.T) {
	t.Parallel()

	t.Run("it should get pools by exchange correctly", func(t *testing.T) {
		ctx := context.Background()
		poolMapCache := cmap.New()
		poolByExchangeMapCache := cmap.New()

		repo := NewPoolCacheCMapRepository(poolMapCache, poolByExchangeMapCache)

		_ = repo.Set(ctx, "address1", entity.Pool{Address: "address1", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address2", entity.Pool{Address: "address2", Exchange: "dmm", ReserveUsd: 1234})
		_ = repo.Set(ctx, "address3", entity.Pool{Address: "address3", Exchange: "dmm", ReserveUsd: 1234})

		assert.Equal(t, true, repo.IsPoolExist(ctx, "address1"))
		assert.Equal(t, true, repo.IsPoolExist(ctx, "address2"))
		assert.Equal(t, false, repo.IsPoolExist(ctx, "address4"))
		assert.Equal(t, false, repo.IsPoolExist(ctx, "address5"))
	})
}
