package route_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestRistrettoRepository_Get(t *testing.T) {
	t.Run("it should return correct routes when keys exist in redis or in memory cache", func(t *testing.T) {
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

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 30 * time.Second},
		})

		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
		}
		routes := []*valueobject.SimpleRoute{
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
		}

		for i, r := range routes {
			encodedRoute, _ := route.EncodeRoute(*r)
			key := genKey(cacheKeys[i], "ethereum")
			redisServer.Set(key, encodedRoute)
			repo.Cache().SetWithTTL(key, r, 1, 10*time.Second)
		}
		invalidKey := valueobject.RouteCacheKeyTTL{
			Key: &valueobject.RouteCacheKey{
				TokenIn:        "ab",
				TokenOut:       "cd",
				OnlySinglePath: false,
				CacheMode:      "normal",
				AmountIn:       "100",
				Dexes:          []string{"uniswap"},
				GasInclude:     false,
				ExcludedPools:  []string{"0xabcd"},
			},
			TTL: time.Second * 10,
		}
		invalidKeyHash := strconv.FormatUint(invalidKey.Key.Hash("ethereum"), 10)
		redisServer.Set(invalidKeyHash, "invalidRoute")
		repo.Cache().SetWithTTL(invalidKeyHash, "invalidRoute", 1, time.Second*10)
		repo.Cache().Wait()

		cacheKeys = append(cacheKeys, invalidKey)
		results, err := repo.Get(context.Background(), cacheKeys)
		resultList := []*valueobject.SimpleRoute{}
		for _, v := range results {
			resultList = append(resultList, v)
		}

		// check if result do not contains invalid key
		_, ok := results[invalidKey]
		assert.False(t, ok)
		assert.Nil(t, err)

		assert.ElementsMatch(t, resultList, routes)
		assert.Nil(t, err)
	})

	t.Run("it should return correct routes when keys exist in redis, but not in memory cache", func(t *testing.T) {
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

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 10,
			Prefix:      "ethereum",

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 50 * time.Second},
		})

		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "c",
					TokenOut:       "d",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"uniswapv3"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xcdefgh"},
				},
				TTL: time.Second * 10,
			},
		}
		routes := []*valueobject.SimpleRoute{
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "c", TokenOutAddress: "d", PoolAddress: "0xcdefgh"}},
				},
			},
		}

		for i, r := range routes {
			encodedRoute, _ := route.EncodeRoute(*r)
			key := genKey(cacheKeys[i], "ethereum")
			redisServer.Set(key, encodedRoute)
		}
		repo.Cache().SetWithTTL(genKey(cacheKeys[0], "ethereum"), routes[0], 1, 10*time.Second)

		nilKey := valueobject.RouteCacheKeyTTL{
			Key: &valueobject.RouteCacheKey{
				TokenIn:        "l",
				TokenOut:       "m",
				OnlySinglePath: true,
				CacheMode:      "normal",
				AmountIn:       "100",
				Dexes:          []string{"pancakev3"},
				GasInclude:     true,
				ExcludedPools:  []string{"0xlmnop"},
			},
			TTL: 10 * time.Second,
		}
		cacheKeys = append(cacheKeys, nilKey)
		_, err = repo.Get(context.Background(), cacheKeys)
		repo.Cache().Wait()

		// Check if all routes are saved correctly into memory after get them from Redis
		resultList := []*valueobject.SimpleRoute{}
		for _, k := range cacheKeys {
			savedRoute, ok := repo.Cache().Get(genKey(k, "ethereum"))
			if k == nilKey {
				assert.False(t, ok)
				assert.Nil(t, savedRoute)
			} else {
				assert.True(t, ok)
				resultList = append(resultList, savedRoute.(*valueobject.SimpleRoute))
			}
		}

		assert.Nil(t, err)

		assert.ElementsMatch(t, resultList, routes)
		assert.Nil(t, err)
	})

	t.Run("it should return empty when keys do not exist in redis", func(t *testing.T) {
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

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 10,
			Prefix:      "ethereum",

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 50 * time.Second},
		})

		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "c",
					TokenOut:       "d",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"uniswapv3"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xcdefgh"},
				},
				TTL: time.Second * 10,
			},
		}

		results, err := repo.Get(context.Background(), cacheKeys)
		repo.Cache().Wait()

		// Check if all routes are saved correctly into memory after get them from Redis
		for _, k := range cacheKeys {
			savedRoute, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.False(t, ok)
			assert.Nil(t, savedRoute)
		}

		assert.Nil(t, err)
		assert.Empty(t, results)
	})
}

func TestRistrettoRepository_Set(t *testing.T) {
	t.Run("it should return no error when keys are set to redis and memory cache successfully", func(t *testing.T) {
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

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,
			Prefix:      "ethereum",

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 30 * time.Second},
		})

		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
		}
		routes := []*valueobject.SimpleRoute{
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
		}
		err = repo.Set(context.Background(), cacheKeys, routes)
		assert.Nil(t, err)
		repo.Cache().Wait()

		// check if cacheKeys are saved in memory correctly
		for i, k := range cacheKeys {
			r, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.True(t, ok)
			assert.Equal(t, r, routes[i])
		}

	})

	t.Run("it should not save any routes to memory cache when set them to redis fail", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,
			Prefix:      "ethereum",

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 30 * time.Second},
		})

		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
		}
		routes := []*valueobject.SimpleRoute{
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
		}
		redisServer.Close()
		err = repo.Set(context.Background(), cacheKeys, routes)
		assert.NotNil(t, err)
		repo.Cache().Wait()

		// check if cacheKeys are not saved in memory correctly
		for _, k := range cacheKeys {
			r, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.False(t, ok)
			assert.Nil(t, r)
		}

	})
}

func TestRistrettoRepository_Del(t *testing.T) {
	t.Run("it should return no error when delete all keys from Redis successfully", func(t *testing.T) {
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

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,
			Prefix:      "ethereum",

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 30 * time.Second},
		})

		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
		}
		routes := []*valueobject.SimpleRoute{
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
		}
		err = repo.Set(context.Background(), cacheKeys, routes)
		assert.Nil(t, err)
		repo.Cache().Wait()

		// check if cacheKeys are saved in memory correctly
		for i, k := range cacheKeys {
			r, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.True(t, ok)
			assert.Equal(t, r, routes[i])
		}

		err = repo.Del(context.Background(), cacheKeys)
		assert.Nil(t, err)

		// check if cacheKeys are deleted from in memory correctly
		for _, k := range cacheKeys {
			r, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.False(t, ok)
			assert.Nil(t, r)
		}
	})

	t.Run("it should not delete any routes to memory cache when delete them to redis fail", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := route.NewRedisRepository(db.Client, route.RedisRepositoryConfig{
			Prefix: "ethereum",
		})

		repo, err := route.NewRistrettoRepository(redisRepo, route.RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,
			Prefix:      "ethereum",

			Route: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 30 * time.Second},
		})
		assert.Nil(t, err)

		cacheKeys := []valueobject.RouteCacheKeyTTL{
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "a",
					TokenOut:       "b",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xabc"},
				},
				TTL: time.Second * 10,
			},
			{
				Key: &valueobject.RouteCacheKey{
					TokenIn:        "x",
					TokenOut:       "y",
					OnlySinglePath: false,
					CacheMode:      "normal",
					AmountIn:       "100",
					Dexes:          []string{"dodo"},
					GasInclude:     false,
					ExcludedPools:  []string{"0xxyz"},
				},
				TTL: time.Second * 10,
			},
		}
		routes := []*valueobject.SimpleRoute{
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
				},
			},
			{
				Distributions: []uint64{100},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
		}
		err = repo.Set(context.Background(), cacheKeys, routes)
		assert.Nil(t, err)
		repo.Cache().Wait()

		// check if cacheKeys are saved in memory correctly
		for i, k := range cacheKeys {
			r, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.True(t, ok)
			assert.Equal(t, r, routes[i])
		}

		redisServer.Close()
		err = repo.Del(context.Background(), cacheKeys)
		assert.NotNil(t, err)

		// check if cacheKeys are not deleted from memory
		for i, k := range cacheKeys {
			r, ok := repo.Cache().Get(genKey(k, "ethereum"))
			assert.True(t, ok)
			assert.Equal(t, r, routes[i])
		}

	})
}
