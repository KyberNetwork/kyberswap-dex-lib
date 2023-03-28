package repository

import (
	"context"
	"strconv"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/redis"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"
)

func TestPriceDatastoreRedisRepository_FindAll(t *testing.T) {
	t.Run("it should return all prices in redis", func(t *testing.T) {
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
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

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
		}

		for _, price := range redisPrices {
			redisServer.HSet("avalanche:prices", price.Address, price.Encode())
		}

		prices, err := repo.FindAll(context.Background())

		assert.ElementsMatch(t, redisPrices, prices)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down ", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		redisServer.Close()

		prices, err := repo.FindAll(context.Background())

		assert.Nil(t, prices)
		assert.Error(t, err)
	})
}

func TestPriceDatastoreRedisRepository_FindByAddresses(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := NewPriceDataStoreRedisRepository(nil)

		prices, err := repo.FindByAddresses(context.Background(), nil)

		assert.Nil(t, prices)
		assert.Nil(t, err)
	})

	t.Run("it should return correct prices when addresses are exists in redis", func(t *testing.T) {
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
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

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

		expectedPrices := []entity.Price{
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

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		redisServer.Close()

		prices, err := repo.FindByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, prices)
		assert.Error(t, err)
	})
}

func TestPriceDatastoreRedisRepository_FindMapPriceByAddresses(t *testing.T) {
	t.Run("it should return nil when addresses is empty", func(t *testing.T) {
		repo := NewPriceDataStoreRedisRepository(nil)

		prices, err := repo.FindMapPriceByAddresses(context.Background(), nil)

		assert.Nil(t, prices)
		assert.Nil(t, err)
	})

	t.Run("it should return correct prices when addresses are exists in redis", func(t *testing.T) {
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
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

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

		priceByAddress, err := repo.FindMapPriceByAddresses(context.Background(), []string{"address1", "address2", "address4"})

		expectedPriceByAddress := map[string]float64{
			"address1": 10000,
			"address2": 20000,
		}

		assert.Equal(t, expectedPriceByAddress, priceByAddress)
		assert.Nil(t, err)
	})

	t.Run("it should prefer market price when it exists", func(t *testing.T) {
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

		repo := NewPriceDataStoreRedisRepository(db)

		// Prepare data
		redisPrices := []entity.Price{
			{
				Address:     "address1",
				Price:       10000,
				Liquidity:   10000,
				LpAddress:   "lpAddress1",
				MarketPrice: 11111,
			},
			{
				Address:     "address2",
				Price:       20000,
				Liquidity:   20000,
				LpAddress:   "lpAddress2",
				MarketPrice: 22222,
			},
		}

		for _, price := range redisPrices {
			redisServer.HSet(":prices", price.Address, price.Encode())
		}

		priceByAddress, err := repo.FindMapPriceByAddresses(context.Background(), []string{"address1", "address2"})

		expectedPriceByAddress := map[string]float64{
			"address1": 11111,
			"address2": 22222,
		}

		assert.Equal(t, expectedPriceByAddress, priceByAddress)
		assert.Nil(t, err)
	})

	t.Run("it should prefer our calculated price when market price does not exist", func(t *testing.T) {
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

		repo := NewPriceDataStoreRedisRepository(db)

		// Prepare data
		redisPrices := []entity.Price{
			{
				Address:     "address1",
				Price:       10000,
				Liquidity:   10000,
				LpAddress:   "lpAddress1",
				MarketPrice: 0,
			},
			{
				Address:   "address2",
				Price:     20000,
				Liquidity: 20000,
				LpAddress: "lpAddress2",
			},
			{
				Address:     "address3",
				Price:       30000,
				Liquidity:   30000,
				LpAddress:   "lpAddress3",
				MarketPrice: 0,
			},
		}

		for _, price := range redisPrices {
			redisServer.HSet(":prices", price.Address, price.Encode())
		}

		priceByAddress, err := repo.FindMapPriceByAddresses(context.Background(), []string{"address1", "address2", "address3"})

		expectedPriceByAddress := map[string]float64{
			"address1": 10000,
			"address2": 20000,
			"address3": 30000,
		}

		assert.Equal(t, expectedPriceByAddress, priceByAddress)
		assert.Nil(t, err)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		redisServer.Close()

		prices, err := repo.FindMapPriceByAddresses(context.Background(), []string{"address1"})

		assert.Nil(t, prices)
		assert.Error(t, err)
	})
}

func TestPriceDatastoreRedisRepository_Persist(t *testing.T) {
	t.Run("it should persist data correctly", func(t *testing.T) {
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
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		thePrice := entity.Price{
			Address:     "address1",
			Price:       10000,
			Liquidity:   10000,
			LpAddress:   "lpAddress1",
			MarketPrice: 10000,
		}

		err = repo.Persist(context.Background(), thePrice)

		encodedPrice := redisServer.HGet("avalanche:prices", "address1")

		assert.Nil(t, err)
		assert.Equal(t, thePrice.Encode(), encodedPrice)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.Persist(context.Background(), entity.Price{})

		assert.Error(t, err)
	})
}

func TestPriceDatastoreRedisRepository_Delete(t *testing.T) {
	t.Run("it should delete data correctly", func(t *testing.T) {
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
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

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

		repo := NewPriceDataStoreRedisRepository(db)

		thePrice := entity.Price{
			Address: "address1",
		}

		err = repo.Delete(context.Background(), thePrice)

		assert.Nil(t, err)

		encodedPrice := redisServer.HGet("avalanche:prices", "address1")

		assert.Empty(t, encodedPrice)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		// Setup redis client
		port, err := strconv.Atoi(redisServer.Port())
		if err != nil {
			t.Fatalf("failed to convert redis port: %v", err.Error())
		}

		redisConfig := &redis.Config{
			Host:   redisServer.Host(),
			Port:   port,
			Prefix: "avalanche",
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.Delete(context.Background(), entity.Price{Address: "address1"})

		assert.Error(t, err)
	})
}

func TestPriceDatastoreRedisRepository_DeleteMultiple(t *testing.T) {
	t.Run("it should delete multiple records correctly", func(t *testing.T) {
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
			redisServer.HSet(":prices", price.Address, price.Encode())
		}

		repo := NewPriceDataStoreRedisRepository(db)

		err = repo.DeleteMultiple(context.Background(), redisPrices)

		assert.Nil(t, err)

		encodedPrice1 := redisServer.HGet(":prices", "address1")
		encodedPrice2 := redisServer.HGet(":prices", "address2")
		encodedPrice3 := redisServer.HGet(":prices", "address3")

		assert.Empty(t, encodedPrice1)
		assert.Empty(t, encodedPrice2)
		assert.Empty(t, encodedPrice3)
	})

	t.Run("it should return error when redis server is down", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

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

		repo := NewPriceDataStoreRedisRepository(db)

		redisServer.Close()

		err = repo.DeleteMultiple(context.Background(), []entity.Price{{
			Address:     "address1",
			Price:       10000,
			Liquidity:   10000,
			LpAddress:   "lpAddress1",
			MarketPrice: 10000,
		}})

		assert.Error(t, err)
	})
}
