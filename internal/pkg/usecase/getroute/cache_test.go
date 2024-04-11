package getroute

import (
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newDefaultRouteCacheKey(amountIn float64, cacheMode valueobject.RouteCacheMode) *valueobject.RouteCacheKey {
	return &valueobject.RouteCacheKey{
		CacheMode:              string(cacheMode),
		AmountIn:               strconv.FormatFloat(amountIn, 'f', -1, 64),
		TokenIn:                "",
		TokenOut:               "",
		SaveGas:                false,
		GasInclude:             false,
		Dexes:                  nil,
		IsPathGeneratorEnabled: false,
		IsHillClimbingEnabled:  false,
		ExcludedPools:          nil,
	}
}

func TestCache_GenKey_ExactCachedPoint(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		param    *types.AggregateParams
		cacheKey *valueobject.RouteCacheKey
		duration time.Duration
		err      error
	}{
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn: big.NewInt(1e18),
			},
			cacheKey: newDefaultRouteCacheKey(float64(1), valueobject.RouteCacheModePoint),
			duration: 30 * time.Second,
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 17,
				},
				AmountIn: big.NewInt(1e18),
			},
			cacheKey: newDefaultRouteCacheKey(float64(10), valueobject.RouteCacheModePoint),
			duration: 10 * time.Second,
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 16,
				},
				AmountIn: big.NewInt(2.2e17),
			},
			cacheKey: newDefaultRouteCacheKey(float64(22), valueobject.RouteCacheModePoint),
			duration: 30 * time.Second,
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 17,
				},
				AmountIn: big.NewInt(2.2e18),
			},
			cacheKey: newDefaultRouteCacheKey(float64(22), valueobject.RouteCacheModePoint),
			duration: 30 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config := valueobject.CacheConfig{
				TTLByAmount: []valueobject.CachePoint{
					{Amount: 1, TTL: 30 * time.Second},
					{Amount: 2, TTL: 10 * time.Second},
					{Amount: 5, TTL: 10 * time.Second},
					{Amount: 10, TTL: 10 * time.Second},
					{Amount: 15, TTL: 10 * time.Second},
					{Amount: 20, TTL: 10 * time.Second},
					{Amount: 22, TTL: 30 * time.Second},
					{Amount: 25, TTL: 10 * time.Second},
					{Amount: 30, TTL: 10 * time.Second},
				},
			}

			cache := &cache{config: config}
			key, duration, err := cache.genKey(tc.param)

			assert.Equal(t, *tc.cacheKey, *key)
			assert.Equal(t, tc.duration, duration)
			assert.ErrorIs(t, tc.err, err)
		})
	}
}

func TestCache_GenKey_CachePointUSD(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		param    *types.AggregateParams
		cacheKey *valueobject.RouteCacheKey
		duration time.Duration
		err      error
	}{
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 1,
				},
				AmountIn:        big.NewInt(100),
				TokenInPriceUSD: 1,
			},
			cacheKey: newDefaultRouteCacheKey(float64(10), valueobject.RouteCacheModeRange),
			duration: 10 * time.Second,
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 1,
				},
				AmountIn:        big.NewInt(20010),
				TokenInPriceUSD: 1,
			},
			cacheKey: newDefaultRouteCacheKey(float64(2001), valueobject.RouteCacheModeRange),
			duration: 14 * time.Second,
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 1,
				},
				AmountIn:        big.NewInt(10500),
				TokenInPriceUSD: 1,
			},
			cacheKey: newDefaultRouteCacheKey(float64(1050), valueobject.RouteCacheModeRange),
			duration: 13 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config := valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName: ShrinkFuncNameRound,
			}

			cache := NewCache(nil, nil, nil, config)
			key, duration, err := cache.genKey(tc.param)

			assert.Equal(t, *tc.cacheKey, *key)
			assert.Equal(t, tc.duration, duration)
			assert.ErrorIs(t, tc.err, err)
		})
	}
}
