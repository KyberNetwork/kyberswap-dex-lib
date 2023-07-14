package token

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

func TestTokenCacheRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return empty when addresses is empty", func(t *testing.T) {
		repo := NewGoCacheRepository(nil, GoCacheRepositoryConfig{
			Expiration:      10 * time.Second,
			CleanupInterval: 20 * time.Second,
		})

		tokens, err := repo.FindByAddresses(context.Background(), nil)

		assert.Empty(t, tokens)
		assert.Nil(t, err)
	})

	t.Run("it should return correct tokens when addresses are exists in redis or in memory cache", func(t *testing.T) {
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
		tokenDatastoreRepo := NewRedisRepository(db.Client, RedisRepositoryConfig{
			Prefix: "",
		})

		repo := NewGoCacheRepository(tokenDatastoreRepo, GoCacheRepositoryConfig{
			Expiration:      10 * time.Second,
			CleanupInterval: 20 * time.Second,
		})

		// Prepare data test case 1 all token from cache
		redisTokens := []entity.Token{
			{
				Address:     "address1",
				Symbol:      "symbol1",
				Name:        "name1",
				Decimals:    18,
				CgkID:       "cgkId1",
				Type:        "erc20",
				PoolAddress: "poolAddress1",
			},
			{
				Address:     "address2",
				Symbol:      "symbol2",
				Name:        "name2",
				Decimals:    18,
				CgkID:       "cgkId2",
				Type:        "erc20",
				PoolAddress: "poolAddress2",
			},
			{
				Address:     "address3",
				Symbol:      "symbol3",
				Name:        "name3",
				Decimals:    18,
				CgkID:       "cgkId3",
				Type:        "erc20",
				PoolAddress: "poolAddress3",
			},
		}

		for _, token := range redisTokens {
			encodedToken, _ := encodeToken(token)
			redisServer.HSet(":tokens", token.Address, encodedToken)
		}

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address3"})

		expectedTokens := []*entity.Token{
			{
				Address:     "address1",
				Symbol:      "symbol1",
				Name:        "name1",
				Decimals:    18,
				CgkID:       "cgkId1",
				Type:        "erc20",
				PoolAddress: "poolAddress1",
			},
			{
				Address:     "address2",
				Symbol:      "symbol2",
				Name:        "name2",
				Decimals:    18,
				CgkID:       "cgkId2",
				Type:        "erc20",
				PoolAddress: "poolAddress2",
			},
			{
				Address:     "address3",
				Symbol:      "symbol3",
				Name:        "name3",
				Decimals:    18,
				CgkID:       "cgkId3",
				Type:        "erc20",
				PoolAddress: "poolAddress3",
			},
		}

		assert.ElementsMatch(t, expectedTokens, tokens)
		assert.Nil(t, err)
	})
}
