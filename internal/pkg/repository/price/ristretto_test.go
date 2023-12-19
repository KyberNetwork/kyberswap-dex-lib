package price

import (
	"context"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/pkg/redis"
)

func TestRistrettoRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return empty when addresses is empty", func(t *testing.T) {
		repo, err := NewRistrettoRepository(nil, RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,

			Price: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 1 * time.Minute},
		})

		assert.Nil(t, err)

		prices, err := repo.FindByAddresses(context.Background(), nil)

		assert.Empty(t, prices)
		assert.Nil(t, err)
	})

	t.Run("it should return correct prices when addresses are exists in redis or in memory cache", func(t *testing.T) {
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
		redisRepo := NewRedisRepository(db.Client, RedisRepositoryConfig{
			Prefix: "",
		})

		repo, err := NewRistrettoRepository(redisRepo, RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,

			Price: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{Cost: 1, TTL: 1 * time.Minute},
		})

		assert.Nil(t, err)

		// Prepare data test case 1 all token from cache
		redisPrices := []entity.Price{
			{
				Address: "address1",
				Price:   1,
			},
			{
				Address: "address2",
				Price:   2,
			},
			{
				Address: "address3",
				Price:   3,
			},
		}

		for _, price := range redisPrices {
			encodedPrice, _ := encodePrice(price)
			redisServer.HSet(":prices", price.Address, encodedPrice)
		}

		repo.cache.SetWithTTL("address4", &entity.Price{Address: "address4", Price: 4}, 1, 1*time.Minute)
		repo.cache.Wait()

		prices, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address3", "address4"})

		assert.Nil(t, err)

		expectedPrices := []*entity.Price{
			{
				Address: "address1",
				Price:   1,
			},
			{
				Address: "address2",
				Price:   2,
			},
			{
				Address: "address3",
				Price:   3,
			},
			{
				Address: "address4",
				Price:   4,
			},
		}

		assert.ElementsMatch(t, expectedPrices, prices)
		assert.Nil(t, err)
	})
}
