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

		repo := NewRedisRepository(db.Client, nil, redisRepositoryConfig)

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

		redisRepository := NewRedisRepository(db.Client, nil, redisRepositoryConfig)
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

		redisRepository := NewRedisRepository(db.Client, nil, redisRepositoryConfig)

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

		redisRepository := NewRedisRepository(db.Client, nil, redisRepositoryConfig)
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

		redisRepository := NewRedisRepository(db.Client, nil, redisRepositoryConfig)
		redisServer.Close()
		pools, err := redisRepository.FindByAddresses(context.Background(), []string{"address1"})
		defer mempool.ReserveMany(pools)

		assert.Error(t, err)
		assert.Nil(t, pools)
	})
}

func TestRedisRepository_HIncreaseByMultiple(t *testing.T) {
	testCases := []struct {
		name        string
		executeFunc func(server *miniredis.Miniredis, repo *redisRepository, t *testing.T)
	}{
		{
			name: "it should increase correct count when there are available hash keys in Redis",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				// Prepare data
				pools := map[string]int64{
					"address1:18:11:45": int64(1),
					"address2:18:11:45": int64(5),
					"address3:18:11:45": int64(6),
				}
				for key, count := range pools {
					redisServer.HSet(
						utils.Join(repo.config.Prefix, key),
						KeyTotalCount,
						fmt.Sprintf("%d", count))
				}

				counters := map[string]int64{
					"address1:18:11:45": int64(1),
					"address2:18:11:45": int64(2),
				}
				expireTime := time.Minute * 1
				pools, errors := repo.IncreasePoolsTotalCount(context.Background(), counters, expireTime)
				expectedResult := map[string]int64{
					"address1:18:11:45": int64(2),
					"address2:18:11:45": int64(7),
				}
				// asert returned result
				assert.Equal(t, expectedResult, pools)
				assert.Equal(t, errors, []error{})

				// assert data on Redis
				expectedData := map[string]string{
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address1:18:11:45"): "2",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address2:18:11:45"): "7",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address3:18:11:45"): "6",
				}
				for key, count := range expectedData {
					r := redisServer.HGet(key, KeyTotalCount)
					assert.Equal(t, count, r)
				}
			},
		},
		{
			name: "it should create new hash key when the key is not found in Redis",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				// Prepare data
				pools := map[string]int64{
					"address1:18:11:45": int64(1),
					"address2:18:11:45": int64(5),
					"address3:18:11:45": int64(6),
				}
				for key, count := range pools {
					redisServer.HSet(
						utils.Join(repo.config.Prefix, key),
						KeyTotalCount,
						fmt.Sprintf("%d", count))
				}

				counters := map[string]int64{
					"address2:18:11:45": int64(2),
					"address4:18:11:45": int64(2),
				}
				expireTime := time.Minute * 1
				pools, errors := repo.IncreasePoolsTotalCount(context.Background(), counters, expireTime)
				expectedResult := map[string]int64{
					"address2:18:11:45": int64(7),
					"address4:18:11:45": int64(2),
				}
				// asert returned result
				assert.Equal(t, expectedResult, pools)
				assert.Equal(t, errors, []error{})

				// assert data on Redis
				expectedData := map[string]string{
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address1:18:11:45"): "1",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address2:18:11:45"): "7",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address3:18:11:45"): "6",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address4:18:11:45"): "2",
				}
				for key, count := range expectedData {
					r := redisServer.HGet(key, KeyTotalCount)
					assert.Equal(t, count, r)
				}
			},
		},
		{
			name: "it should create new fields when the hash key doesn't contain that fields",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				// Prepare data
				pools := map[string]int64{
					"address1:18:11:45": int64(1),
					"address2:18:11:45": int64(5),
					"address3:18:11:45": int64(6),
				}
				for key, count := range pools {
					redisServer.HSet(
						utils.Join(repo.config.Prefix, key),
						"failedCount",
						fmt.Sprintf("%d", count))
				}

				counters := map[string]int64{
					"address2:18:11:45": int64(2),
					"address4:18:11:45": int64(2),
				}
				expireTime := time.Minute * 1
				pools, errors := repo.IncreasePoolsTotalCount(context.Background(), counters, expireTime)
				expectedResult := map[string]int64{
					"address2:18:11:45": int64(2),
					"address4:18:11:45": int64(2),
				}
				// asert returned result
				assert.Equal(t, expectedResult, pools)
				assert.Equal(t, errors, []error{})

				// assert data on Redis
				expectedData := map[string]string{
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address1:18:11:45"): "",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address2:18:11:45"): "2",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address3:18:11:45"): "",
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address4:18:11:45"): "2",
				}
				for key, count := range expectedData {
					r := redisServer.HGet(key, KeyTotalCount)
					assert.Equal(t, count, r)
				}
			},
		},
		{
			name: "it should return correct TTL when check TTL for hash key",
			executeFunc: func(redisServer *miniredis.Miniredis, repo *redisRepository, t *testing.T) {
				// Prepare data
				pools := map[string]int64{
					"address1:18:11:45": int64(1),
					"address3:18:11:45": int64(6),
				}
				for key, count := range pools {
					redisServer.HSet(
						utils.Join(repo.config.Prefix, key),
						KeyTotalCount,
						fmt.Sprintf("%d", count))
				}

				counters := map[string]int64{
					"address1:18:11:45": int64(2),
				}
				expireTime := time.Minute * 3
				pools, errors := repo.IncreasePoolsTotalCount(context.Background(), counters, expireTime)
				expectedResult := map[string]int64{
					"address1:18:11:45": int64(3),
				}
				// asert returned result
				assert.Equal(t, expectedResult, pools)
				assert.Equal(t, errors, []error{})

				// assert data on Redis
				expectedData := map[string]string{
					fmt.Sprintf("%s:%s", repo.config.Prefix, "address1:18:11:45"): "3",
				}
				for key := range expectedData {
					duration := redisServer.TTL(key)
					assert.True(t, duration.Minutes() > 1)
				}
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

			repo := NewRedisRepository(db.Client, nil, RedisRepositoryConfig{
				Prefix: chainId,
			})

			tc.executeFunc(redisServer, repo, t)
			redisServer.Close()
		})
	}

}
