package synthereum

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	usdcAddress = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	eurcAddress = "0x60a3e35cc302bfa44cb288bc5a4f316fdb1adb42"
	jeurAddress = "0x4154550f4db74dc38d1fe98e1f3f28ed6dad627d"
)

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	bytes, err := json.Marshal(v)
	require.NoError(t, err)
	return string(bytes)
}

// multiLpTestPool mirrors the jEUR/USDC multi-LP pool with a tracked state of:
// 1 USDC -> 0.848725 jEUR net of fee (fee 0.15%), 1 jEUR -> 1.174706 USDC net of fee,
// 500k jEUR mint capacity, 12 jEUR outstanding (redeemable).
func multiLpTestPool(t *testing.T) entity.Pool {
	t.Helper()
	extra := Extra{
		MintProbeIn:    uint256.MustFromDecimal("1000000"),
		MintProbeOut:   uint256.MustFromDecimal("848725000000000000"),
		MintProbeFee:   uint256.MustFromDecimal("1500"),
		RedeemProbeIn:  uint256.MustFromDecimal("1000000000000000000"),
		RedeemProbeOut: uint256.MustFromDecimal("1174706"),
		RedeemProbeFee: uint256.MustFromDecimal("1764"),
		FeePercentage:  uint256.MustFromDecimal("1500000000000000"),
		MaxSynthCap:    uint256.MustFromDecimal("500000000000000000000000"),
		TotalSynth:     uint256.MustFromDecimal("12000000000000000000"),
	}
	return entity.Pool{
		Address:  "0x67aefc812ec0a83a327c05d6e7913c35b48bfb94",
		Exchange: "synthereum",
		Type:     DexType,
		Reserves: []string{"14096472", "500000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: usdcAddress, Decimals: 6, Swappable: true},
			{Address: jeurAddress, Decimals: 18, Swappable: true},
		},
		Extra:       mustMarshal(t, extra),
		StaticExtra: mustMarshal(t, StaticExtra{PoolType: poolTypeMultiLP}),
	}
}

// wrapperTestPool mirrors the jEUR/EURC fixed-rate wrapper with 607k EURC
// redeemable from the Morpho vault.
func wrapperTestPool(t *testing.T) entity.Pool {
	t.Helper()
	extra := Extra{
		WrapperReserve: uint256.MustFromDecimal("607000000000"),
	}
	return entity.Pool{
		Address:  "0x41b0667ea45a5401d95f9a5d281287630704b798",
		Exchange: "synthereum",
		Type:     DexType,
		Reserves: []string{"607000000000", defaultSynthReserve},
		Tokens: []*entity.PoolToken{
			{Address: eurcAddress, Decimals: 6, Swappable: true},
			{Address: jeurAddress, Decimals: 18, Swappable: true},
		},
		Extra: mustMarshal(t, extra),
		StaticExtra: mustMarshal(t, StaticExtra{
			PoolType: poolTypeWrapper,
			Vault:    "0xbeef086b8807dc5e5a1740c5e3a7c4c366ea6ab5",
		}),
	}
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		entityPool        func(*testing.T) entity.Pool
		tokenIn           string
		amountIn          string
		tokenOut          string
		expectedAmountOut string
		expectedFee       string
		expectedErr       error
	}{
		{
			name:              "multi-lp: mint 1000 USDC -> jEUR net of 0.15% fee",
			entityPool:        multiLpTestPool,
			tokenIn:           usdcAddress,
			amountIn:          "1000000000",
			tokenOut:          jeurAddress,
			expectedAmountOut: "848725000000000000000",
			expectedFee:       "1500000",
		},
		{
			name:        "multi-lp: mint above maxTokensCapacity is refused",
			entityPool:  multiLpTestPool,
			tokenIn:     usdcAddress,
			amountIn:    "700000000000", // 700k USDC -> ~594k jEUR > 500k cap
			tokenOut:    jeurAddress,
			expectedErr: ErrExceedsMaxCapacity,
		},
		{
			name:              "multi-lp: redeem 10 jEUR -> USDC net of 0.15% fee",
			entityPool:        multiLpTestPool,
			tokenIn:           jeurAddress,
			amountIn:          "10000000000000000000",
			tokenOut:          usdcAddress,
			expectedAmountOut: "11747060",
			expectedFee:       "17640",
		},
		{
			name:        "multi-lp: redeem above totalSyntheticTokens is refused",
			entityPool:  multiLpTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "13000000000000000000", // 13 jEUR > 12 jEUR outstanding
			tokenOut:    usdcAddress,
			expectedErr: ErrExceedsRedeemCapacity,
		},
		{
			name:        "multi-lp: zero amount in is refused",
			entityPool:  multiLpTestPool,
			tokenIn:     usdcAddress,
			amountIn:    "0",
			tokenOut:    jeurAddress,
			expectedErr: ErrInvalidAmountIn,
		},
		{
			name:              "wrapper: wrap 100 EURC -> 100 jEUR (x1e12, zero fee)",
			entityPool:        wrapperTestPool,
			tokenIn:           eurcAddress,
			amountIn:          "100000000",
			tokenOut:          jeurAddress,
			expectedAmountOut: "100000000000000000000",
			expectedFee:       "0",
		},
		{
			name:              "wrapper: unwrap 100 jEUR -> 100 EURC (/1e12, zero fee)",
			entityPool:        wrapperTestPool,
			tokenIn:           jeurAddress,
			amountIn:          "100000000000000000000",
			tokenOut:          eurcAddress,
			expectedAmountOut: "100000000",
			expectedFee:       "0",
		},
		{
			name:        "wrapper: unwrap above vault reserve is refused",
			entityPool:  wrapperTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "700000000000000000000000", // 700k jEUR > 607k EURC reserve
			tokenOut:    eurcAddress,
			expectedErr: ErrInsufficientWrapReserve,
		},
		{
			name:        "wrapper: dust unwrap rounding to zero is refused",
			entityPool:  wrapperTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "500000000000", // 5e11 wei jEUR < 1e12
			tokenOut:    eurcAddress,
			expectedErr: ErrZeroAmountOut,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sim, err := NewPoolSimulator(tc.entityPool(t))
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: poolPkg.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})

			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.tokenOut, result.TokenAmountOut.Token)
			assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount.String())
			assert.Equal(t, tc.expectedFee, result.Fee.Amount.String())
			assert.Positive(t, result.Gas)
		})
	}
}

func TestCalcAmountOut_TradeUnavailableWithoutTrackedState(t *testing.T) {
	t.Parallel()

	p := multiLpTestPool(t)
	p.Extra = "{}" // freshly listed pool, tracker has not run yet
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: usdcAddress, Amount: bignumber.NewBig10("1000000")},
		TokenOut:      jeurAddress,
	})
	assert.ErrorIs(t, err, ErrTradeUnavailable)

	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("1000000000000000000")},
		TokenOut:      usdcAddress,
	})
	assert.ErrorIs(t, err, ErrTradeUnavailable)
}

func TestUpdateBalance_MultiLp(t *testing.T) {
	t.Parallel()

	sim, err := NewPoolSimulator(multiLpTestPool(t))
	require.NoError(t, err)

	// redeeming 800 jEUR is impossible before the mint (only 12 jEUR outstanding)
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("800000000000000000000")},
		TokenOut:      usdcAddress,
	})
	assert.ErrorIs(t, err, ErrExceedsRedeemCapacity)

	// mint 1000 USDC -> 848.725 jEUR
	amountIn := poolPkg.TokenAmount{Token: usdcAddress, Amount: bignumber.NewBig10("1000000000")}
	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{TokenAmountIn: amountIn, TokenOut: jeurAddress})
	require.NoError(t, err)

	cloned := sim.CloneState().(*PoolSimulator)

	sim.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  amountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	// outstanding synthetic supply grew: the 800 jEUR redeem now passes
	redeemResult, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("800000000000000000000")},
		TokenOut:      usdcAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, "939764800", redeemResult.TokenAmountOut.Amount.String())

	// mint capacity decreased by the minted amount
	assert.Equal(t, "499151275000000000000000", sim.extra.MaxSynthCap.Dec())
	assert.Equal(t, "860725000000000000000", sim.extra.TotalSynth.Dec())

	// the clone kept the pre-swap state
	_, err = cloned.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("800000000000000000000")},
		TokenOut:      usdcAddress,
	})
	assert.ErrorIs(t, err, ErrExceedsRedeemCapacity)
}

func TestUpdateBalance_Wrapper(t *testing.T) {
	t.Parallel()

	sim, err := NewPoolSimulator(wrapperTestPool(t))
	require.NoError(t, err)

	// unwrap 600k jEUR -> 600k EURC, leaving 7k EURC in the vault
	amountIn := poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("600000000000000000000000")}
	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{TokenAmountIn: amountIn, TokenOut: eurcAddress})
	require.NoError(t, err)
	assert.Equal(t, "600000000000", result.TokenAmountOut.Amount.String())

	sim.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  amountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "7000000000", sim.extra.WrapperReserve.Dec())

	// a further 8k jEUR unwrap exceeds the remaining reserve
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("8000000000000000000000")},
		TokenOut:      eurcAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientWrapReserve)

	// wrapping refills the redeemable reserve
	wrapIn := poolPkg.TokenAmount{Token: eurcAddress, Amount: bignumber.NewBig10("2000000000")}
	wrapResult, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{TokenAmountIn: wrapIn, TokenOut: jeurAddress})
	require.NoError(t, err)
	sim.UpdateBalance(poolPkg.UpdateBalanceParams{
		TokenAmountIn:  wrapIn,
		TokenAmountOut: *wrapResult.TokenAmountOut,
		Fee:            *wrapResult.Fee,
		SwapInfo:       wrapResult.SwapInfo,
	})
	assert.Equal(t, "9000000000", sim.extra.WrapperReserve.Dec())
}
