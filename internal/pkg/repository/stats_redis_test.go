package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestStatsRedisRepository_Get(t *testing.T) {
	t.Run("it should return correct stats in redis", func(t *testing.T) {
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

		repo := NewStatsRedisRepository(db)

		// Prepare data
		redisStats := entity.Stats{
			TotalPools:  11,
			TotalTokens: 111,
			Pools: map[string]entity.PoolStatsItem{
				"0x11111": {
					Size: 11,
					Tvl:  11111,
				},
				"0x22222": {
					Size: 22,
					Tvl:  22222,
				},
			},
		}

		redisServer.HSet(fmt.Sprintf(":%s", StatsKey), TotalPoolKey, strconv.Itoa(redisStats.TotalPools))
		redisServer.HSet(fmt.Sprintf(":%s", StatsKey), TotalTokenKey, strconv.Itoa(redisStats.TotalTokens))

		encodedPools, _ := json.Marshal(redisStats.Pools)

		redisServer.HSet(fmt.Sprintf(":%s", StatsKey), PoolsKey, string(encodedPools))

		stats, err := repo.Get(context.Background())

		assert.Equal(t, redisStats, stats)
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

		repo := NewStatsRedisRepository(db)

		redisServer.Close()

		stats, err := repo.Get(context.Background())

		assert.Equal(t, entity.Stats{}, stats)
		assert.Error(t, err)
	})
}

func TestStatsRedisRepository_Persist(t *testing.T) {
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

		repo := NewStatsRedisRepository(db)

		stats := entity.Stats{
			TotalPools:  11,
			TotalTokens: 111,
			Pools: map[string]entity.PoolStatsItem{
				"0x11111": {
					Size: 11,
					Tvl:  11111,
				},
				"0x22222": {
					Size: 22,
					Tvl:  22222,
				},
			},
		}

		err = repo.Persist(context.Background(), stats)

		totalPoolsRedis := redisServer.HGet(fmt.Sprintf(":%s", StatsKey), TotalPoolKey)
		totalTokensRedis := redisServer.HGet(fmt.Sprintf(":%s", StatsKey), TotalTokenKey)
		poolsRedis := redisServer.HGet(fmt.Sprintf(":%s", StatsKey), PoolsKey)

		var pools map[string]entity.PoolStatsItem

		_ = json.Unmarshal([]byte(poolsRedis), &pools)

		assert.Nil(t, err)
		assert.Equal(t, totalPoolsRedis, strconv.Itoa(stats.TotalPools))
		assert.Equal(t, totalTokensRedis, strconv.Itoa(stats.TotalTokens))
		assert.Equal(t, pools, stats.Pools)
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

		repo := NewStatsRedisRepository(db)

		redisServer.Close()

		err = repo.Persist(context.Background(), entity.Stats{})

		assert.Error(t, err)
	})
}
