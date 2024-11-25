package poolrank

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/redis"
	redisClient "github.com/redis/go-redis/v9"
)

func wrap(cfg RedisRepositoryConfig) Config {
	return Config{
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{
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
				true, true, SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			0,
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
			valueobject.NativeTvl,
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{
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
				true, false, SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			0,
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
			valueobject.NativeTvl,
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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
				false, true, SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			0,
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
			valueobject.NativeTvl,
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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
				false, false, SortByTVL, pool.Address, pool.ReserveUsd, true)
		}

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			0,
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
			valueobject.NativeTvl,
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

		redisServer.Close()

		pools, err := repo.FindBestPoolIDs(
			context.Background(),
			"poolTokenAddress1",
			"poolTokenAddress2",
			0,
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
			valueobject.NativeTvl,
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, SortByTVL, p.Address, p.ReserveUsd, true)

		// directKeyPair: :tvl:poolTokenAddress2-poolTokenAddress1
		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, false, SortByTVL, p.Address, p.ReserveUsd, true)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, true, SortByTVL, p.Address, p.ReserveUsd, true)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		_ = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, false, SortByTVL, p.Address, p.ReserveUsd, true)

		assert.Nil(t, err)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByTVL, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByTVL, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

		redisServer.Close()

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		pool := &entity.Pool{}
		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, SortByTVL, pool.Address, pool.ReserveUsd, true)

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, false, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, true, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

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

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, false, false, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)

		directPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, "poolTokenAddress2-poolTokenAddress1"))
		whitelistPools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		token1Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token1Address))
		token2Pools, _ := redisServer.SortedSet(fmt.Sprintf(":%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, token2Address))

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: ""}))

		redisServer.Close()

		token1Address := "poolTokenAddress1"
		token2Address := "poolTokenAddress2"

		err = repo.AddToSortedSet(context.Background(), token1Address, token2Address, true, true, "", SortByAmplifiedTvl, 0, false)

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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: prefix}))
		p := &entity.Pool{
			Address:      "pooladdress2",
			ReserveUsd:   20000,
			AmplifiedTvl: 100,
		}

		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)
		assert.Nil(t, err)
		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, SortByTVL, p.Address, p.ReserveUsd, true)
		assert.Nil(t, err)

		// assert data before delete
		expectedTvlScore := map[string]float64{"pooladdress2": 20000}
		expectedAmplifiedTvlScore := map[string]float64{"pooladdress2": 100}

		directPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		directPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsTvl, expectedTvlScore)

		globalTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s", SortByTVL))
		assert.Equal(t, globalTvl, expectedTvlScore)

		whitelistPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		assert.Equal(t, whitelistPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, KeyWhitelist))
		assert.Equal(t, whitelistPoolsTvl, expectedTvlScore)

		whitelistToken1AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken1Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByTVL, KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1Tvl, expectedTvlScore)

		whitelistToken2AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken2Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByTVL, KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2Tvl, expectedTvlScore)

		err = repo.RemoveFromSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)
		assert.Nil(t, err)
		repo.RemoveFromSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, SortByTVL, p.Address, p.ReserveUsd, true)
		assert.Nil(t, err)

		// asset data after delete
		directPoolsAmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, "tokenaddress2-tokenaddress1"))
		assert.Nil(t, directPoolsAmplifiedTvl)
		directPoolsTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, "tokenaddress2-tokenaddress1"))
		assert.Nil(t, directPoolsTvl)

		globalTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s", SortByTVL))
		assert.Nil(t, globalTvl)

		whitelistPoolsAmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		assert.Nil(t, whitelistPoolsAmplifiedTvl)
		whitelistPoolsTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, KeyWhitelist))
		assert.Nil(t, whitelistPoolsTvl)

		whitelistToken1AmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, "tokenaddress1"))
		assert.Nil(t, whitelistToken1AmplifiedTvl)
		whitelistToken1Tvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByTVL, KeyWhitelist, "tokenaddress1"))
		assert.Nil(t, whitelistToken1Tvl)

		whitelistToken2AmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, "tokenaddress2"))
		assert.Nil(t, whitelistToken2AmplifiedTvl)
		whitelistToken2Tvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByTVL, KeyWhitelist, "tokenaddress2"))
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

		repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{Prefix: prefix}))
		p := &entity.Pool{
			Address:      "pooladdress2",
			ReserveUsd:   20000,
			AmplifiedTvl: 100,
		}

		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, SortByAmplifiedTvl, p.Address, p.AmplifiedTvl, false)
		assert.Nil(t, err)
		err = repo.AddToSortedSet(context.TODO(), "tokenaddress1", "tokenaddress2", true, true, SortByTVL, p.Address, p.ReserveUsd, true)
		assert.Nil(t, err)

		// assert data before delete
		expectedTvlScore := map[string]float64{"pooladdress2": 20000}
		expectedAmplifiedTvlScore := map[string]float64{"pooladdress2": 100}

		directPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		directPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, "tokenaddress2-tokenaddress1"))
		assert.Equal(t, directPoolsTvl, expectedTvlScore)

		globalTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s", SortByTVL))
		assert.Equal(t, globalTvl, expectedTvlScore)

		whitelistPoolsAmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		assert.Equal(t, whitelistPoolsAmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistPoolsTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, KeyWhitelist))
		assert.Equal(t, whitelistPoolsTvl, expectedTvlScore)

		whitelistToken1AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken1Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByTVL, KeyWhitelist, "tokenaddress1"))
		assert.Equal(t, whitelistToken1Tvl, expectedTvlScore)

		whitelistToken2AmplifiedTvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByAmplifiedTvl, KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2AmplifiedTvl, expectedAmplifiedTvlScore)
		whitelistToken2Tvl, _ := redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s:%s", SortByTVL, KeyWhitelist, "tokenaddress2"))
		assert.Equal(t, whitelistToken2Tvl, expectedTvlScore)

		err = repo.RemoveAddressFromIndex(context.TODO(), SortByTVL, []string{"pooladdress2"})
		assert.Nil(t, err)
		err = repo.RemoveAddressFromIndex(context.TODO(), SortByAmplifiedTvl, []string{"pooladdress2"})
		assert.Nil(t, err)

		// asset data after delete
		globalTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s", SortByTVL))
		assert.Nil(t, globalTvl)

		whitelistPoolsAmplifiedTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByAmplifiedTvl, KeyWhitelist))
		assert.Nil(t, whitelistPoolsAmplifiedTvl)
		whitelistPoolsTvl, _ = redisServer.SortedSet(fmt.Sprintf("ethereum:%s:%s", SortByTVL, KeyWhitelist))
		assert.Nil(t, whitelistPoolsTvl)

	})
}

func TestRedisRepository_AddToWhitelistSortedSet(t *testing.T) {
	type testInput struct {
		name      string
		oldScores []routerEntity.PoolScore
		scores    []routerEntity.PoolScore
		err       error
	}
	tests := []testInput{
		{
			name: "it should save correct data when the old Score set doesn't exist",
			scores: []routerEntity.PoolScore{
				{
					LiquidityScore: 92129,
					Pool:           "0x764510ab1d39cf300e7abe8f5b8977d18f290628",
					Level:          2,
				},
				{
					LiquidityScore: 4645,
					Pool:           "0x99c7550be72f05ec31c446cd536f8a29c89fdb77",
					Level:          2,
				},
				{
					LiquidityScore: 3392940,
					Pool:           "bebop_0x6b175474e89094c44da98b954eedeac495271d0f_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Level:          6,
				},
			},
		},
		{
			name: "it should save correct data and delete the old score set",
			oldScores: []routerEntity.PoolScore{
				{
					LiquidityScore: 270110,
					Pool:           "0xc7cbff2a23d0926604f9352f65596e65729b8a17",
					Level:          4,
				},
				{
					LiquidityScore: 107094,
					Pool:           "hashflow_v3_mm29_5_0x6b175474e89094c44da98b954eedeac495271d0f_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Level:          5,
				},
				{
					LiquidityScore: 33868,
					Pool:           "bebop_0x6b175474e89094c44da98b954eedeac495271d0f_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Level:          6,
				},
				{
					LiquidityScore: 464598,
					Pool:           "0x99c7550be72f05ec31c446cd536f8a29c89fdb77",
					Level:          3,
				},
			},
			scores: []routerEntity.PoolScore{
				{
					LiquidityScore: 92129,
					Pool:           "0x764510ab1d39cf300e7abe8f5b8977d18f290628",
					Level:          2,
				},
				{
					LiquidityScore: 4645,
					Pool:           "0x99c7550be72f05ec31c446cd536f8a29c89fdb77",
					Level:          2,
				},
				{
					LiquidityScore: 3392940,
					Pool:           "bebop_0x6b175474e89094c44da98b954eedeac495271d0f_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Level:          6,
				},
			},
		},
		{
			name: "it should save correct data and delete the whole old score set",
			oldScores: []routerEntity.PoolScore{
				{
					LiquidityScore: 270110,
					Pool:           "0xc7cbff2a23d0926604f9352f65596e65729b8a17",
					Level:          4,
				},
				{
					LiquidityScore: 107094,
					Pool:           "hashflow_v3_mm29_5_0x6b175474e89094c44da98b954eedeac495271d0f_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Level:          5,
				},
			},
			scores: []routerEntity.PoolScore{
				{
					LiquidityScore: 92129,
					Pool:           "0x764510ab1d39cf300e7abe8f5b8977d18f290628",
					Level:          2,
				},
				{
					LiquidityScore: 4645,
					Pool:           "0x99c7550be72f05ec31c446cd536f8a29c89fdb77",
					Level:          2,
				},
				{
					LiquidityScore: 3392940,
					Pool:           "bebop_0x6b175474e89094c44da98b954eedeac495271d0f_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Level:          6,
				},
			},
		},
		{
			name: "it should retain old sorted set when new set is empty and return error",
			oldScores: []routerEntity.PoolScore{
				{
					LiquidityScore: 270110,
					Pool:           "0xc7cbff2a23d0926604f9352f65596e65729b8a17",
					Level:          4,
				},
				{
					LiquidityScore: 107094,
					Pool:           "hashflow_v3_mm29_5_0x6b175474e89094c44da98b954eedeac495271d0f_0xdac17f958d2ee523a2206206994597c13d831ec7",
					Level:          5,
				},
			},
			err: errors.New("can not add empty list to whitelist sorted set"),
		},
	}

	key := "ethereum:liquidityScoreTvl:whitelist"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

			// prepare data
			for _, score := range test.oldScores {
				encoded := score.EncodeScore(true)
				redisServer.ZAdd(
					key,
					encoded,
					score.Pool,
				)
			}
			if len(test.oldScores) != 0 {
				// verify scores after prepare data
				sortedSet, err := redisServer.SortedSet(key)
				assert.Nil(t, err)
				assert.Equal(t, len(sortedSet), len(test.oldScores))
				for _, score := range test.oldScores {
					encoded := score.EncodeScore(true)
					assert.Equal(t, sortedSet[score.Pool], encoded)
				}
			}

			repo := NewRedisRepository(db.Client, wrap(RedisRepositoryConfig{
				Prefix: "ethereum",
			}))

			err = repo.AddToWhitelistSortedSet(context.TODO(), test.scores, SortByLiquidityScoreTvl, 500)
			if test.err == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, test.err.Error(), err.Error())
			}

			// verify scores after inserting
			if len(test.scores) != 0 {
				sortedSet, _ := redisServer.SortedSet(key)
				assert.Equal(t, len(sortedSet), len(test.scores))
				for _, score := range test.scores {
					encoded := score.EncodeScore(true)
					assert.Equal(t, sortedSet[score.Pool], encoded)
				}
			}

		})
	}
}

func TestRedisRepository_FindBestPoolIDsByScore(t *testing.T) {
	options := valueobject.GetBestPoolsOptions{
		DirectPoolsCount:    100,
		WhitelistPoolsCount: 500,
		TokenInPoolsCount:   200,
		TokenOutPoolCount:   200,

		AmplifiedTvlDirectPoolsCount:    50,
		AmplifiedTvlWhitelistPoolsCount: 200,
		AmplifiedTvlTokenInPoolsCount:   100,
		AmplifiedTvlTokenOutPoolCount:   100,
	}
	type testInput struct {
		name           string
		prepare        func(redisClient redisClient.UniversalClient) *redisRepository
		tokenIn        string
		tokenOut       string
		amountIn       float64
		sortBy         string
		expectedResult []string
	}
	tests := []testInput{
		{
			name: "it should return correct data with related score when tokenIn is non-whitelist, token out is whitelist",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				redisPools := []*entity.Pool{
					{
						Address:    "pool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:    "pool2",
						ReserveUsd: 1000,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistA",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:      "pool3",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistA",
							},
						},
					},
					{
						Address:    "pool4",
						ReserveUsd: 1000,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistC",
							},
							{
								Address: "whitelistD",
							},
						},
					},
				}
				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "pool1", redisPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistA",
					false, true, SortByTVLNative, "pool3", redisPools[2].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistA",
					false, true, SortByAmplifiedTVLNative, "pool3", redisPools[2].AmplifiedTvl, false)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "pool2",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "pool4",
						LiquidityScore: 4535,
						Level:          2,
					},
				}, SortByLiquidityScoreTvl, options.WhitelistPoolsCount)

				return repo
			},
			tokenIn:        "nonWhitelistA",
			tokenOut:       "whitelistB",
			amountIn:       10000,
			sortBy:         SortByLiquidityScoreTvl,
			expectedResult: []string{"pool3", "pool2"},
		},
		{
			name: "it should return correct data with related score both tokens is whitelist",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				redisPools := []*entity.Pool{
					{
						Address:    "pool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:      "pool2",
						ReserveUsd:   1000,
						AmplifiedTvl: 1000,
						SwapFee:      0.3,
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistA",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:      "pool3",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistD",
							},
						},
					},
					{
						Address:    "pool4",
						ReserveUsd: 1000,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistC",
							},
							{
								Address: "whitelistD",
							},
						},
					},
					{
						Address:    "pool5",
						ReserveUsd: 1000,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistD",
							},
							{
								Address: "whitelistA",
							},
						},
					},
				}
				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "pool1", redisPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistD",
					false, true, SortByTVLNative, "pool3", redisPools[2].ReserveUsd, false)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "pool2",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "pool4",
						LiquidityScore: 4535,
						Level:          2,
					},
					{
						Pool:           "pool5",
						LiquidityScore: 14056,
						Level:          3,
					},
				}, SortByLiquidityScoreTvl, options.WhitelistPoolsCount)

				return repo
			},
			tokenIn:        "whitelistA",
			tokenOut:       "whitelistD",
			amountIn:       10,
			sortBy:         SortByLiquidityScoreTvl,
			expectedResult: []string{"pool2", "pool5", "pool4"},
		},
		{
			name: "it should return correct data with related score both tokens is non-whitelist",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				redisPools := []*entity.Pool{
					{
						Address:    "pool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:      "pool2",
						ReserveUsd:   1000,
						AmplifiedTvl: 1000,
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistA",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:      "pool3",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistC",
							},
						},
					},
					{
						Address:    "pool4",
						ReserveUsd: 1000,
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistC",
							},
							{
								Address: "whitelistC",
							},
						},
					},
					{
						Address:    "pool5",
						ReserveUsd: 1000,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistC",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:    "pool6",
						ReserveUsd: 1000,
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistC",
							},
							{
								Address: "whitelistA",
							},
						},
					},
				}
				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "pool1", redisPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistC",
					false, true, SortByTVLNative, "pool3", redisPools[2].ReserveUsd, false)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistC", "whitelistC",
					false, true, SortByTVLNative, "pool4", redisPools[3].ReserveUsd, false)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistC", "whitelistA",
					false, true, SortByTVLNative, "pool6", redisPools[5].ReserveUsd, false)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "pool2",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "pool5",
						LiquidityScore: 14056,
						Level:          3,
					},
				}, SortByLiquidityScoreTvl, options.WhitelistPoolsCount)

				return repo
			},
			tokenIn:        "nonWhitelistA",
			tokenOut:       "nonWhitelistC",
			amountIn:       50000,
			sortBy:         SortByLiquidityScoreTvl,
			expectedResult: []string{"pool3", "pool4", "pool6", "pool2"},
		},
		{
			name: "it should return correct data with related score when tokenIn is whitelist, token out is non-whitelist",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				redisPools := []*entity.Pool{
					{
						Address:    "pool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:      "pool2",
						ReserveUsd:   1000,
						AmplifiedTvl: 1000,
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistA",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:      "pool3",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistC",
							},
						},
					},
					{
						Address:    "pool4",
						ReserveUsd: 1000,
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistC",
							},
							{
								Address: "whitelistC",
							},
						},
					},
					{
						Address:    "pool5",
						ReserveUsd: 1000,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "whitelistC",
							},
							{
								Address: "whitelistB",
							},
						},
					},
					{
						Address:    "pool6",
						ReserveUsd: 1000,
						Reserves:   []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistC",
							},
							{
								Address: "whitelistA",
							},
						},
					},
				}
				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistB",
					false, true, SortByTVLNative, "pool1", redisPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistC",
					false, true, SortByTVLNative, "pool3", redisPools[2].ReserveUsd, false)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistC", "whitelistC",
					false, true, SortByTVLNative, "pool4", redisPools[3].ReserveUsd, false)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistC", "whitelistA",
					false, true, SortByTVLNative, "pool6", redisPools[5].ReserveUsd, false)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "pool2",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "pool5",
						LiquidityScore: 14056,
						Level:          3,
					},
				}, SortByLiquidityScoreTvl, options.WhitelistPoolsCount)

				return repo
			},
			tokenIn:        "whitelistC",
			tokenOut:       "nonWhitelistC",
			amountIn:       2500,
			sortBy:         SortByLiquidityScoreTvl,
			expectedResult: []string{"pool2", "pool4", "pool5", "pool6"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

			repo := test.prepare(db.Client)

			pools, err := repo.findBestPoolIDsByScore(
				context.Background(),
				test.tokenIn,
				test.tokenOut,
				test.amountIn,
				options,
				test.sortBy,
			)

			assert.ElementsMatch(t, test.expectedResult, pools)
			assert.Nil(t, err)
		})
	}

}

func TestRedisRepository_FindGlobalBestPoolsByScores(t *testing.T) {
	type testInput struct {
		name           string
		prepare        func(redisClient redisClient.UniversalClient) *redisRepository
		counter        int64
		sortByKey      string
		expectedResult []string
		err            error
	}
	tests := []testInput{
		{
			name: "it should return both whitelist set with liquidity score and global best pool set",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				globalPools := []*entity.Pool{
					{
						Address:    "globalPool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:      "globalPool2",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistA",
							},
						},
					},
				}

				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "globalPool1", globalPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistA",
					false, true, SortByTVLNative, "globalPool2", globalPools[1].ReserveUsd, true)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "wlPool1",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "wlPool2",
						LiquidityScore: 4535,
						Level:          2,
					},
					{
						Pool:           "wlPool3",
						LiquidityScore: 20483745,
						Level:          6,
					},
				}, SortByLiquidityScoreTvl, 10)

				return repo
			},
			counter:        10,
			sortByKey:      SortByLiquidityScoreTvl,
			expectedResult: []string{"wlPool3", "wlPool1", "wlPool2", "globalPool2", "globalPool1"},
		},
		{
			name: "it should return only whitelist set because len of whitelist set reach max count",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				globalPools := []*entity.Pool{
					{
						Address:    "globalPool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:      "globalPool2",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistA",
							},
						},
					},
				}

				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "globalPool1", globalPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistA",
					false, true, SortByTVLNative, "globalPool2", globalPools[1].ReserveUsd, true)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "wlPool1",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "wlPool2",
						LiquidityScore: 4535,
						Level:          2,
					},
					{
						Pool:           "wlPool3",
						LiquidityScore: 20483745,
						Level:          6,
					},
				}, SortByLiquidityScoreTvl, 10)

				return repo
			},
			counter:        3,
			sortByKey:      SortByLiquidityScoreTvl,
			expectedResult: []string{"wlPool3", "wlPool1", "wlPool2"},
		},
		{
			name: "it should return only whitelist set and a part of global set len of 2 set exceeds count",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				globalPools := []*entity.Pool{
					{
						Address:    "globalPool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:      "globalPool2",
						ReserveUsd:   20000,
						AmplifiedTvl: 20000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistA",
							},
						},
					},
					{
						Address:      "globalPool3",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistB",
							},
							{
								Address: "whitelistD",
							},
						},
					},
				}

				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "globalPool1", globalPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistA",
					false, true, SortByTVLNative, "globalPool2", globalPools[1].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistB", "whitelistD",
					false, true, SortByTVLNative, "globalPool3", globalPools[2].ReserveUsd, true)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "wlPool1",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "wlPool2",
						LiquidityScore: 4535,
						Level:          2,
					},
					{
						Pool:           "wlPool3",
						LiquidityScore: 20483745,
						Level:          6,
					},
				}, SortByLiquidityScoreTvl, 10)

				return repo
			},
			counter:        5,
			sortByKey:      SortByLiquidityScoreTvl,
			expectedResult: []string{"wlPool3", "wlPool1", "wlPool2", "globalPool2", "globalPool3"},
		},
		{
			name: "it should return a part of whitelist set because len of whitelist set reach max count",
			prepare: func(client redisClient.UniversalClient) *redisRepository {
				globalPools := []*entity.Pool{
					{
						Address:    "globalPool1",
						ReserveUsd: 100,
						SwapFee:    0.3,
						Type:       "uniswap",
						Reserves:   []string{"reserve1", "reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "nonWhitelistB",
							},
						},
					},
					{
						Address:      "globalPool2",
						ReserveUsd:   10000,
						AmplifiedTvl: 10000,
						Type:         "uni",
						Reserves:     []string{"reserve1, reserve2"},
						Tokens: []*entity.PoolToken{
							{
								Address: "nonWhitelistA",
							},
							{
								Address: "whitelistA",
							},
						},
					},
				}

				repo := NewRedisRepository(client, wrap(RedisRepositoryConfig{
					Prefix: "ethereum",
				}))
				ctx := context.TODO()

				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "nonWhitelistB",
					false, false, SortByTVLNative, "globalPool1", globalPools[0].ReserveUsd, true)
				_ = repo.AddToSortedSet(ctx, "nonWhitelistA", "whitelistA",
					false, true, SortByTVLNative, "globalPool2", globalPools[1].ReserveUsd, true)

				// Add to pool score set
				repo.AddToWhitelistSortedSet(ctx, []routerEntity.PoolScore{
					{
						Pool:           "wlPool1",
						LiquidityScore: 107143,
						Level:          5,
					},
					{
						Pool:           "wlPool2",
						LiquidityScore: 4535,
						Level:          2,
					},
					{
						Pool:           "wlPool3",
						LiquidityScore: 20483745,
						Level:          6,
					},
					{
						Pool:           "wlPool4",
						LiquidityScore: 20483745,
						Level:          7,
					},
				}, SortByLiquidityScoreTvl, 10)

				return repo
			},
			counter:        3,
			sortByKey:      SortByLiquidityScoreTvl,
			expectedResult: []string{"wlPool4", "wlPool3", "wlPool1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

			repo := test.prepare(db.Client)

			pools, err := repo.FindGlobalBestPoolsByScores(
				context.Background(),
				test.counter,
				test.sortByKey)

			assert.Equal(t, test.expectedResult, pools)
			if test.err == nil {
				assert.Nil(t, err)
			}
		})
	}

}
