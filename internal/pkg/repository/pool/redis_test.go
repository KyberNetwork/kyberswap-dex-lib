package pool

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/mempool"
	"github.com/KyberNetwork/router-service/pkg/redis"
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

		repo, _ := NewRedisRepository(db.Client, nil, Config{
			Redis: RedisRepositoryConfig{
				Prefix: "ethereum",
			},
			Ristretto: RistrettoConfig{
				NumCounters: 500,
				MaxCost:     1,
				BufferItems: 100,
			},
		})

		redisPools := []entity.Pool{
			{
				Address: "address1",
			},
			{
				Address: "address2",
			},
		}

		for _, pool := range redisPools {
			encodedPool, _ := encodePool(pool)
			redisServer.HSet("ethereum:pools", pool.Address, encodedPool)
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

		redisRepository, _ := NewRedisRepository(db.Client, nil, Config{
			Redis: RedisRepositoryConfig{
				Prefix: "ethereum",
			},
			Ristretto: RistrettoConfig{
				NumCounters: 500,
				MaxCost:     1,
				BufferItems: 100,
			},
		})
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

		redisRepository, _ := NewRedisRepository(db.Client, nil, Config{
			Redis: RedisRepositoryConfig{
				Prefix: "ethereum",
			},
			Ristretto: RistrettoConfig{
				NumCounters: 500,
				MaxCost:     1,
				BufferItems: 100,
			},
		})

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
			encodedPool, _ := encodePool(pool)
			redisServer.HSet("ethereum:pools", pool.Address, encodedPool)
		}

		pools, err := redisRepository.FindByAddresses(context.Background(), []string{"address1", "address2", "address4"})
		defer mempool.ReserveMany(pools)

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

		redisRepository, _ := NewRedisRepository(db.Client, nil, Config{
			Redis: RedisRepositoryConfig{
				Prefix: "ethereum",
			},
			Ristretto: RistrettoConfig{
				NumCounters: 500,
				MaxCost:     1,
				BufferItems: 100,
			},
		})
		pools, err := redisRepository.FindByAddresses(context.Background(), nil)
		defer mempool.ReserveMany(pools)

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

		redisRepository, _ := NewRedisRepository(db.Client, nil, Config{
			Redis: RedisRepositoryConfig{
				Prefix: "ethereum",
			},
			Ristretto: RistrettoConfig{
				NumCounters: 500,
				MaxCost:     1,
				BufferItems: 100,
			},
		})
		redisServer.Close()
		pools, err := redisRepository.FindByAddresses(context.Background(), []string{"address1"})
		defer mempool.ReserveMany(pools)

		assert.Error(t, err)
		assert.Nil(t, pools)
	})
}

func TestRistrettoRepository_Get(t *testing.T) {
	t.Run("it should return correct faulty pools when keys exist in local memory cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "ethereum",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		poolClient := mocks.NewMockIPoolClient(ctrl)
		poolClient.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		repo, err := NewRedisRepository(db.Client, poolClient, Config{
			Redis: RedisRepositoryConfig{
				Prefix:            "ethereum",
				MaxFaultyPoolSize: 5,
			},
			Ristretto: RistrettoConfig{
				NumCounters: 5000,
				MaxCost:     500,
				BufferItems: 64,

				FaultyPools: struct {
					Cost int64         `mapstructure:"cost"`
					TTL  time.Duration `mapstructure:"ttl"`
				}{Cost: 1, TTL: 10 * time.Minute},
			},
		})
		assert.Nil(t, err)

		// save faulty pools to local cache
		faultyPools := []string{"0xabc", "0xxyz", "0xcdfgh"}
		ok := repo.cache.SetWithTTL(utils.Join("ethereum", faultyPoolKey), faultyPools, 1, 1*time.Minute)
		repo.cache.Wait()
		assert.True(t, ok)

		res, err := repo.GetFaultyPools(context.Background())

		assert.ElementsMatch(t, res, faultyPools)
		assert.Nil(t, err)

	})

	t.Run("it should return correct faulty pools when keys doesn't exist in local memory cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "ethereum",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		poolClient := mocks.NewMockIPoolClient(ctrl)
		faultyPools := []string{"0xabc", "0xxyz", "0xcdfgh"}
		poolClient.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(faultyPools, nil)
		repo, err := NewRedisRepository(db.Client, poolClient, Config{
			Redis: RedisRepositoryConfig{
				Prefix:            "ethereum",
				MaxFaultyPoolSize: 5,
			},
			Ristretto: RistrettoConfig{
				NumCounters: 5000,
				MaxCost:     500,
				BufferItems: 64,

				FaultyPools: struct {
					Cost int64         `mapstructure:"cost"`
					TTL  time.Duration `mapstructure:"ttl"`
				}{Cost: 1, TTL: 10 * time.Minute},
			},
		})
		assert.Nil(t, err)
		res, err := repo.GetFaultyPools(context.Background())
		repo.cache.Wait()

		// assert in mem cache contains correct faulty pools
		cachedData, ok := repo.cache.Get("ethereum:faultyPools")
		assert.True(t, ok)
		assert.ElementsMatch(t, res, faultyPools)
		assert.ElementsMatch(t, cachedData, faultyPools)
		assert.Nil(t, err)

	})

	t.Run("it should return correct faulty pools when keys doesn't exist in local memory cache with paging logic", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "ethereum",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		poolClient := mocks.NewMockIPoolClient(ctrl)
		faultyPools := []string{"0xabc", "0xxyz", "0xcdfgh"}
		poolClient.EXPECT().GetFaultyPools(gomock.Any(), gomock.Eq(int64(0)), gomock.Eq(int64(3))).Times(1).Return(faultyPools, nil)
		poolClient.EXPECT().GetFaultyPools(gomock.Any(), gomock.Eq(int64(3)), gomock.Eq(int64(3))).Times(1).Return([]string{"0xlmn", "0xdef"}, nil)
		repo, err := NewRedisRepository(db.Client, poolClient, Config{
			Redis: RedisRepositoryConfig{
				Prefix:            "ethereum",
				MaxFaultyPoolSize: 3,
			},
			Ristretto: RistrettoConfig{
				NumCounters: 5000,
				MaxCost:     500,
				BufferItems: 64,

				FaultyPools: struct {
					Cost int64         `mapstructure:"cost"`
					TTL  time.Duration `mapstructure:"ttl"`
				}{Cost: 1, TTL: 10 * time.Minute},
			},
		})
		assert.Nil(t, err)
		res, err := repo.GetFaultyPools(context.Background())
		repo.cache.Wait()

		totalFaultyPools := []string{"0xabc", "0xxyz", "0xcdfgh", "0xlmn", "0xdef"}

		// assert in mem cache contains correct faulty pools
		cachedData, ok := repo.cache.Get("ethereum:faultyPools")
		assert.True(t, ok)
		assert.ElementsMatch(t, res, totalFaultyPools)
		assert.ElementsMatch(t, cachedData, totalFaultyPools)
		assert.Nil(t, err)

	})
}
