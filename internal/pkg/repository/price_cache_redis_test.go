package repository

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"

	"github.com/stretchr/testify/assert"
)

func TestPriceCacheRedisRepository_Keys(t *testing.T) {
	t.Parallel()

	t.Run("it should return correct keys", func(t *testing.T) {
		ctx := context.Background()
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		cache, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		redisServer.HSet(":prices", "key1", "1")
		redisServer.HSet(":prices", "key2", "2")

		repo := NewPriceCacheRedisRepository(cache)

		assert.ElementsMatch(t, repo.Keys(ctx), []string{"key1", "key2"})
	})
}

func TestPriceCacheRedisRepository_Get(t *testing.T) {
	t.Parallel()

	t.Run("it should return ErrPriceNotFoundInCache when price not found in cache", func(t *testing.T) {
		ctx := context.Background()

		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}
		cache, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		redisServer.HSet(":prices", "address1", entity.Price{Price: 1000, Address: "address1"}.Encode())

		repo := NewPriceCacheRedisRepository(cache)

		price, err := repo.Get(ctx, "address2")

		assert.ErrorIs(t, err, ErrPriceNotFoundInCache)
		assert.Equal(t, entity.Price{Address: "address2"}, price)
	})

	t.Run("it should return correct price when price is found in cache", func(t *testing.T) {
		ctx := context.Background()

		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}
		cache, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		redisServer.HSet(":prices", "address1", entity.Price{Price: 1000, Address: "address1"}.Encode())

		repo := NewPriceCacheRedisRepository(cache)

		price, err := repo.Get(ctx, "address1")

		assert.Nil(t, err)
		assert.Equal(t, entity.Price{Price: 1000, Address: "address1"}, price)
	})
}

func TestPriceCacheRedisRepository_Set(t *testing.T) {
	t.Parallel()

	t.Run("it should set to the map correctly", func(t *testing.T) {
		ctx := context.Background()

		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()
		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		cache, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceCacheRedisRepository(cache)
		err = repo.Set(ctx, "address1", entity.Price{Price: 1000, Address: "address1"})

		assert.Nil(t, err)

		price := redisServer.HGet(":prices", "address1")

		assert.EqualValues(t, entity.Price{Price: 1000, Address: "address1"}.Encode(), price)
	})
}

func TestPriceCacheRedisRepository_Remove(t *testing.T) {
	t.Parallel()

	t.Run("it should remove correctly", func(t *testing.T) {
		ctx := context.Background()

		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}
		cache, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		redisServer.HSet(":prices", "address1", entity.Price{Price: 1000, Address: "address1"}.Encode())

		repo := NewPriceCacheRedisRepository(cache)

		err = repo.Remove(ctx, "address1")

		assert.Nil(t, err)

		_, err = repo.Get(ctx, "address1")

		assert.ErrorIs(t, err, ErrPriceNotFoundInCache)
	})
}

func TestPriceCacheRedisRepository_Count(t *testing.T) {
	t.Parallel()

	t.Run("it should return number of keys correctly", func(t *testing.T) {
		ctx := context.Background()

		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		cache, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		redisServer.HSet(":prices", "address1", entity.Price{Price: 1000, Address: "address1"}.Encode())
		redisServer.HSet(":prices", "address2", entity.Price{Price: 1000, Address: "address2"}.Encode())

		repo := NewPriceCacheRedisRepository(cache)
		count := repo.Count(ctx)

		assert.Equal(t, 2, count)
	})
}
