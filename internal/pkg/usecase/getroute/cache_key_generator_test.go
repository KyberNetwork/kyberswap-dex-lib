package getroute

import (
	"context"
	"errors"
	"fmt"
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

func newDefaultRouteCacheKey(amountIn float64, cacheMode valueobject.RouteCacheMode, ttl time.Duration) []valueobject.RouteCacheKeyTTL {
	return []valueobject.RouteCacheKeyTTL{
		{
			Key: &valueobject.RouteCacheKey{
				CacheMode:     string(cacheMode),
				AmountIn:      strconv.FormatFloat(amountIn, 'f', -1, 64),
				TokenIn:       "",
				TokenOut:      "",
				SaveGas:       false,
				GasInclude:    false,
				Dexes:         nil,
				ExcludedPools: nil,
			},
			TTL: ttl,
		},
	}
}

func newMultiRouteCacheKeys(amountIns []float64, cacheMode valueobject.RouteCacheMode, ttl []time.Duration) []valueobject.RouteCacheKeyTTL {
	results := []valueobject.RouteCacheKeyTTL{}
	for i, a := range amountIns {
		results = append(results, valueobject.RouteCacheKeyTTL{
			Key: &valueobject.RouteCacheKey{
				CacheMode:     string(cacheMode),
				AmountIn:      strconv.FormatFloat(a, 'f', -1, 64),
				TokenIn:       "",
				TokenOut:      "",
				SaveGas:       false,
				GasInclude:    false,
				Dexes:         nil,
				ExcludedPools: nil,
			},
			TTL: ttl[i],
		})
	}

	return results
}

func TestKeyGenerator_ExactCachedPoint(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		param    *types.AggregateParams
		cacheKey []valueobject.RouteCacheKeyTTL
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
			cacheKey: newDefaultRouteCacheKey(float64(1), valueobject.RouteCacheModePoint, 30*time.Second),
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 17,
				},
				AmountIn: big.NewInt(1e18),
			},
			cacheKey: newDefaultRouteCacheKey(float64(10), valueobject.RouteCacheModePoint, 10*time.Second),
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 16,
				},
				AmountIn: big.NewInt(2.2e17),
			},
			cacheKey: newDefaultRouteCacheKey(float64(22), valueobject.RouteCacheModePoint, 30*time.Second),
		},
		{
			name: "Gen key should return correct key with duration for exact point input",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 17,
				},
				AmountIn: big.NewInt(2.2e18),
			},
			cacheKey: newDefaultRouteCacheKey(float64(22), valueobject.RouteCacheModePoint, 30*time.Second),
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

			keyGen := &routeKeyGenerator{config: config}
			keys, err := keyGen.genKey(context.TODO(), tc.param)

			assert.ElementsMatch(t, tc.cacheKey, keys.ToSlice())
			assert.ErrorIs(t, tc.err, err)
		})
	}
}

func TestKeyGenerator_CachePointUSD(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		param     *types.AggregateParams
		cacheKeys []valueobject.RouteCacheKeyTTL
		err       error
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
			cacheKeys: newDefaultRouteCacheKey(float64(10), valueobject.RouteCacheModeRangeByUSD, 10*time.Second),
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
			cacheKeys: newDefaultRouteCacheKey(float64(2001), valueobject.RouteCacheModeRangeByUSD, 14*time.Second),
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
			cacheKeys: newDefaultRouteCacheKey(float64(1050), valueobject.RouteCacheModeRangeByUSD, 13*time.Second),
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
				DefaultTTL:     5 * time.Second,
				ShrinkFuncName: string(ShrinkFuncNameRound),
			}

			keyGen := newCacheKeyGenerator(config)
			keys, err := keyGen.genKey(context.TODO(), tc.param)

			assert.ElementsMatch(t, tc.cacheKeys, keys.ToSlice())
			assert.ErrorIs(t, tc.err, err)
		})
	}
}

func Test_ApplyConfig(t *testing.T) {
	testCases := []struct {
		name      string
		oldConfig valueobject.CacheConfig
		newConfig valueobject.CacheConfig
		expected  valueobject.CacheConfig
		err       error
	}{
		{
			name: "Should apply correct config when remote config was changed",
			oldConfig: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:             string(ShrinkFuncNameRound),
				EnableNewCacheKeyGenerator: false,
			},
			newConfig: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
				},
				ShrinkFuncName: string(ShrinkFuncNamePow),
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{
						ShrinkFuncName:     string(ShrinkFuncNameDecimal),
						ShrinkFuncConstant: 100,
					},
				},
				EnableNewCacheKeyGenerator: true,
			},
			expected: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
				},
				ShrinkFuncName: string(ShrinkFuncNamePow),
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{
						ShrinkFuncName:     string(ShrinkFuncNameDecimal),
						ShrinkFuncConstant: 100,
					},
				},
				EnableNewCacheKeyGenerator: true,
			},
		},
		{
			name: "Should apply correct config when only ShrinkFuncName in ShrinkAmountInConfig change",
			oldConfig: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName: string(ShrinkFuncNameRound),
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{
						ShrinkFuncName: string(ShrinkFuncNameDecimal),
					},
				},
				EnableNewCacheKeyGenerator: false,
			},
			newConfig: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName: string(ShrinkFuncNameRound),
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{
						ShrinkFuncName:     string(ShrinkFuncNameLogarithm),
						ShrinkFuncConstant: 1.5,
					},
					{
						ShrinkFuncName:     string(ShrinkFuncNameLogarithm),
						ShrinkFuncConstant: 2,
					},
				},
				EnableNewCacheKeyGenerator: false,
			},
			expected: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName: string(ShrinkFuncNameRound),
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{
						ShrinkFuncName:     string(ShrinkFuncNameLogarithm),
						ShrinkFuncConstant: 1.5,
					},
					{
						ShrinkFuncName:     string(ShrinkFuncNameLogarithm),
						ShrinkFuncConstant: 2,
					},
				},
				EnableNewCacheKeyGenerator: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			keyGenerator := newCacheKeyGenerator(tc.oldConfig)
			newConfig := Config{
				Cache: tc.newConfig,
			}
			keyGenerator.applyConfig(newConfig)
			assert.Equal(t, keyGenerator.config.ShrinkFuncName, tc.expected.ShrinkFuncName)
			assert.Equal(t, keyGenerator.config.ShrinkDecimalBase, tc.expected.ShrinkDecimalBase)
			assert.ElementsMatch(t, keyGenerator.config.ShrinkAmountInConfigs, tc.expected.ShrinkAmountInConfigs)
			assert.Equal(t, keyGenerator.config.EnableNewCacheKeyGenerator, tc.expected.EnableNewCacheKeyGenerator)
		})
	}
}

func TestKeyGenerator_GenKeyV1(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		param     *types.AggregateParams
		cacheKeys []valueobject.RouteCacheKeyTTL
		config    valueobject.CacheConfig
		err       error
	}{
		{
			name: "Gen key v1 should return correct key by cache point TTL",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn: bigIntFromString("100000000000100000000"),
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				ShrinkDecimalBase: 10,
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 10 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(100.0000000001), valueobject.RouteCacheModePoint, 10*time.Second),
		},
		{
			name: "Gen key v1 should not return key by cache point TTL due to exceed threshold",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("100000000900000000000"),
				TokenInPriceUSD: 1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(100), valueobject.RouteCacheModeRangeByUSD, 18*time.Second),
		},
		{
			name: "Gen key v1 return no key because token in has no price",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("103000000000000000000"),
				TokenInPriceUSD: -1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: []valueobject.RouteCacheKeyTTL{},
			err:       ErrNoTokenInPrice,
		},
		{
			name: "Gen key v1 should not return key by cache point TTL due to exceed threshold, it return cache key by amount usd, round 3.36 to 3",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("3360000000000000000"),
				TokenInPriceUSD: 1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(3), valueobject.RouteCacheModeRangeByUSD, 18*time.Second),
		},
		{
			name: "Gen key v1 should not return key by cache point TTL due to exceed threshold, it return cache key by amount usd, round decimal 35.9 to 40",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("35900000000000000000"),
				TokenInPriceUSD: 1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(40), valueobject.RouteCacheModeRangeByUSD, 18*time.Second),
		},
		{
			name: "Gen key v1 should not return key by cache point TTL due to exceed threshold, it return cache key by amount usd, round decimal 136 to 100",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("136000000000000000000"),
				TokenInPriceUSD: 1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(100), valueobject.RouteCacheModeRangeByUSD, 18*time.Second),
		},
		{
			name: "Gen key v1 should not return key by cache point TTL due to exceed threshold, round 1350.6 to 1000",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("1030600000000000000000"),
				TokenInPriceUSD: 1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(1000), valueobject.RouteCacheModeRangeByUSD, 12*time.Second),
		},
		{
			name: "Gen key v1 should return cache key with amount in by usd (1,59) is above threshold (1)",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("1060000000000000000"),
				TokenInPriceUSD: 1.5,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:          string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:       10,
				ShrinkAmountInThreshold: 100000,
				MinAmountInUSD:          1,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(2), valueobject.RouteCacheModeRangeByUSD, 18*time.Second),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			keyGen := newCacheKeyGenerator(tc.config)
			keys, err := keyGen.genKey(context.TODO(), tc.param)

			if keys.IsEmpty() {
				assert.Empty(t, tc.cacheKeys)
			} else {
				assert.ElementsMatch(t, tc.cacheKeys, keys.ToSlice())
			}
			assert.ErrorIs(t, tc.err, err)
		})
	}
}

func TestKeyGenerator_GenKeyV2(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		param     *types.AggregateParams
		cacheKeys []valueobject.RouteCacheKeyTTL
		config    valueobject.CacheConfig
		err       error
	}{
		{
			name: "Gen key v2 should return cache key by amount usd, round 1350.6 to 1000",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromString("1030600000000000000000"),
				TokenInPriceUSD: 1,
			},
			config: valueobject.CacheConfig{
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
					{Amount: 100, TTL: 10 * time.Second},
				},
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				ShrinkFuncName:             string(ShrinkFuncNameDecimal),
				ShrinkDecimalBase:          10,
				EnableNewCacheKeyGenerator: true,
				ShrinkAmountInThreshold:    100000,
			},
			cacheKeys: newDefaultRouteCacheKey(float64(1000), valueobject.RouteCacheModeRangeByUSD, 12*time.Second),
		},
		{
			name: "Gen key v2 should return cache key by amount, when token in has no price, apply logarithm shrinking function with base 1.1 1.3 1.5",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn: bigIntFromScientificNotation("2e21"),
			},
			config: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				TTLByAmountRange: []valueobject.AmountInCacheRange{
					{AmountLowerBound: bigIntFromScientificNotation("0"), TTL: 60 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("2e17"), TTL: 40 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("1e18"), TTL: 40 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("1e24"), TTL: 10 * time.Second},
				},
				ShrinkFuncName:             string(ShrinkDecimalBase),
				ShrinkDecimalBase:          100,
				EnableNewCacheKeyGenerator: true,
				ShrinkAmountInThreshold:    100000,
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.1},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.3},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.5},
				},
			},
			cacheKeys: newMultiRouteCacheKeys([]float64{2048, 2015, 2217}, valueobject.RouteCacheModeRangeByAmount, []time.Duration{40 * time.Second, 40 * time.Second, 40 * time.Second}),
		},
		{
			name: "Gen key v2 should return cached keys which are below threshold",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromScientificNotation("2e21"),
				TokenInPriceUSD: 0,
			},
			config: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				TTLByAmountRange: []valueobject.AmountInCacheRange{
					{AmountLowerBound: bigIntFromScientificNotation("0"), TTL: 60 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("2e17"), TTL: 40 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("1e18"), TTL: 40 * time.Second},
				},
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.1},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.3},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.5},
				},
				ShrinkAmountInThreshold:    30,
				EnableNewCacheKeyGenerator: true,
			},
			cacheKeys: newMultiRouteCacheKeys([]float64{2015}, valueobject.RouteCacheModeRangeByAmount, []time.Duration{40 * time.Second}),
			err:       nil,
		},
		{
			name: "Gen key v2 should return errors when all shrunk values are above threshold",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromScientificNotation("2e21"),
				TokenInPriceUSD: 0,
			},
			config: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				TTLByAmountRange: []valueobject.AmountInCacheRange{
					{AmountLowerBound: bigIntFromScientificNotation("0"), TTL: 60 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("2e17"), TTL: 40 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("1e18"), TTL: 40 * time.Second},
				},
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.3},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.5},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.8},
				},
				ShrinkAmountInThreshold:    14,
				EnableNewCacheKeyGenerator: true,
			},
			cacheKeys: []valueobject.RouteCacheKeyTTL{},
			err:       errors.New("different between shunk value and amount in without decimal is above threshold"),
		},
		{
			name: "Gen key v2 should return result token in usd below threshold",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromScientificNotation("2e19"),
				TokenInPriceUSD: 5e-324,
			},
			config: valueobject.CacheConfig{
				TTLByAmountUSDRange: []valueobject.CacheRange{
					{AmountUSDLowerBound: 0, TTL: 18 * time.Second},
					{AmountUSDLowerBound: 101, TTL: 20 * time.Second},
					{AmountUSDLowerBound: 500, TTL: 12 * time.Second},
					{AmountUSDLowerBound: 1001, TTL: 13 * time.Second},
					{AmountUSDLowerBound: 2000, TTL: 14 * time.Second},
					{AmountUSDLowerBound: 5000, TTL: 15 * time.Second},
				},
				TTLByAmountRange: []valueobject.AmountInCacheRange{
					{AmountLowerBound: bigIntFromScientificNotation("0"), TTL: 60 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("2e17"), TTL: 40 * time.Second},
					{AmountLowerBound: bigIntFromScientificNotation("1e18"), TTL: 40 * time.Second},
				},
				ShrinkAmountInConfigs: []valueobject.ShrinkFunctionConfig{
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.3},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.5},
					{ShrinkFuncName: string(ShrinkFuncNameLogarithm), ShrinkFuncConstant: 1.8},
				},
				ShrinkAmountInThreshold:    100000,
				EnableNewCacheKeyGenerator: true,
				MinAmountInUSD:             0.9,
			},
			cacheKeys: newMultiRouteCacheKeys([]float64{17, 18, 19}, valueobject.RouteCacheModeRangeByAmount, []time.Duration{40 * time.Second, 40 * time.Second, 40 * time.Second}),
			err:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			keyGen := newCacheKeyGenerator(tc.config)
			keys, err := keyGen.genKey(context.TODO(), tc.param)

			if keys.IsEmpty() {
				assert.Empty(t, tc.cacheKeys)
			} else {
				assert.ElementsMatch(t, tc.cacheKeys, keys.ToSlice())
			}
			if tc.err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
		})
	}
}

func bigIntFromScientificNotation(s string) *big.Int {
	value, _, err := big.ParseFloat(s, 10, 0, big.ToNearestEven)
	if err != nil {
		fmt.Printf("bigFloatFromString err %e\n", err)
	}
	res, _ := value.Int(new(big.Int))
	return res
}

func bigIntFromString(s string) *big.Int {
	value, _ := new(big.Int).SetString(s, 10)
	return value
}

type Data struct {
	TokenIn        string   `json:"tokenIn,omitempty"`
	TokenOut       string   `json:"tokenOut,omitempty"`
	AmountIn       *big.Int `json:"amountIn,omitempty"`
	TokenInDecimal int      `json:"tokenInDecimal,omitempty"`
}
