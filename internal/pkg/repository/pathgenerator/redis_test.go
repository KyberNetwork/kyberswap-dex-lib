package pathgenerator

import (
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestBestPathRepository_Operation(t *testing.T) {
	t.Run("it should set and get from redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "custom-prefix",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewRedisRepository(db.Client, RedisRepositoryConfig{Prefix: redisConfig.Prefix})

		// Prepare data
		testData := []*entity.MinimalPath{
			{
				Pools:  []string{"pool1"},
				Tokens: []string{"token1"},
			},
		}

		sourceHash := uint64(0)
		repo.SetBestPaths(sourceHash, "tokenIn", "tokenOut", testData, 10*time.Second)
		actualData := repo.GetBestPaths(sourceHash, "tokenIn", "tokenOut")

		assert.ElementsMatch(t, testData, actualData)

		// Test replace data
		testData[0].Tokens = append(testData[0].Tokens, "token2")
		repo.SetBestPaths(sourceHash, "tokenIn", "tokenOut", testData, 10*time.Second)
		actualData = repo.GetBestPaths(sourceHash, "tokenIn", "tokenOut")
		assert.ElementsMatch(t, testData, actualData)
	})
}
