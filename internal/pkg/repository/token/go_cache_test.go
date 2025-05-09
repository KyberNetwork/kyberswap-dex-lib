package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	tokenPkg "github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

func TestTokenCacheRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return empty when addresses is empty", func(t *testing.T) {
		repo, _ := tokenPkg.NewGoCacheRepository(nil, &tokenPkg.RistrettoConfig{

			Token: struct {
				Cost        int64         `mapstructure:"cost"`
				NumCounters int64         `mapstructure:"numCounters"`
				MaxCost     int64         `mapstructure:"maxCost"`
				BufferItems int64         `mapstructure:"bufferItems"`
				TTL         time.Duration `mapstructure:"ttl"`
			}{
				Cost:        1,
				NumCounters: 100,
				MaxCost:     10,
				BufferItems: 64,
				TTL:         3 * time.Minute,
			},

			TokenInfo: struct {
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
		tokenDatastoreRepo := tokenPkg.NewSimplifiedTokenRepository(db.Client, tokenPkg.RedisRepositoryConfig{
			Prefix: "",
		}, nil)

		repo, _ := tokenPkg.NewGoCacheRepository(tokenDatastoreRepo, &tokenPkg.RistrettoConfig{

			Token: struct {
				Cost        int64         `mapstructure:"cost"`
				NumCounters int64         `mapstructure:"numCounters"`
				MaxCost     int64         `mapstructure:"maxCost"`
				BufferItems int64         `mapstructure:"bufferItems"`
				TTL         time.Duration `mapstructure:"ttl"`
			}{
				Cost:        1,
				NumCounters: 100,
				MaxCost:     10,
				BufferItems: 64,
				TTL:         3 * time.Minute,
			},

			TokenInfo: struct {
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

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1", "address2", "address3"})

		expectedTokens := []*entity.SimplifiedToken{
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

		assert.ElementsMatch(t, expectedTokens, tokens)
		assert.Nil(t, err)
	})
}
