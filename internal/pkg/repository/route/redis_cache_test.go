package route_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

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

		cache := route.NewRedisCacheRepository(
			db.Client,
			route.RedisCacheRepositoryConfig{
				Prefix:         "",
				LocalCacheSize: 2,
				LocalCacheTTL:  time.Second,
			})
		err = cache.Set(
			context.Background(),
			&valueobject.RouteCacheKey{
				TokenIn:                "addressIn",
				TokenOut:               "addressOut",
				SaveGas:                false,
				CacheMode:              "normal",
				AmountIn:               "100",
				Dexes:                  []string{"dodo"},
				GasInclude:             false,
				IsPathGeneratorEnabled: false,
			},
			&valueobject.SimpleRoute{
				Distributions: []uint64{1},
				Paths:         nil,
			},
			time.Second,
		)

		assert.Nil(t, err)

		dbResult, err := redisServer.Get("1839672685618436818")
		if err != nil {
			t.Fatalf("failed to get redis data: %v", err.Error())
		}
		assert.NotNil(t, dbResult)

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

		cache := route.NewRedisCacheRepository(
			db.Client,
			route.RedisCacheRepositoryConfig{
				Prefix:         "",
				LocalCacheSize: 2,
				LocalCacheTTL:  time.Second,
			})
		redisServer.Close()

		err = cache.Set(
			context.Background(),
			&valueobject.RouteCacheKey{
				TokenIn:    "addressIn",
				TokenOut:   "addressOut",
				SaveGas:    false,
				CacheMode:  "normal",
				AmountIn:   "100",
				Dexes:      []string{"dodo"},
				GasInclude: false,
			},
			&valueobject.SimpleRoute{
				Distributions: []uint64{1},
				Paths:         nil,
			},
			time.Second,
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

		cache := route.NewRedisCacheRepository(
			db.Client,
			route.RedisCacheRepositoryConfig{
				Prefix:         "",
				LocalCacheSize: 2,
				LocalCacheTTL:  time.Second,
			})

		err = cache.Set(
			context.Background(),
			&valueobject.RouteCacheKey{
				TokenIn:    "addressIn",
				TokenOut:   "addressOut",
				SaveGas:    false,
				CacheMode:  "normal",
				AmountIn:   "100",
				Dexes:      []string{"dodo"},
				GasInclude: false,
			},
			&valueobject.SimpleRoute{
				Distributions: []uint64{1},
				Paths:         nil,
			},
			time.Second,
		)

		assert.Nil(t, err)

		result, err := cache.Get(
			context.Background(),
			&valueobject.RouteCacheKey{
				TokenIn:    "addressIn",
				TokenOut:   "addressOut",
				SaveGas:    false,
				CacheMode:  "normal",
				AmountIn:   "100",
				Dexes:      []string{"dodo"},
				GasInclude: false,
			},
		)

		assert.Nil(t, err)
		assert.NotNil(t, result)
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

		cache := route.NewRedisCacheRepository(
			db.Client,
			route.RedisCacheRepositoryConfig{
				Prefix:         "",
				LocalCacheSize: 2,
				LocalCacheTTL:  time.Second,
			})

		_, err = cache.Get(
			context.Background(),
			&valueobject.RouteCacheKey{
				TokenIn:    "addressIn",
				TokenOut:   "addressOut",
				SaveGas:    false,
				CacheMode:  "normal",
				AmountIn:   "100",
				Dexes:      []string{"dodo"},
				GasInclude: false,
			},
		)

		assert.Error(t, err)
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

		cache := route.NewRedisCacheRepository(
			db.Client,
			route.RedisCacheRepositoryConfig{
				Prefix:         "",
				LocalCacheSize: 2,
				LocalCacheTTL:  time.Second,
			})
		redisServer.Close()

		_, err = cache.Get(
			context.Background(),
			&valueobject.RouteCacheKey{
				TokenIn:    "addressIn",
				TokenOut:   "addressOut",
				SaveGas:    false,
				CacheMode:  "normal",
				AmountIn:   "100",
				Dexes:      []string{"dodo"},
				GasInclude: false,
			},
		)

		assert.Error(t, err)
	})
}
