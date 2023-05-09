package price_test

import (
	"context"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestRedisRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return correct prices when addresses are exists in redis", func(t *testing.T) {
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}
		defer redisServer.Close()

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		repoConfig := price.RedisRepositoryConfig{
			Prefix: "avalanche",
		}
		repo := price.NewRedisRepository(db.Client, repoConfig)
		// Prepare data
		redisPrices := []entity.Price{
			{
				Address:     "address1",
				Price:       10000,
				Liquidity:   10000,
				LpAddress:   "lpAddress1",
				MarketPrice: 10000,
			},
			{
				Address:     "address2",
				Price:       20000,
				Liquidity:   20000,
				LpAddress:   "lpAddress2",
				MarketPrice: 20000,
			},
			{
				Address:     "address3",
				Price:       30000,
				Liquidity:   30000,
				LpAddress:   "lpAddress3",
				MarketPrice: 30000,
			},
		}

		for _, price := range redisPrices {
			redisServer.HSet("avalanche:prices", price.Address, price.Encode())
		}

		prices, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address4"})

		expectedPrices := []*entity.Price{
			{
				Address:     "address1",
				Price:       10000,
				Liquidity:   10000,
				LpAddress:   "lpAddress1",
				MarketPrice: 10000,
			},
			{
				Address:     "address2",
				Price:       20000,
				Liquidity:   20000,
				LpAddress:   "lpAddress2",
				MarketPrice: 20000,
			},
		}

		assert.ElementsMatch(t, expectedPrices, prices)
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
			t.Fatalf("failed to setup redis client for testing: %v", err.Error())
		}

		repoConfig := price.RedisRepositoryConfig{
			Prefix: "",
		}
		repo := price.NewRedisRepository(db.Client, repoConfig)
		prices, err := repo.FindByAddresses(context.Background(), []string{})

		assert.Nil(t, prices)
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
			t.Fatalf("failed to setup redis client for testing: %v", err.Error())
		}

		repoConfig := price.RedisRepositoryConfig{
			Prefix: "",
		}
		repo := price.NewRedisRepository(db.Client, repoConfig)
		redisServer.Close()
		prices, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Error(t, err)
		assert.Nil(t, prices)
	})
}
