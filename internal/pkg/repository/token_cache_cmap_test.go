package repository

import (
	"context"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/stretchr/testify/assert"
)

func TestTokenCacheCMapRepository_Keys(t *testing.T) {
	t.Parallel()

	t.Run("it should return correct keys", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("key1", 1)
		cache.Set("key2", 2)

		repo := NewTokenCacheCMapRepository(cache)

		assert.ElementsMatch(t, repo.Keys(ctx), []string{"key1", "key2"})
	})
}

func TestTokenCacheCMapRepository_Get(t *testing.T) {
	t.Parallel()

	t.Run("it should return ErrTokenNotFoundInCache when token not found in cache", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Token{Address: "address1", Symbol: "symbol1"})

		repo := NewTokenCacheCMapRepository(cache)

		token, err := repo.Get(ctx, "address2")

		assert.ErrorIs(t, err, ErrTokenNotFoundInCache)
		assert.Equal(t, entity.Token{Address: "address2"}, token)
	})

	t.Run("it should return correct token when token is found in cache", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Token{Address: "address1", Symbol: "symbol1"})

		repo := NewTokenCacheCMapRepository(cache)

		token, err := repo.Get(ctx, "address1")

		assert.Nil(t, err)
		assert.Equal(t, entity.Token{Address: "address1", Symbol: "symbol1"}, token)
	})
}

func TestTokenCacheCMapRepository_Set(t *testing.T) {
	t.Parallel()

	t.Run("it should set to the map correctly", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()

		repo := NewTokenCacheCMapRepository(cache)
		err := repo.Set(ctx, "address1", entity.Token{Address: "address1", Symbol: "symbol1"})

		assert.Nil(t, err)

		token, ok := cache.Get("address1")

		assert.True(t, ok)
		assert.EqualValues(t, entity.Token{Address: "address1", Symbol: "symbol1"}, token.(entity.Token))
	})
}

func TestTokenCacheCMapRepository_Remove(t *testing.T) {
	t.Parallel()

	t.Run("it should remove correctly", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Token{Address: "address1", Symbol: "symbol1"})

		repo := NewTokenCacheCMapRepository(cache)
		err := repo.Remove(ctx, "address1")

		assert.Nil(t, err)

		_, ok := cache.Get("address1")

		assert.False(t, ok)
	})
}

func TestTokenCacheCMapRepository_Count(t *testing.T) {
	t.Parallel()

	t.Run("it should return number of keys correctly", func(t *testing.T) {
		ctx := context.Background()
		cache := cmap.New()
		cache.Set("address1", entity.Token{Address: "address1", Symbol: "symbol1"})

		repo := NewTokenCacheCMapRepository(cache)
		count := repo.Count(ctx)

		assert.Equal(t, 1, count)
	})
}
