package pool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

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

		redisRepositoryConfig := RedisRepositoryConfig{
			Prefix: "",
		}

		repo := NewRedisRepository(db.Client, redisRepositoryConfig)

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

		redisRepositoryConfig := RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := NewRedisRepository(db.Client, redisRepositoryConfig)
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

		redisRepositoryConfig := RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := NewRedisRepository(db.Client, redisRepositoryConfig)

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
			redisServer.HSet(":pools", pool.Address, encodedPool)
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

		redisRepositoryConfig := RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := NewRedisRepository(db.Client, redisRepositoryConfig)
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

		redisRepositoryConfig := RedisRepositoryConfig{
			Prefix: "",
		}

		redisRepository := NewRedisRepository(db.Client, redisRepositoryConfig)
		redisServer.Close()
		pools, err := redisRepository.FindByAddresses(context.Background(), []string{"address1"})
		defer mempool.ReserveMany(pools)

		assert.Error(t, err)
		assert.Nil(t, pools)
	})
}

func TestRedisRepository_GetFaultyPools(t *testing.T) {
	testCases := []struct {
		name        string
		executeFunc func(server *miniredis.Miniredis, repo *redisRepository, t *testing.T)
	}{
		{
			name: "it should return correct faulty pools when there are faulty pools exists in redis",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				// Prepare data
				faultyPools := []string{
					"address1",
					"address2",
					"address3",
				}
				currentTime := time.Now()
				for _, address := range faultyPools {
					redisServer.ZAdd(
						utils.Join(repo.config.Prefix, KeyPools, KeyFaulty),
						float64(currentTime.Add(time.Minute*1).UnixMilli()),
						address)
				}
				redisServer.ZAdd(
					utils.Join(repo.config.Prefix, KeyPools, KeyFaulty),
					float64(currentTime.Add(-time.Minute*1).UnixMilli()),
					"address4")

				pools, err := repo.GetFaultyPools(context.Background(), currentTime.UnixMilli(), 0, -1)

				expectedPools := []string{
					"address1",
					"address2",
					"address3",
				}

				assert.ElementsMatch(t, expectedPools, pools)
				assert.Nil(t, err)
			},
		},
		{
			name: "it should return empty faulty pool list when there are only expired faulty pools exists in redis",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				// Prepare data
				faultyPools := []string{
					"address5",
					"address6",
					"address7",
				}
				currentTime := time.Now()
				for _, address := range faultyPools {
					redisServer.ZAdd(
						utils.Join(repo.config.Prefix, KeyPools, KeyFaulty),
						float64(currentTime.Add(-time.Minute*1).UnixMilli()),
						address)
				}

				pools, err := repo.GetFaultyPools(context.Background(), currentTime.UnixMilli(), 0, -1)

				expectedPools := []string{}

				assert.ElementsMatch(t, expectedPools, pools)
				assert.Nil(t, err)
			},
		},
		{
			name: "it should return empty faulty pool list when the key doesn't exist in Redis",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				currentTime := time.Now()

				pools, err := repo.GetFaultyPools(context.Background(), currentTime.UnixMilli(), 0, -1)

				assert.Empty(t, pools)
				assert.Nil(t, err)
			},
		},
		{
			name: "it should return correct faulty pools when there are faulty pools exists in redis with paging options",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				currentTime := time.Now()
				for i := 0; i < 11; i++ {
					redisServer.ZAdd(
						utils.Join(repo.config.Prefix, KeyPools, KeyFaulty),
						float64(currentTime.Add(time.Minute*1).UnixMilli()),
						fmt.Sprintf("address%d", i))
				}
				redisServer.ZAdd(
					utils.Join(repo.config.Prefix, KeyPools, KeyFaulty),
					float64(currentTime.Add(-time.Minute*1).UnixMilli()),
					"address9")

				pools, err := repo.GetFaultyPools(context.Background(), currentTime.UnixMilli(), 0, 4)
				assert.Nil(t, err)
				assert.Equal(t, len(pools), 4)

				pools, err = repo.GetFaultyPools(context.Background(), currentTime.UnixMilli(), 8, 6)
				assert.Nil(t, err)
				assert.Equal(t, len(pools), 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redisServer, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to setup redis server: %v", err.Error())
			}

			chainId := "ethereum"
			redisConfig := &redis.Config{
				Addresses: []string{redisServer.Addr()},
				Prefix:    chainId,
			}

			db, err := redis.New(redisConfig)
			if err != nil {
				t.Fatalf("failed to setup redis client: %v", err.Error())
			}

			repo := NewRedisRepository(db.Client, RedisRepositoryConfig{
				Prefix: chainId,
			})

			tc.executeFunc(redisServer, repo, t)
			redisServer.Close()
		})
	}

}
