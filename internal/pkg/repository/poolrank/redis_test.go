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
			_ = repo.AddToSortedSetScoreByTvl(
				context.Background(),
				pool,
				"poolTokenAddress1",
				"poolTokenAddress2",
				true,
				true,
			)
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
			_ = repo.AddToSortedSetScoreByTvl(
				context.Background(),
				pool,
				"poolTokenAddress1",
				"poolTokenAddress2",
				true,
				false,
			)
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
			_ = repo.AddToSortedSetScoreByTvl(
				context.Background(),
				pool,
				"poolTokenAddress1",
				"poolTokenAddress2",
				false,
				true,
			)
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
			_ = repo.AddToSortedSetScoreByTvl(
				context.Background(),
				pool,
				"poolTokenAddress1",
				"poolTokenAddress2",
				false,
				false,
			)
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

		err = repo.AddToSortedSetScoreByTvl(context.Background(), p, token1Address, token2Address, true, true)

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

		err = repo.AddToSortedSetScoreByTvl(context.Background(), p, token1Address, token2Address, true, false)

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

		err = repo.AddToSortedSetScoreByTvl(context.Background(), p, token1Address, token2Address, false, true)

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

		err = repo.AddToSortedSetScoreByTvl(context.Background(), p, token1Address, token2Address, false, false)

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

		err = repo.AddToSortedSetScoreByTvl(context.Background(), &entity.Pool{}, token1Address, token2Address, true, true)

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

		err = repo.AddToSortedSetScoreByAmplifiedTvl(context.Background(), p, token1Address, token2Address, true, true)

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

		err = repo.AddToSortedSetScoreByAmplifiedTvl(context.Background(), p, token1Address, token2Address, true, false)

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

		err = repo.AddToSortedSetScoreByAmplifiedTvl(context.Background(), p, token1Address, token2Address, false, true)

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

		err = repo.AddToSortedSetScoreByAmplifiedTvl(context.Background(), p, token1Address, token2Address, false, false)

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

		err = repo.AddToSortedSetScoreByAmplifiedTvl(context.Background(), &entity.Pool{}, token1Address, token2Address, true, true)

		assert.Error(t, err)
	})
}
