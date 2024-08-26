package poolrank_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

func wrap(cfg poolrank.RedisRepositoryConfig) poolrank.Config {
	return poolrank.Config{
		Redis:            cfg,
		UseNativeRanking: false,
	}
}

func TestRedisRepository_FindBestPoolIDs(t *testing.T) {
	t.Run("it should return correct data when both tokens in pool are in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{
			Prefix: "",
		}))

		// Prepare data
		redisPools := []*entity.Pool{
			{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			{
				Address:      "address2",
				ReserveUsd:   1000,
				AmplifiedTvl: 1000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra2",
				StaticExtra: "staticExtra2",
				TotalSupply: "totalSupply2",
			},
			{
				Address:      "address3",
				ReserveUsd:   10000,
				AmplifiedTvl: 10000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra3",
				StaticExtra: "staticExtra3",
				TotalSupply: "totalSupply3",
			},
		}

		for _, pool := range redisPools {
			_ = repo.AddToSortedSet(context.Background(), "poolTokenAddress1", "poolTokenAddress2",
				true, true, poolrank.SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			valueobject.GetBestPoolsOptions{
				DirectPoolsCount:    100,
				WhitelistPoolsCount: 500,
				TokenInPoolsCount:   200,
				TokenOutPoolCount:   200,

				AmplifiedTvlDirectPoolsCount:    50,
				AmplifiedTvlWhitelistPoolsCount: 200,
				AmplifiedTvlTokenInPoolsCount:   100,
				AmplifiedTvlTokenOutPoolCount:   100,
			},
		)

		assert.ElementsMatch(t, []string{"address1", "address2", "address3"}, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return correct data when only token1 is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{
			Prefix: "",
		}))

		// Prepare data
		redisPools := []*entity.Pool{
			{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			{
				Address:      "address2",
				ReserveUsd:   1000,
				AmplifiedTvl: 1000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra2",
				StaticExtra: "staticExtra2",
				TotalSupply: "totalSupply2",
			},
			{
				Address:      "address3",
				ReserveUsd:   10000,
				AmplifiedTvl: 10000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra3",
				StaticExtra: "staticExtra3",
				TotalSupply: "totalSupply3",
			},
		}

		for _, pool := range redisPools {
			_ = repo.AddToSortedSet(context.Background(), "poolTokenAddress1", "poolTokenAddress2",
				true, false, poolrank.SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			valueobject.GetBestPoolsOptions{
				DirectPoolsCount:    100,
				WhitelistPoolsCount: 500,
				TokenInPoolsCount:   200,
				TokenOutPoolCount:   200,

				AmplifiedTvlDirectPoolsCount:    50,
				AmplifiedTvlWhitelistPoolsCount: 200,
				AmplifiedTvlTokenInPoolsCount:   100,
				AmplifiedTvlTokenOutPoolCount:   100,
			},
		)

		assert.ElementsMatch(t, []string{"address1", "address2", "address3"}, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return correct data when only token2 is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		// Prepare data
		redisPools := []*entity.Pool{
			{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			{
				Address:      "address2",
				ReserveUsd:   1000,
				AmplifiedTvl: 1000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra2",
				StaticExtra: "staticExtra2",
				TotalSupply: "totalSupply2",
			},
			{
				Address:      "address3",
				ReserveUsd:   10000,
				AmplifiedTvl: 10000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra3",
				StaticExtra: "staticExtra3",
				TotalSupply: "totalSupply3",
			},
		}

		for _, pool := range redisPools {
			_ = repo.AddToSortedSet(context.Background(), "poolTokenAddress1", "poolTokenAddress2",
				false, true, poolrank.SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			valueobject.GetBestPoolsOptions{
				DirectPoolsCount:    100,
				WhitelistPoolsCount: 500,
				TokenInPoolsCount:   200,
				TokenOutPoolCount:   200,

				AmplifiedTvlDirectPoolsCount:    50,
				AmplifiedTvlWhitelistPoolsCount: 200,
				AmplifiedTvlTokenInPoolsCount:   100,
				AmplifiedTvlTokenOutPoolCount:   100,
			},
		)

		assert.ElementsMatch(t, []string{"address1", "address2", "address3"}, pools)
		assert.Nil(t, err)
	})

	t.Run("it should return correct data when no token is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		// Prepare data
		redisPools := []*entity.Pool{
			{
				Address:      "address1",
				ReserveUsd:   100,
				AmplifiedTvl: 100,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra1",
				StaticExtra: "staticExtra1",
				TotalSupply: "totalSupply1",
			},
			{
				Address:      "address2",
				ReserveUsd:   1000,
				AmplifiedTvl: 1000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra2",
				StaticExtra: "staticExtra2",
				TotalSupply: "totalSupply2",
			},
			{
				Address:      "address3",
				ReserveUsd:   10000,
				AmplifiedTvl: 10000,
				SwapFee:      0.3,
				Exchange:     "",
				Type:         "uni",
				Timestamp:    12345,
				Reserves:     []string{"reserve1, reserve2"},
				Tokens: []*entity.PoolToken{
					{
						Address:   "poolTokenAddress1",
						Name:      "poolTokenName1",
						Symbol:    "poolTokenSymbol1",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "poolTokenAddress2",
						Name:      "poolTokenName2",
						Symbol:    "poolTokenSymbol2",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "extra3",
				StaticExtra: "staticExtra3",
				TotalSupply: "totalSupply3",
			},
		}

		for _, pool := range redisPools {
			_ = repo.AddToSortedSet(context.Background(), "poolTokenAddress1", "poolTokenAddress2",
				false, false, poolrank.SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			valueobject.GetBestPoolsOptions{
				DirectPoolsCount:    100,
				WhitelistPoolsCount: 500,
				TokenInPoolsCount:   200,
				TokenOutPoolCount:   200,

				AmplifiedTvlDirectPoolsCount:    50,
				AmplifiedTvlWhitelistPoolsCount: 200,
				AmplifiedTvlTokenInPoolsCount:   100,
				AmplifiedTvlTokenOutPoolCount:   100,
			},
		)

		assert.ElementsMatch(t, []string{"address1", "address2", "address3"}, pools)
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		redisServer.Close()

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			valueobject.GetBestPoolsOptions{
				DirectPoolsCount:    100,
				WhitelistPoolsCount: 500,
				TokenInPoolsCount:   200,
				TokenOutPoolCount:   200,

				AmplifiedTvlDirectPoolsCount:    50,
				AmplifiedTvlWhitelistPoolsCount: 200,
				AmplifiedTvlTokenInPoolsCount:   100,
				AmplifiedTvlTokenOutPoolCount:   100,
			},
		)

		assert.Nil(t, pools)
		assert.Error(t, err)
	})
}

func TestRedisRepository_AddToSortedSetScoreByTvl(t *testing.T) {
	t.Run("it should set data correctly when both tokens in pool are in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)

		// directKeyPair: :tvl:poolTokenAddress2-poolTokenAddress1
		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		// All the sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.Equal(t, float64(100), whitelistPools["address1"])
		assert.Equal(t, float64(100), token1Pools["address1"])
		assert.Equal(t, float64(100), token2Pools["address1"])
	})

	t.Run("it should set data correctly when only token1 is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, false, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		// Only the direct pair and whitelist-token2 sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, whitelistPools)
		assert.ElementsMatch(t, map[string]float64{}, token1Pools)
		assert.Equal(t, float64(100), token2Pools["address1"])
	})

	t.Run("it should set data correctly when only token2 is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, true, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		// Only the direct pair and whitelist-token1 sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, whitelistPools)
		assert.Equal(t, float64(100), token1Pools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, token2Pools)

	})

	t.Run("it should set data correctly when no token in pool is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, false, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)

		assert.Nil(t, err)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, token2Address))

		// Only the direct pair sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, whitelistPools)
		assert.ElementsMatch(t, map[string]float64{}, token1Pools)
		assert.ElementsMatch(t, map[string]float64{}, token2Pools)
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		redisServer.Close()

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		pool := &entity.Pool{}
		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, poolrank.SortByTVL, pool.Address, pool.ReserveUsd, true)

		assert.Error(t, err)
	})
}

func TestRedisRepository_AddToSortedSetScoreByAmplifiedTvl(t *testing.T) {
	t.Run("it should set data correctly when both tokens in pool are in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		fmt.Println(directPools)
		fmt.Println(whitelistPools)
		fmt.Println(token1Pools)
		fmt.Println(token2Pools)

		// All the sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.Equal(t, float64(100), whitelistPools["address1"])
		assert.Equal(t, float64(100), token1Pools["address1"])
		assert.Equal(t, float64(100), token2Pools["address1"])
	})

	t.Run("it should set data correctly when only token1 is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, false, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		// Only the direct pair and whitelist-token2 sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, whitelistPools)
		assert.ElementsMatch(t, map[string]float64{}, token1Pools)
		assert.Equal(t, float64(100), token2Pools["address1"])
	})

	t.Run("it should set data correctly when only token2 is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, true, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		// Only the direct pair and whitelist-token1 sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, whitelistPools)
		assert.Equal(t, float64(100), token1Pools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, token2Pools)

	})

	t.Run("it should set data correctly when no token in pool is in whitelist", func(t *testing.T) {
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		p := &entity.Pool{
			Address:      "address1",
			ReserveUsd:   100,
			AmplifiedTvl: 100,
			SwapFee:      0.3,
			Exchange:     "",
			Type:         "uni",
			Timestamp:    12345,
			Reserves:     []string{"reserve1, reserve2"},
			Tokens: []*entity.PoolToken{
				{
					Address:   "poolTokenAddress1",
					Name:      "poolTokenName1",
					Symbol:    "poolTokenSymbol1",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "poolTokenAddress2",
					Name:      "poolTokenName2",
					Symbol:    "poolTokenSymbol2",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "extra1",
			StaticExtra: "staticExtra1",
			TotalSupply: "totalSupply1",
		}

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, false, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, token2Address))

		assert.Nil(t, err)

		// Only the direct pair sorted sets should contain this pool "address1"
		assert.Equal(t, float64(100), directPools["address1"])
		assert.ElementsMatch(t, map[string]float64{}, whitelistPools)
		assert.ElementsMatch(t, map[string]float64{}, token1Pools)
		assert.ElementsMatch(t, map[string]float64{}, token2Pools)
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

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: ""}))

		redisServer.Close()

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, "", poolrank.SortByAmplifiedTvl, 0, false)

		assert.Error(t, err)
	})
}

func TestRedisRepository_RemoveFromSortedSet(t *testing.T) {
	t.Run("it should remove data correctly when both tokens in pool are in whitelist, amplifiedTvl set", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}

		defer redisServer.Close()

		prefix := "ethereum"
		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    prefix,
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: prefix}))
		p := &entity.Pool{
			Address:      "pooladdress2",
			ReserveUsd:   20000,
			AmplifiedTvl: 100,
			SwapFee:      200,
			Reserves:     []string{"20000", "30000"},
		}

		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)
		assert.Nil(t, err)
		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)
		assert.Nil(t, err)

		// assert data before delete
		expectedTvlScore := map[string]float64{"pooladdress2": 20000}
		expectedAmplifiedTvlScore := map[string]float64{"pooladdress2": 100}

		directPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		directPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsTvl, expectedTvlScore)

		globalTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s", poolrank.SortByTVL))
		assert.Equal(t, globalTvl, expectedTvlScore)

		whitelistPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		assert.Equal(t, whitelistPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		assert.Equal(t, whitelistPoolsTvl, expectedTvlScore)

		whitelistToken1AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken1Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1Tvl, expectedTvlScore)

		whitelistToken2AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken2Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2Tvl, expectedTvlScore)

		err = repo.RemoveFromSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)
		assert.Nil(t, err)
		repo.RemoveFromSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)
		assert.Nil(t, err)

		// asset data after delete
		directPoolsAmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, "tokenaddress2-tokenaddress1"))
		assert.Nil(t, directPoolsAmplifiedTvl)
		directPoolsTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, "tokenaddress2-tokenaddress1"))
		assert.Nil(t, directPoolsTvl)

		globalTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s", poolrank.SortByTVL))
		assert.Nil(t, globalTvl)

		whitelistPoolsAmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		assert.Nil(t, whitelistPoolsAmplifiedTvl)
		whitelistPoolsTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		assert.Nil(t, whitelistPoolsTvl)

		whitelistToken1AmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, "tokenaddress1"))
		assert.Nil(t, whitelistToken1AmplifiedTvl)
		whitelistToken1Tvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, "tokenaddress1"))
		assert.Nil(t, whitelistToken1Tvl)

		whitelistToken2AmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, "tokenaddress2"))
		assert.Nil(t, whitelistToken2AmplifiedTvl)
		whitelistToken2Tvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, "tokenaddress2"))
		assert.Nil(t, whitelistToken2Tvl)

	})
}

func TestRedisRepository_RemoveAddressFromIndex(t *testing.T) {
	t.Run("it should remove pools from both sorted set global and whitlist whitelist set", func(t *testing.T) {
		// Setup redis server
		redisServer, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to setup redis for testing: %v", err.Error())
		}
		defer redisServer.Close()

		prefix := "ethereum"
		redisConfig := &redis.Config{
			Addresses: []string{redisServer.Addr()},
			Prefix:    prefix,
		}

		db, err := redis.New(redisConfig)
		if err != nil {
			t.Fatalf("failed to init redis client: %v", err.Error())
		}

		repo := poolrank.NewRedisRepository(db.Client, wrap(poolrank.RedisRepositoryConfig{Prefix: prefix}))
		p := &entity.Pool{
			Address:      "pooladdress2",
			ReserveUsd:   20000,
			AmplifiedTvl: 100,
			Reserves:     []string{"20000", "30000"},
		}

		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, poolrank.SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)
		assert.Nil(t, err)
		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, poolrank.SortByTVL, p.Address, p.ReserveUsd, true)
		assert.Nil(t, err)

		// assert data before delete
		expectedTvlScore := map[string]float64{"pooladdress2": 20000}
		expectedAmplifiedTvlScore := map[string]float64{"pooladdress2": 100}

		directPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		directPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsTvl, expectedTvlScore)

		globalTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s", poolrank.SortByTVL))
		assert.Equal(t, globalTvl, expectedTvlScore)

		whitelistPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		assert.Equal(t, whitelistPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		assert.Equal(t, whitelistPoolsTvl, expectedTvlScore)

		whitelistToken1AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken1Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1Tvl, expectedTvlScore)

		whitelistToken2AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken2Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2Tvl, expectedTvlScore)

		err = repo.RemoveAddressFromIndex(context.TODO(), poolrank.SortByTVL, []string{"pooladdress2"})
		assert.Nil(t, err)
		err = repo.RemoveAddressFromIndex(context.TODO(), poolrank.SortByAmplifiedTvl, []string{"pooladdress2"})
		assert.Nil(t, err)

		// asset data after delete
		globalTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s", poolrank.SortByTVL))
		assert.Nil(t, globalTvl)

		whitelistPoolsAmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByAmplifiedTvl, poolrank.KeyWhitelist))
		assert.Nil(t, whitelistPoolsAmplifiedTvl)
		whitelistPoolsTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", poolrank.SortByTVL, poolrank.KeyWhitelist))
		assert.Nil(t, whitelistPoolsTvl)

	})
}
