package token_test

import (
	"context"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestRedisRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := token.NewRedisRepository(nil, token.RedisRepositoryConfig{})

		tokens, err := repo.FindByAddresses(context.Background(), nil)

		assert.Nil(t, tokens)
		assert.Nil(t, err)
	})

	t.Run("it should return correct tokens when addresses are exists in redis", func(t *testing.T) {
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

		repo := token.NewRedisRepository(db.Client, token.RedisRepositoryConfig{Prefix: ""})

		// Prepare data
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

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address4"})

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
		}

		assert.Nil(t, err)
		assert.ElementsMatch(t, expectedTokens, tokens)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
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

		repo := token.NewRedisRepository(db.Client, token.RedisRepositoryConfig{Prefix: ""})

		redisServer.Close()

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, tokens)
		assert.Error(t, err)
	})
}
