package repository

import (
	"context"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestTokenDatastoreRedisRepository_FindAll(t *testing.T) {
	t.Run("it should return all tokens in redis", func(t *testing.T) {
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

		repo := NewTokenDataStoreRedisRepository(db)

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
		}

		for _, token := range redisTokens {
			redisServer.HSet(":tokens", token.Address, token.Encode())
		}

		tokens, err := repo.FindAll(context.Background())

		assert.ElementsMatch(t, redisTokens, tokens)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down ", func(t *testing.T) {
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

		repo := NewTokenDataStoreRedisRepository(db)

		redisServer.Close()

		tokens, err := repo.FindAll(context.Background())

		assert.Nil(t, tokens)
		assert.Error(t, err)
	})
}

func TestTokenDatastoreRedisRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := NewTokenDataStoreRedisRepository(nil)

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

		repo := NewTokenDataStoreRedisRepository(db)

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
		}

		assert.ElementsMatch(t, expectedTokens, tokens)
		assert.Nil(t, err)
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

		repo := NewTokenDataStoreRedisRepository(db)

		redisServer.Close()

		tokens, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, tokens)
		assert.Error(t, err)
	})
}

func TestTokenDatastoreRedisRepository_Persist(t *testing.T) {
	t.Run("it should persist data correctly", func(t *testing.T) {
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

		repo := NewTokenDataStoreRedisRepository(db)

		theToken := entity.Token{
			Address:     "address1",
			Symbol:      "symbol1",
			Name:        "name1",
			Decimals:    18,
			CgkID:       "cgkId1",
			Type:        "erc20",
			PoolAddress: "poolAddress1",
		}

		err = repo.Persist(context.Background(), theToken)

		encodedToken := redisServer.HGet(":tokens", "address1")

		assert.Nil(t, err)
		assert.Equal(t, theToken.Encode(), encodedToken)
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

		repo := NewTokenDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.Persist(context.Background(), entity.Token{})

		assert.Error(t, err)
	})
}

func TestTokenDatastoreRedisRepository_Delete(t *testing.T) {
	t.Run("it should delete data correctly", func(t *testing.T) {
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

		repo := NewTokenDataStoreRedisRepository(db)

		theToken := entity.Token{
			Address: "address1",
		}

		err = repo.Delete(context.Background(), theToken)

		assert.Nil(t, err)

		encodedToken := redisServer.HGet(":tokens", "address1")

		assert.Empty(t, encodedToken)
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

		repo := NewTokenDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.Delete(context.Background(), entity.Token{Address: "address1"})

		assert.Error(t, err)
	})
}
