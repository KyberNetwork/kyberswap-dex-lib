package repository

import (
	"context"
	"testing"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/stretchr/testify/assert"
)

func TestPriceCacheCMapRepository_Keys(t *testing.T) {
	t.Parallel()

	t.Run("it should return correct keys", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("key1", 1)
		cache.Set("key2", 2)

		repo := NewPriceCacheCMapRepository(cache)

		assert.ElementsMatch(t, repo.Keys(ctx), []string{"key1", "key2"})
	})
}

func TestPriceCacheCMapRepository_Get(t *testing.T) {
	t.Parallel()

	t.Run("it should return ErrPriceNotFoundInCache when price not found in cache", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Price{Price: 1000, Address: "address1"})

		repo := NewPriceCacheCMapRepository(cache)

		price, err := repo.Get(ctx, "address2")

		assert.ErrorIs(t, err, ErrPriceNotFoundInCache)
		assert.Equal(t, entity.Price{Address: "address2"}, price)
	})

	t.Run("it should return correct price when price is found in cache", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Price{Price: 1000, Address: "address1"})

		repo := NewPriceCacheCMapRepository(cache)

		price, err := repo.Get(ctx, "address1")

		assert.Nil(t, err)
		assert.Equal(t, entity.Price{Price: 1000, Address: "address1"}, price)
	})
}

func TestPriceCacheCMapRepository_Set(t *testing.T) {
	t.Parallel()

	t.Run("it should set to the map correctly", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()

		repo := NewPriceCacheCMapRepository(cache)
		err := repo.Set(ctx, "address1", entity.Price{Price: 1000, Address: "address1"})

		assert.Nil(t, err)

		price, ok := cache.Get("address1")

		assert.True(t, ok)
		assert.EqualValues(t, entity.Price{Price: 1000, Address: "address1"}, price.(entity.Price))
	})
}

func TestPriceCacheCMapRepository_Remove(t *testing.T) {
	t.Parallel()

	t.Run("it should remove correctly", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Price{Price: 1000, Address: "address1"})

		repo := NewPriceCacheCMapRepository(cache)
		err := repo.Remove(ctx, "address1")

		assert.Nil(t, err)

		_, ok := cache.Get("address1")

		assert.False(t, ok)
	})
}

func TestPriceCacheCMapRepository_Count(t *testing.T) {
	t.Parallel()

	t.Run("it should return number of keys correctly", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Price{Price: 1000, Address: "address1"})

		repo := NewPriceCacheCMapRepository(cache)
		count := repo.Count(ctx)

		assert.Equal(t, 1, count)
	})
}
