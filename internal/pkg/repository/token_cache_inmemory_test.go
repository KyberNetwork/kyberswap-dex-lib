package repository

import (
	"context"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/patrickmn/go-cache"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

func TestTokenCacheInmemoryRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return empty when addresses is empty", func(t *testing.T) {
		repo := NewTokenCacheRepository(nil, cache.New(cache.NoExpiration, cache.NoExpiration))

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
		tokenDatastoreRepo := NewTokenDataStoreRedisRepository(db)

		repo := NewTokenCacheRepository(tokenDatastoreRepo, cache.New(cache.NoExpiration, cache.NoExpiration))

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
			redisServer.HSet(":tokens", token.Address, token.Encode())
		}

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address3"})

		expectedTokens := []entity.Token{
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
