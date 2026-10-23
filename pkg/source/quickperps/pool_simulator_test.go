package quickperps

import (
	"context"
	"math/big"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPool_CalcAmountOut(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut *poolPkg.TokenAmount
		expectedFee       *poolPkg.TokenAmount
		expectedGas       int64
		expectedErr       error
	}{
		{
			name: "it should return correct amount using getPriceV1",
			entityPool: entity.Pool{
				Address:  "0x99b31498b0a1dae01fc3433e3cb60f095340935c",
				Exchange: "quickperps",
				Type:     "quickperps",
				Reserves: []string{
					"657181327163967442895",
					"2924727278",
					"419037171254726109212969",
					"503045830168",
					"283581698250",
					"88943524272059284457598",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9",
						Swappable: true,
					},
					{
						Address:   "0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1",
						Swappable: true,
					},
					{
						Address:   "0xa2036f0538221a77a3937f1379699f44945018d0",
						Swappable: true,
					},
					{
						Address:   "0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035",
						Swappable: true,
					},
					{
						Address:   "0x1e4a5963abfd975d8c9021ce480b42188849d41d",
						Swappable: true,
					},
					{
						Address:   "0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4",
						Swappable: true,
					},
				},
				Extra: "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":10,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\",\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\",\"0xa2036f0538221a77a3937f1379699f44945018d0\",\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\",\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\",\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\"],\"poolAmounts\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":283581698250,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":657181327163967442895,\"0xa2036f0538221a77a3937f1379699f44945018d0\":419037171254726109212969,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":503045830168,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":88943524272059284457598,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":2924727278},\"bufferAmounts\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":264528000000,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":347000000000000000000,\"0xa2036f0538221a77a3937f1379699f44945018d0\":283368000000000000000000,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":793839000000,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":79386000000000000000000,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":1400000000},\"reservedAmounts\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":15102033483,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":165724739717004144459,\"0xa2036f0538221a77a3937f1379699f44945018d0\":156091358474313659327297,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":6476329143,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":0,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":287789247},\"tokenDecimals\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":6,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":18,\"0xa2036f0538221a77a3937f1379699f44945018d0\":18,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":6,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":18,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":8},\"stableTokens\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":true,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":false,\"0xa2036f0538221a77a3937f1379699f44945018d0\":false,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":true,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":true,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":false},\"usdqAmounts\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":283869850634637421002015,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":1237237742192508308542747,\"0xa2036f0538221a77a3937f1379699f44945018d0\":367103827909300721130288,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":502353085553584371191561,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":88934002839863855272999,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":1057631288731679426410181},\"maxUsdqAmounts\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":750000000000000000000000,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":2025000000000000000000000,\"0xa2036f0538221a77a3937f1379699f44945018d0\":760000000000000000000000,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":2250000000000000000000000,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":225000000000000000000000,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":1500000000000000000000000},\"tokenWeights\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":10000,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":27000,\"0xa2036f0538221a77a3937f1379699f44945018d0\":10000,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":30000,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":3000,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":20000},\"priceFeed\":{\"favorPrimaryPrice\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":15000000000000000000000000000,\"priceSampleSpace\":null,\"spreadThresholdBasisPoints\":30,\"expireTimeForPriceFeed\":86400,\"priceDecimals\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":18,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":18,\"0xa2036f0538221a77a3937f1379699f44945018d0\":18,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":18,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":18,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":18},\"spreadBasisPoints\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":0,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":0,\"0xa2036f0538221a77a3937f1379699f44945018d0\":10,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":0,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":0,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":0},\"adjustmentBasisPoints\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":0,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":0,\"0xa2036f0538221a77a3937f1379699f44945018d0\":0,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":0,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":0,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":0},\"strictStableTokens\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":false,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":false,\"0xa2036f0538221a77a3937f1379699f44945018d0\":false,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":false,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":false,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":false},\"isAdjustmentAdditive\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":false,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":false,\"0xa2036f0538221a77a3937f1379699f44945018d0\":false,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":false,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":false,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1700647502,\"maxDeviationBasisPoints\":100,\"minAuthorizations\":1,\"priceDuration\":300,\"maxPriceUpdateDelay\":3600,\"spreadBasisPointsIfChainError\":500,\"spreadBasisPointsIfInactive\":50,\"prices\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":1000141160000000000000000000000,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":2011605000000000000000000000000000,\"0xa2036f0538221a77a3937f1379699f44945018d0\":760488130000000000000000000000,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":999950010000000000000000000000,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":999882250000000000000000000000,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":36730716432920000000000000000000000},\"priceData\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xa2036f0538221a77a3937f1379699f44945018d0\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0}},\"maxCumulativeDeltaDiffs\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":0,\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":0,\"0xa2036f0538221a77a3937f1379699f44945018d0\":0,\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":0,\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":0,\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":0}},\"secondaryPriceFeedVersion\":2,\"priceFeeds\":{\"0x1e4a5963abfd975d8c9021ce480b42188849d41d\":{\"price\":1000465000000000200,\"timestamp\":1700590706},\"0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9\":{\"price\":2008830000000000000000,\"timestamp\":1700645943},\"0xa2036f0538221a77a3937f1379699f44945018d0\":{\"price\":759581640000000000,\"timestamp\":1700645998},\"0xa8ce8aee21bc2a48a5ef670afcc9274c7bbbc035\":{\"price\":999950000000000000,\"timestamp\":1700590619},\"0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4\":{\"price\":999478150000000000,\"timestamp\":1700590707},\"0xea034fb02eb1808c2cc3adbc15f447b93cbe08e1\":{\"price\":36696046300000000000000,\"timestamp\":1700647214}}},\"usdq\":{\"address\":\"0x48aC594dd00c4aAcF40f83337fc6dA31F9F439A7\",\"totalSupply\":3537129465723189637251199},\"UseSwapPricing\":false}}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9",
				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			},
			tokenOut:          "0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4",
			expectedAmountOut: &poolPkg.TokenAmount{Token: "0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4", Amount: bignumber.NewBig10("1810460589430022628736")},
			expectedErr:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var extra Extra
			err := json.Unmarshal([]byte(tc.entityPool.Extra), &extra)
			assert.Nil(t, err)

			extra.Vault.PriceFeed.PriceFeedProxies["0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9"].Timestamp = uint32(time.Now().Unix())
			extra.Vault.PriceFeed.PriceFeedProxies["0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4"].Timestamp = uint32(time.Now().Unix())

			extraBytes, err := json.Marshal(&extra)
			assert.Nil(t, err)

			tc.entityPool.Extra = string(extraBytes)
			pool, _ := NewPoolSimulator(tc.entityPool)

			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
				return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func setBlockPinFixtureValue(value any, address common.Address) any {
	v := reflect.New(reflect.TypeOf(value)).Elem()
	v.Set(reflect.ValueOf(value))
	var fill func(reflect.Value)
	fill = func(v reflect.Value) {
		if !v.CanSet() {
			return
		}
		if v.Type() == reflect.TypeOf(common.Address{}) {
			v.Set(reflect.ValueOf(address))
			return
		}
		switch v.Kind() {
		case reflect.Bool:
			v.SetBool(true)
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				fill(v.Field(i))
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				fill(v.Index(i))
			}
		}
	}
	fill(v)
	return v.Interface()
}

func TestGetNewPoolStatePinsAllCallsToFirstBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		addr := "0x1111111111111111111111111111111111111111"
		resolver := func(_ common.Address, method abi.Method, inputs []any) ([]any, error) {
			if method.Name == "getStates" && len(method.Outputs) == 3 {
				count := 2
				if len(inputs) > 0 {
					if tokens, ok := inputs[0].([]common.Address); ok {
						count = len(tokens)
					}
				}
				addresses := make([]common.Address, count)
				for i := range addresses {
					addresses[i] = common.HexToAddress(addr)
				}
				ints := make([]*big.Int, 3+count*2)
				for i := range ints {
					ints[i] = big.NewInt(1)
				}
				return []any{addresses, ints, make([]bool, 2)}, nil
			}
			if method.Name == "priceSampleSpace" {
				values := make([]any, len(method.Outputs))
				for i, output := range method.Outputs {
					values[i] = ethrpcmock.Value(output.Type)
					v := reflect.ValueOf(values[i])
					if v.Kind() == reflect.Ptr {
						values[i] = big.NewInt(2)
					} else {
						v = reflect.New(v.Type()).Elem()
						v.SetUint(2)
						values[i] = v.Interface()
					}
				}
				return values, nil
			}
			values := make([]any, len(method.Outputs))
			for i, output := range method.Outputs {
				values[i] = setBlockPinFixtureValue(ethrpcmock.Value(output.Type), common.HexToAddress(addr))
			}
			return values, nil
		}
		srv, recorder := ethrpcmock.NewWithResolver(t, resolver, chainlinkABI, fastPriceFeedV1ABI, fastPriceFeedV2ABI, pancakePairABI, priceFeedABI, vaultABI, vaultPriceFeedABI, erc20ABI)
		client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x5555555555555555555555555555555555555555"))
		tracker, err := NewPoolTracker(&Config{DexID: DexTypeQuickperps, VaultAddress: addr}, client)
		require.NoError(t, err)
		got, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: addr}, pool.GetNewPoolStateParams{})
		require.NoError(t, err)

		require.Equal(t, uint64(ethrpcmock.Block), got.BlockNumber)

		require.NotEmpty(t, got.Reserves)
		ethrpcmock.CheckPool(t, "quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlock", got)
		ethrpcmock.CheckRequests(t, "quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlock", recorder, [2]string{"tryAggregate", "tryBlockAndAggregate"})
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
	})
}

func TestGetNewPoolStatePinsAllCallsToFirstBlockV1(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		addr := "0x1111111111111111111111111111111111111111"
		resolver := func(_ common.Address, method abi.Method, inputs []any) ([]any, error) {
			if method.Name == "getStates" && len(method.Outputs) == 3 {
				count := 2
				if len(inputs) > 0 {
					if tokens, ok := inputs[0].([]common.Address); ok {
						count = len(tokens)
					}
				}
				addresses := make([]common.Address, count)
				for i := range addresses {
					addresses[i] = common.HexToAddress(addr)
				}
				ints := make([]*big.Int, 3+count*2)
				for i := range ints {
					ints[i] = big.NewInt(1)
				}
				return []any{addresses, ints, make([]bool, 2)}, nil
			}
			if method.Name == "priceSampleSpace" {
				values := make([]any, len(method.Outputs))
				for i, output := range method.Outputs {
					values[i] = ethrpcmock.Value(output.Type)
					v := reflect.ValueOf(values[i])
					if v.Kind() == reflect.Ptr {
						values[i] = big.NewInt(2)
					} else {
						v = reflect.New(v.Type()).Elem()
						v.SetUint(2)
						values[i] = v.Interface()
					}
				}
				return values, nil
			}
			values := make([]any, len(method.Outputs))
			for i, output := range method.Outputs {
				values[i] = setBlockPinFixtureValue(ethrpcmock.Value(output.Type), common.HexToAddress(addr))
			}
			return values, nil
		}
		srv, recorder := ethrpcmock.NewWithResolver(t, resolver, chainlinkABI, fastPriceFeedV1ABI, fastPriceFeedV2ABI, pancakePairABI, priceFeedABI, vaultABI, vaultPriceFeedABI, erc20ABI)
		client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x5555555555555555555555555555555555555555"))
		tracker, err := NewPoolTracker(&Config{DexID: DexTypeQuickperps, VaultAddress: addr, UseSecondaryPriceFeedV1: true}, client)
		require.NoError(t, err)
		got, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: addr}, pool.GetNewPoolStateParams{})
		require.NoError(t, err)

		require.Equal(t, uint64(ethrpcmock.Block), got.BlockNumber)

		require.NotEmpty(t, got.Reserves)
		ethrpcmock.CheckPool(t, "quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlockV1", got)
		ethrpcmock.CheckRequests(t, "quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlockV1", recorder, [2]string{"tryAggregate", "tryBlockAndAggregate"})
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
	})
}
