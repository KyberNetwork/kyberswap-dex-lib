package token_test

import (
	"context"
	"testing"

	tokenPkg "github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestRedisRepository_FindByAddresses_SimplifedToken(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := tokenPkg.NewSimplifiedTokenRepository(nil, tokenPkg.RedisRepositoryConfig{}, nil)

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

		repo := tokenPkg.NewSimplifiedTokenRepository(db.Client, tokenPkg.RedisRepositoryConfig{Prefix: ""}, nil)

		// Prepare data
		redisTokens := []entity.SimplifiedToken{
			{
				Address:  "address1",
				Decimals: 18,
			},
			{
				Address:  "address2",
				Decimals: 18,
			},
			{
				Address:  "address3",
				Decimals: 18,
			},
		}

		for _, token := range redisTokens {
			encodedToken, _ := tokenPkg.EncodeToken(token)
			redisServer.HSet(":tokens", token.Address, encodedToken)
		}

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address4"})

		expectedTokens := []*entity.SimplifiedToken{
			{
				Address:  "address1",
				Decimals: 18,
			},
			{
				Address:  "address2",
				Decimals: 18,
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

		repo := tokenPkg.NewSimplifiedTokenRepository(db.Client, tokenPkg.RedisRepositoryConfig{Prefix: ""}, nil)

		redisServer.Close()

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, tokens)
		assert.Error(t, err)
	})
}

func TestRedisRepository_FindByAddresses_FullToken(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := tokenPkg.NewFullTokenRepository(nil, tokenPkg.RedisRepositoryConfig{}, nil)

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

		repo := tokenPkg.NewFullTokenRepository(db.Client, tokenPkg.RedisRepositoryConfig{Prefix: ""}, nil)

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
			encodedToken, _ := tokenPkg.EncodeToken(token)
			redisServer.HSet(":tokens", token.Address, encodedToken)
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

		repo := tokenPkg.NewFullTokenRepository(db.Client, tokenPkg.RedisRepositoryConfig{Prefix: ""}, nil)

		redisServer.Close()

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, tokens)
		assert.Error(t, err)
	})
}
