package pool_test

import (
	"context"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestRedisRepository_FindAllAddresses(t *testing.T) {
	t.Run("it should return all pools in redis", func(t *testing.T) {
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

		redisRepositoryConfig := pool.RedisRepositoryConfig{
			Prefix: "",
		}

		repo := pool.NewRedisRepository(db.Client, redisRepositoryConfig)

		redisPools := []entity.Pool{
			{
				Address: "address1",
			},
			{
				Address: "address2",
			},
		}

		for _, pool := range redisPools {
			encodedPool, _ := pool.Encode()
			redisServer.HSet(":pools", pool.Address, encodedPool)
		}

		addresses, err := repo.FindAllAddresses(context.Background())

		assert.ElementsMatch(t, []string{"address1", "address2"}, addresses)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
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

		redisRepositoryConfig := pool.RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := pool.NewRedisRepository(db.Client, redisRepositoryConfig)
		redisServer.Close()

		addresses, err := redisRepository.FindAllAddresses(context.Background())

		assert.Nil(t, addresses)
		assert.Error(t, err)
	})
}

func TestRedisRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return correct pools when addresses are exists in redis", func(t *testing.T) {
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

		redisRepositoryConfig := pool.RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := pool.NewRedisRepository(db.Client, redisRepositoryConfig)

		// Prepare data
		redisPools := []entity.Pool{
			{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			{
				Address:      "address2",
				ReserveUsd:   1000,
				AmplifiedTvl: 1000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra2",
				StaticExtra: "staticExtra2",
				TotalSupply: "totalSupply2",
			},
			{
				Address:      "address3",
				ReserveUsd:   10000,
				AmplifiedTvl: 10000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra3",
				StaticExtra: "staticExtra3",
				TotalSupply: "totalSupply3",
			},
		}
		for _, pool := range redisPools {
			encodedPool, _ := pool.Encode()
			redisServer.HSet(":pools", pool.Address, encodedPool)
		}

		pools, err := redisRepository.FindByAddresses(context.Background(), []string{"address1", "address2", "address4"})
		expectedPools := []*entity.Pool{
			{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			{
				Address:      "address2",
				ReserveUsd:   1000,
				AmplifiedTvl: 1000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra2",
				StaticExtra: "staticExtra2",
				TotalSupply: "totalSupply2",
			},
		}

		assert.ElementsMatch(t, expectedPools, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
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
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		redisRepositoryConfig := pool.RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := pool.NewRedisRepository(db.Client, redisRepositoryConfig)
		pools, err := redisRepository.FindByAddresses(context.Background(), nil)

		assert.Nil(t, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
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

		redisRepositoryConfig := pool.RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := pool.NewRedisRepository(db.Client, redisRepositoryConfig)
		redisServer.Close()
		pools, err := redisRepository.FindByAddresses(context.Background(), []string{"address1"})

		assert.Error(t, err)
		assert.Nil(t, pools)
	})
}
