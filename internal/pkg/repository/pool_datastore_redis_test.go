package repository

import (
	"context"
	"strconv"
	"testing"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestPoolDatastoreRedisRepository_FindAll(t *testing.T) {
	t.Run("it should return all pools in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

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
		}

		for _, pool := range redisPools {
			encodedPool, _ := pool.Encode()
			redisServer.HSet(":pools", pool.Address, encodedPool)
		}

		pools, err := repo.FindAll(context.Background())

		assert.ElementsMatch(t, redisPools, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down ", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

		redisServer.Close()

		pools, err := repo.FindAll(context.Background())

		assert.Nil(t, pools)
		assert.Error(t, err)
	})
}

func TestPoolDatastoreRedisRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := NewPoolDataStoreRedisRepository(nil)

		pools, err := repo.FindByAddresses(context.Background(), nil)

		assert.Nil(t, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return correct pools when addresses are exists in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

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

		pools, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address4"})

		expectedPools := []entity.Pool{
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

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

		redisServer.Close()

		pools, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, pools)
		assert.Error(t, err)
	})
}

func TestPoolDatastoreRedisRepository_Persist(t *testing.T) {
	t.Run("it should persist data correctly", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

		thePool := entity.Pool{
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
		}

		err = repo.Persist(context.Background(), thePool)

		redisPool := redisServer.HGet(":pools", "address1")

		expectedPoolEncoded, _ := thePool.Encode()

		assert.Nil(t, err)
		assert.Equal(t, expectedPoolEncoded, redisPool)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.Persist(context.Background(), entity.Pool{})

		assert.Error(t, err)
	})
}

func TestPoolDatastoreRedisRepository_Delete(t *testing.T) {
	t.Run("it should delete data correctly", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

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

		repo := NewPoolDataStoreRedisRepository(db)

		thePool := entity.Pool{
			Address: "address1",
		}

		err = repo.Delete(context.Background(), thePool)

		assert.Nil(t, err)

		encodedPool := redisServer.HGet(":pools", "address1")

		assert.Empty(t, encodedPool)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPoolDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.Delete(context.Background(), entity.Pool{Address: "address1"})

		assert.Error(t, err)
	})
}
