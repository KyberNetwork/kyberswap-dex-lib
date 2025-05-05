package token

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/pkg/redis"
)

func TestTokenCacheRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return empty when addresses is empty", func(t *testing.T) {
		repo, _ := NewGoCacheRepository(nil, RistrettoConfig{

			Token: struct {
				Expiration      time.Duration `mapstructure:"expiration"`
				CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
			}{
				Expiration:      10 * time.Second,
				CleanupInterval: 20 * time.Second,
			},

			Decimal: struct {
				Cost        int64 `mapstructure:"cost"`
				NumCounters int64 `mapstructure:"numCounters"`
				MaxCost     int64 `mapstructure:"maxCost"`
				BufferItems int64 `mapstructure:"bufferItems"`
			}{
				Cost:        1,
				NumCounters: 100,
				MaxCost:     10,
				BufferItems: 64,
			}})

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
		tokenDatastoreRepo := NewRepository(db.Client, RedisRepositoryConfig{
			Prefix: "",
		}, nil)

		repo, _ := NewGoCacheRepository(tokenDatastoreRepo, RistrettoConfig{

			Token: struct {
				Expiration      time.Duration `mapstructure:"expiration"`
				CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
			}{
				Expiration:      10 * time.Second,
				CleanupInterval: 20 * time.Second,
			},

			Decimal: struct {
				Cost        int64 `mapstructure:"cost"`
				NumCounters int64 `mapstructure:"numCounters"`
				MaxCost     int64 `mapstructure:"maxCost"`
				BufferItems int64 `mapstructure:"bufferItems"`
			}{
				Cost:        1,
				NumCounters: 100,
				MaxCost:     10,
				BufferItems: 64,
			}})

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

func TestTokenCacheRepository_FindDecimalByAddresses(t *testing.T) {
	t.Run("it should return empty when addresses is empty", func(t *testing.T) {
		repo, _ := NewGoCacheRepository(nil, RistrettoConfig{

			Token: struct {
				Expiration      time.Duration `mapstructure:"expiration"`
				CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
			}{
				Expiration:      10 * time.Second,
				CleanupInterval: 20 * time.Second,
			},

			Decimal: struct {
				Cost        int64 `mapstructure:"cost"`
				NumCounters int64 `mapstructure:"numCounters"`
				MaxCost     int64 `mapstructure:"maxCost"`
				BufferItems int64 `mapstructure:"bufferItems"`
			}{
				Cost:        1,
				NumCounters: 100,
				MaxCost:     10,
				BufferItems: 64,
			}})

		decimals, err := repo.FindDecimalByAddresses(context.Background(), nil)

		assert.Empty(t, decimals)
		assert.Nil(t, err)
	})

	t.Run("it should return correct decimal when addresses are exists in redis or in memory cache", func(t *testing.T) {
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
		tokenDatastoreRepo := NewRepository(db.Client, RedisRepositoryConfig{
			Prefix: "",
		}, nil)

		repo, _ := NewGoCacheRepository(tokenDatastoreRepo, RistrettoConfig{

			Token: struct {
				Expiration      time.Duration `mapstructure:"expiration"`
				CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
			}{
				Expiration:      10 * time.Second,
				CleanupInterval: 20 * time.Second,
			},

			Decimal: struct {
				Cost        int64 `mapstructure:"cost"`
				NumCounters int64 `mapstructure:"numCounters"`
				MaxCost     int64 `mapstructure:"maxCost"`
				BufferItems int64 `mapstructure:"bufferItems"`
			}{
				Cost:        1,
				NumCounters: 100,
				MaxCost:     10,
				BufferItems: 64,
			}})

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
				Decimals:    16,
				CgkID:       "cgkId2",
				Type:        "erc20",
				PoolAddress: "poolAddress2",
			},
			{
				Address:     "address3",
				Symbol:      "symbol3",
				Name:        "name3",
				Decimals:    6,
				CgkID:       "cgkId3",
				Type:        "erc20",
				PoolAddress: "poolAddress3",
			},
		}

		for _, token := range redisTokens {
			encodedToken, _ := encodeToken(token)
			redisServer.HSet(":tokens", token.Address, encodedToken)
		}

		decimals, err := repo.FindDecimalByAddresses(context.Background(), []string{"address1", "address2", "address3"})

		expectedDecimals := map[string]uint8{
			"address1": uint8(18),
			"address2": uint8(16),
			"address3": uint8(6),
		}

		assert.Equal(t, expectedDecimals, decimals)
		assert.Nil(t, err)
	})
}
