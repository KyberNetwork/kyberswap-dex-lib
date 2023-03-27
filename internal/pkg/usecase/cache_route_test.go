package usecase

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func TestCacheRoute_Get(t *testing.T) {
	t.Run("it should return correct error when routeCacheRepo.Get return error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		theErr := errors.New("some error")
		ctx := context.Background()
		key := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "5000000000000000000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Get(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1").
			Return(nil, 15*time.Second, theErr)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		cachedRoute, err := routeCache.Get(ctx, key)

		assert.Nil(t, cachedRoute)
		assert.ErrorIs(t, err, theErr)
	})

	t.Run("it should return correct error when data length is zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		key := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "5000000000000000000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Get(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1").
			Return(nil, 15*time.Second, nil)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		cachedRoute, err := routeCache.Get(ctx, key)

		assert.Nil(t, cachedRoute)
		assert.ErrorIs(t, err, ErrRouteCacheNotFound)
	})

	t.Run("it should return correct error when cache expired", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		key := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "5000000000000000000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Get(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1").
			Return([]byte("route_cache"), time.Duration(0), nil)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		cachedRoute, err := routeCache.Get(ctx, key)

		assert.Nil(t, cachedRoute)
		assert.ErrorIs(t, err, ErrRouteCacheExpired)
	})

	t.Run("it should return correct error when unmarshal failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		key := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "5000000000000000000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Get(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1").
			Return([]byte("route_cache"), 15*time.Second, nil)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		cachedRoute, err := routeCache.Get(ctx, key)

		assert.Nil(t, cachedRoute)
		assert.ErrorIs(t, err, ErrRouteCacheUnmarshalFailed)
	})

	t.Run("it should return correct cachedRoute", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		key := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "5000000000000000000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Get(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:1").
			Return([]byte(`{"Input":{"Token":"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7","Amount":50000,"AmountUsd":0},"Paths":[{"Input":{"Token":"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7","Amount":50000,"AmountUsd":0},"Output":{"Token":"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664","Amount":50000,"AmountUsd":1361.535157},"TotalGas":320000,"Pools":[],"Tokens":[{"address":"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7","symbol":"WAVAX","name":"Wrapped AVAX","decimals":18,"cgkId":"avalanche-2","type":"erc20","poolAddress":""},{"address":"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664","symbol":"USDC.e","name":"USD Coin","decimals":6,"cgkId":"usd-coin-avalanche-bridged-usdc-e","type":"erc20","poolAddress":""}],"PriceImpact":1000,"MidPrice":null}]}`), 15*time.Second, nil)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		cachedRoute, err := routeCache.Get(ctx, key)

		expectedCachedRoute := &core.CachedRoute{
			Input: pool.TokenAmount{
				Token:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
				Amount:    new(big.Int).SetInt64(50000),
				AmountUsd: 0,
			},
			Paths: []core.CachedPath{
				{
					Input: pool.TokenAmount{
						Token:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
						Amount:    new(big.Int).SetInt64(50000),
						AmountUsd: 0,
					},
					Output: pool.TokenAmount{
						Token:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
						Amount:    new(big.Int).SetInt64(50000),
						AmountUsd: 1361.535157,
					},
					TotalGas: 320000,
					Tokens: []entity.Token{
						{
							Address:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
							Symbol:      "WAVAX",
							Name:        "Wrapped AVAX",
							Decimals:    18,
							CgkID:       "avalanche-2",
							Type:        "erc20",
							PoolAddress: "",
						},
						{
							Address:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
							Symbol:      "USDC.e",
							Name:        "USD Coin",
							Decimals:    6,
							CgkID:       "usd-coin-avalanche-bridged-usdc-e",
							Type:        "erc20",
							PoolAddress: "",
						},
					},
					PriceImpact: new(big.Int).SetInt64(1000),
				},
			},
		}
		assert.Equal(t, expectedCachedRoute, cachedRoute)
		assert.Nil(t, err)
	})
}

func TestCacheRoute_Set(t *testing.T) {
	t.Run("it should return error when set cache failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		theErr := errors.New("some error")
		cachedRoute := core.CachedRoute{
			Input: pool.TokenAmount{
				Token:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
				Amount:    new(big.Int).SetInt64(50000),
				AmountUsd: 0,
			},
			Paths: []core.CachedPath{
				{
					Input: pool.TokenAmount{
						Token:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
						Amount:    new(big.Int).SetInt64(50000),
						AmountUsd: 0,
					},
					Output: pool.TokenAmount{
						Token:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
						Amount:    new(big.Int).SetInt64(50000),
						AmountUsd: 1361.535157,
					},
					TotalGas: 320000,
					PoolIDs:  []string{},
					Tokens: []entity.Token{
						{
							Address:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
							Symbol:      "WAVAX",
							Name:        "Wrapped AVAX",
							Decimals:    18,
							CgkID:       "avalanche-2",
							Type:        "erc20",
							PoolAddress: "",
						},
						{
							Address:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
							Symbol:      "USDC.e",
							Name:        "USD Coin",
							Decimals:    6,
							CgkID:       "usd-coin-avalanche-bridged-usdc-e",
							Type:        "erc20",
							PoolAddress: "",
						},
					},
					PriceImpact: new(big.Int).SetInt64(1000),
				},
			},
		}
		key := valueobject.RouteCacheKey{
			TokenIn:   "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:  "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:   true,
			CacheMode: valueobject.RouteCacheModePoint,
			AmountIn:  "5000000000000000000000",
			Dexes:     []string{"gmx", "uniswap"},
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Set(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:0", gomock.Any(), gomock.Any()).
			Return(theErr)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		err := routeCache.Set(ctx, key, cachedRoute, new(big.Int).SetInt64(1000000), 6, 1)

		assert.ErrorIs(t, err, theErr)
	})

	t.Run("it should set cache without error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		cachedRoute := core.CachedRoute{
			Input: pool.TokenAmount{
				Token:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
				Amount:    new(big.Int).SetInt64(50000),
				AmountUsd: 0,
			},
			Paths: []core.CachedPath{
				{
					Input: pool.TokenAmount{
						Token:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
						Amount:    new(big.Int).SetInt64(50000),
						AmountUsd: 0,
					},
					Output: pool.TokenAmount{
						Token:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
						Amount:    new(big.Int).SetInt64(50000),
						AmountUsd: 1361.535157,
					},
					TotalGas: 320000,
					PoolIDs:  []string{},
					Tokens: []entity.Token{
						{
							Address:     "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
							Symbol:      "WAVAX",
							Name:        "Wrapped AVAX",
							Decimals:    18,
							CgkID:       "avalanche-2",
							Type:        "erc20",
							PoolAddress: "",
						},
						{
							Address:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
							Symbol:      "USDC.e",
							Name:        "USD Coin",
							Decimals:    6,
							CgkID:       "usd-coin-avalanche-bridged-usdc-e",
							Type:        "erc20",
							PoolAddress: "",
						},
					},
					PriceImpact: new(big.Int).SetInt64(1000),
				},
			},
		}
		key := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "5000000000000000000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: false,
		}

		routeCacheRepo := usecase.NewMockIRouteCacheRepository(ctrl)
		routeCacheRepo.EXPECT().
			Set(ctx, "avalanche:0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be-0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7:1:token:5000000000000000000000:gmx-uniswap:0", gomock.Any(), gomock.Any()).
			Return(nil)

		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{KeyPrefix: "avalanche"},
			routeCacheRepo,
		)

		err := routeCache.Set(ctx, key, cachedRoute, new(big.Int).SetInt64(1000000), 6, 1)

		assert.Nil(t, err)
	})
}

func TestCacheRoute_GenKey(t *testing.T) {
	t.Run("it should gen correct key in point mode", func(t *testing.T) {
		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{
				CachePoints: []CachePointConfig{
					{
						Amount: 1,
						TTL:    2,
					},
				},
			},
			nil,
		)

		key := routeCache.GenKey(
			"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			new(big.Int).SetInt64(1000000),
			6,
			1,
			true,
			[]string{"gmx", "uniswap"},
			true,
		)

		expectedKey := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModePoint,
			AmountIn:   "1000000",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		assert.Equal(t, expectedKey, key)
	})

	t.Run("it should gen correct key in range mode", func(t *testing.T) {
		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{
				CachePoints: []CachePointConfig{
					{
						Amount: 1,
						TTL:    2,
					},
				},
				CacheRanges: []CacheRangeConfig{
					{
						FromUSD: 1,
						ToUSD:   2,
						TTL:     2,
					},
				},
			},
			nil,
		)

		key := routeCache.GenKey(
			"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			new(big.Int).SetInt64(1200000),
			6,
			1,
			true,
			[]string{"gmx", "uniswap"},
			true,
		)

		expectedKey := valueobject.RouteCacheKey{
			TokenIn:    "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be",
			TokenOut:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			SaveGas:    true,
			CacheMode:  valueobject.RouteCacheModeRange,
			AmountIn:   "1",
			Dexes:      []string{"gmx", "uniswap"},
			GasInclude: true,
		}

		assert.Equal(t, expectedKey, key)
	})
}

func TestCacheRoute_GetCacheTTL(t *testing.T) {
	t.Run("it should return correct ttl in point mode", func(t *testing.T) {
		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{
				CachePoints: []CachePointConfig{
					{
						Amount: 1,
						TTL:    2 * time.Second,
					},
				},
				CacheRanges: []CacheRangeConfig{
					{
						FromUSD: 1,
						ToUSD:   2,
						TTL:     3 * time.Second,
					},
				},
			},
			nil,
		)

		ttl := routeCache.GetCacheTTL(new(big.Float).SetInt64(1), 1)

		assert.Equal(t, 2*time.Second, ttl)
	})

	t.Run("it should return correct ttl in range mode", func(t *testing.T) {
		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{
				CachePoints: []CachePointConfig{
					{
						Amount: 1,
						TTL:    2 * time.Second,
					},
				},
				CacheRanges: []CacheRangeConfig{
					{
						FromUSD: 1,
						ToUSD:   2,
						TTL:     3 * time.Second,
					},
				},
			},
			nil,
		)

		ttl := routeCache.GetCacheTTL(new(big.Float).SetFloat64(1.2), 1)

		assert.Equal(t, 3*time.Second, ttl)
	})

	t.Run("it should return correct default ttl", func(t *testing.T) {
		routeCache := NewCacheRouteUseCase(
			CacheRouteConfig{
				CachePoints: []CachePointConfig{
					{
						Amount: 1,
						TTL:    2 * time.Second,
					},
				},
				CacheRanges: []CacheRangeConfig{
					{
						FromUSD: 1,
						ToUSD:   2,
						TTL:     3 * time.Second,
					},
				},
				DefaultCacheTTL: 4 * time.Second,
			},
			nil,
		)

		ttl := routeCache.GetCacheTTL(new(big.Float).SetFloat64(4), 4)

		assert.Equal(t, 4*time.Second, ttl)
	})
}
