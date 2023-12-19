package gas

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/pkg/redis"
)

func TestRistrettoRepository_GetSuggestedGasPrice(t *testing.T) {
	t.Run("it should return correct gasPrice in memory cache", func(t *testing.T) {
		repo, err := NewRistrettoRepository(nil, RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,
			SuggestedGasPrice: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{
				Cost: 1,
				TTL:  2 * time.Minute,
			},
		})

		assert.Nil(t, err)

		repo.cache.SetWithTTL(CacheKeySuggestedGasPrice, big.NewInt(1), repo.config.SuggestedGasPrice.Cost, repo.config.SuggestedGasPrice.TTL)
		repo.cache.Wait()

		suggestedGasPrice, err := repo.GetSuggestedGasPrice(context.Background())

		assert.Nil(t, err)
		assert.Equal(t, 0, suggestedGasPrice.Cmp(big.NewInt(1)))
	})

	t.Run("it should return correct gasPrice in redis", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		redisServer.HSet(":metadata", "suggested_gas_price", "100")

		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    "",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}
		redisRepo := NewRedisRepository(db.Client, nil, RedisRepositoryConfig{Prefix: ""})
		repo, err := NewRistrettoRepository(redisRepo, RistrettoConfig{
			NumCounters: 5000,
			MaxCost:     500,
			BufferItems: 64,
			SuggestedGasPrice: struct {
				Cost int64         `mapstructure:"cost"`
				TTL  time.Duration `mapstructure:"ttl"`
			}{
				Cost: 1,
				TTL:  2 * time.Minute,
			},
		})

		assert.Nil(t, err)

		suggestedGasPrice, err := repo.GetSuggestedGasPrice(context.Background())

		assert.Nil(t, err)
		assert.Equal(t, 0, suggestedGasPrice.Cmp(big.NewInt(100)))
	})
}
