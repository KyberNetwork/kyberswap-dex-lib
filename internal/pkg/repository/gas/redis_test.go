package gas_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

type mockGasPricer struct {
}

func (m *mockGasPricer) SuggestGasPrice(_ context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

func TestRedisRepository_GetSuggestedGasPrice(t *testing.T) {
	t.Run("it should return suggested gas price", func(t *testing.T) {
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
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		mockGasPricer := &mockGasPricer{}
		redisServer.HSet(":metadata", "suggested_gas_price", "1")

		repo := gas.NewRedisRepository(db.Client, mockGasPricer, gas.RedisRepositoryConfig{Prefix: ""})
		gasPricer, err := repo.GetSuggestedGasPrice(context.Background())

		assert.Nil(t, err)
		assert.Equal(t, "1", gasPricer.String())
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
			t.Fatalf("failed to setup redis client: %v", err.Error())
		}

		mockGasPricer := &mockGasPricer{}

		repo := gas.NewRedisRepository(db.Client, mockGasPricer, gas.RedisRepositoryConfig{Prefix: ""})
		redisServer.Close()
		gasPricer, err := repo.GetSuggestedGasPrice(context.Background())

		assert.Error(t, err)
		assert.Nil(t, gasPricer)
	})
}
