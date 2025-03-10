package route_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func genKey(key valueobject.RouteCacheKeyTTL, prefix string) string {
	return utils.Join(prefix, strconv.FormatUint(key.Key.Hash(prefix), 10))
}

func TestRedisCacheRepository_Set(t *testing.T) {
	t.Run("it should have set data in redis", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})
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
		routes := []*valueobject.SimpleRouteWithExtraData{
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
					},
				},
			},
			{
				AMMRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
					},
				},
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
					},
				},
			},
		}
		cachedRoutes, err := cache.Set(
			context.Background(),
			cacheKeys,
			routes,
		)
		for i, r := range cachedRoutes {
			assert.Equal(t, r, routes[i])
		}

		assert.Nil(t, err)
		assert.Equal(t, len(cachedRoutes), 2)

		dbData := []*valueobject.SimpleRouteWithExtraData{}
		for _, key := range cacheKeys {
			dbResult, _ := redisServer.Get(genKey(key, "ethereum"))
			route, _ := route.DecodeRoute(dbResult)
			dbData = append(dbData, route)
		}
		assert.ElementsMatch(t, cachedRoutes, dbData)

	})

	t.Run("it should return err when redis server down", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})
		redisServer.Close()

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
		}
		routes := []*valueobject.SimpleRouteWithExtraData{
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
					},
				},
			},
		}
		_, err = cache.Set(
			context.Background(),
			cacheKeys,
			routes,
		)

		assert.Error(t, err)
	})
}

func TestRedisCacheRepository_Get(t *testing.T) {
	t.Run("it should get data from redis successfully", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

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
		routes := []*valueobject.SimpleRouteWithExtraData{
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
					},
				},
			},
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
					},
				},
			},
		}

		for i, k := range cacheKeys {
			encoded, _ := route.EncodeRoute(*routes[i])
			redisServer.Set(genKey(k, "ethereum"), encoded)
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})

		result, err := cache.Get(context.Background(), cacheKeys)

		assert.Nil(t, err)
		// verify result
		resultList := []*valueobject.SimpleRouteWithExtraData{}
		for _, v := range result {
			resultList = append(resultList, v)
		}
		assert.ElementsMatch(t, resultList, routes)
	})

	t.Run("it should get data from redis successfully, combine get and set", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

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
		routes := []*valueobject.SimpleRouteWithExtraData{
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
					},
				},
			},
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
					},
				},
			},
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})
		savedRoutes, err := cache.Set(context.Background(), cacheKeys, routes)
		assert.Nil(t, err)

		// add extra keys without data in redis
		nilKey := valueobject.RouteCacheKeyTTL{
			Key: &valueobject.RouteCacheKey{
				TokenIn:        "c",
				TokenOut:       "d",
				OnlySinglePath: false,
				CacheMode:      "normal",
				AmountIn:       "100",
				Dexes:          []string{"dodo"},
				GasInclude:     false,
				ExcludedPools:  []string{"0xcdf"},
			},
			TTL: 10 * time.Second,
		}
		cacheKeys = append(cacheKeys, nilKey)
		result, err := cache.Get(context.Background(), cacheKeys)

		assert.Nil(t, err)
		// verify result
		resultList := []*valueobject.SimpleRouteWithExtraData{}
		for _, v := range result {
			resultList = append(resultList, v)
		}
		assert.Nil(t, result[nilKey])
		assert.ElementsMatch(t, savedRoutes, resultList)
	})

	t.Run("it should return nil when redis does not have data", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})

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
		result, err := cache.Get(context.Background(), cacheKeys)

		assert.Nil(t, err)
		assert.Empty(t, result)

	})

	t.Run("it should return err when redis server down", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})
		redisServer.Close()

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
		_, err = cache.Get(context.Background(), cacheKeys)

		assert.Error(t, err)
	})
}

func TestRedisCacheRepository_Del(t *testing.T) {
	t.Run("it should get delete data from redis successfully", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis server: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

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
		routes := []*valueobject.SimpleRouteWithExtraData{
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
					},
				},
			},
			{
				BestRoute: &valueobject.SimpleRoute{
					Distributions: []uint64{100},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
					},
				},
			},
		}

		for i, k := range cacheKeys {
			encoded, _ := route.EncodeRoute(*routes[i])
			redisServer.Set(genKey(k, "ethereum"), encoded)
		}

		cache := route.NewRedisRepository(
			db.Client,
			route.RedisRepositoryConfig{
				Prefix: "ethereum",
			})

		err = cache.Del(context.Background(), cacheKeys)

		assert.Nil(t, err)
	})
}
