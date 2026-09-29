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

// multiLpTestPool mirrors the jEUR/USDC multi-LP pool with a tracked state pulled
// from Base mainnet (pool 0x67ae...bfb94) at the time of writing: EUR/USD price
// 1.16789, fee 0.15%, 500k jEUR mint capacity, 12 jEUR outstanding (redeemable).
// Expected amounts below are computed with the pool's own exact integer formula
// (SynthereumMultiLpLiquidityPoolLib._calculateMint/_calculateRedeem, both
// PreciseUnitMath-floor-rounded), cross-checked against getMintTradeInfo/
// getRedeemTradeInfo on-chain.
func multiLpTestPool(t *testing.T) entity.Pool {
	t.Helper()
	extra := Extra{
		Price:         uint256.MustFromDecimal("1167890000000000000"),
		FeePercentage: uint256.MustFromDecimal("1500000000000000"),
		MaxSynthCap:   uint256.MustFromDecimal("500000000000000000000000"),
		TotalSynth:    uint256.MustFromDecimal("12000000000000000000"),
	}
	return entity.Pool{
		Address:  "0x67aefc812ec0a83a327c05d6e7913c35b48bfb94",
		Exchange: "synthereum",
		Type:     DexType,
		Reserves: []string{"14014680", "500000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: usdcAddress, Decimals: 6, Swappable: true},
			{Address: jeurAddress, Decimals: 18, Swappable: true},
		},
		Extra:       mustMarshal(t, extra),
		StaticExtra: mustMarshal(t, StaticExtra{PoolType: PoolTypeMultiLP}),
	}
}

// wrapperTestPool mirrors the jEUR/EURC fixed-rate wrapper. WrapperSynthCap
// (totalSyntheticTokens(), the wrapper's own binding unwrap check) is deliberately
// set below the vault-equivalent of WrapperReserve (607k EURC * 1e12 = 607k jEUR)
// so tests exercise the wrapper's own cap as the actual binding constraint, the way
// it is on-chain today.
func wrapperTestPool(t *testing.T) entity.Pool {
	t.Helper()
	extra := Extra{
		WrapperReserve:  uint256.MustFromDecimal("607000000000"),
		WrapperSynthCap: uint256.MustFromDecimal("605000000000000000000000"),
		WrapperRate:     uint256.MustFromDecimal("1000000000000000000"),
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
			PoolType: PoolTypeWrapper,
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
			expectedAmountOut: "854960655541189666835",
			expectedFee:       "1500000",
		},
		{
			name:        "multi-lp: mint above maxTokensCapacity is refused",
			entityPool:  multiLpTestPool,
			tokenIn:     usdcAddress,
			amountIn:    "700000000000", // 700k USDC -> ~598k jEUR > 500k cap
			tokenOut:    jeurAddress,
			expectedErr: ErrExceedsMaxCapacity,
		},
		{
			name:              "multi-lp: redeem 10 jEUR -> USDC net of 0.15% fee",
			entityPool:        multiLpTestPool,
			tokenIn:           jeurAddress,
			amountIn:          "10000000000000000000",
			tokenOut:          usdcAddress,
			expectedAmountOut: "11661382",
			expectedFee:       "17518",
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
			name:        "wrapper: unwrap above capacity is refused",
			entityPool:  wrapperTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "700000000000000000000000", // 700k jEUR exceeds both the vault- and synth-cap side
			tokenOut:    eurcAddress,
			expectedErr: ErrInsufficientWrapReserve,
		},
		{
			// Regression check for the on-chain unwrap cap bug: 606k jEUR is an exact
			// multiple of the scaling factor and sits under the vault-derived reserve
			// (607k EURC == 607k jEUR-equivalent), so a vault-only check would have
			// wrongly accepted it. FixedRateLendingWrapper.unwrap's actual guard is
			// totalSyntheticTokens() (605k jEUR here), which this amount exceeds.
			name:        "wrapper: unwrap within vault reserve but above totalSyntheticTokens is refused",
			entityPool:  wrapperTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "606000000000000000000000",
			tokenOut:    eurcAddress,
			expectedErr: ErrInsufficientWrapReserve,
		},
		{
			// FixedRateLendingWrapper.unwrap reverts with 'Wrong synth token rounding'
			// for any amount that isn't an exact multiple of the scaling factor when
			// conversionRate() == 1e18 — it does not floor the remainder.
			name:        "wrapper: unwrap not a multiple of the scaling factor reverts",
			entityPool:  wrapperTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "100000000000000000001", // 100 jEUR + 1 wei
			tokenOut:    eurcAddress,
			expectedErr: ErrWrongSynthTokenRounding,
		},
		{
			name:        "wrapper: dust unwrap (not a multiple of the scaling factor) reverts",
			entityPool:  wrapperTestPool,
			tokenIn:     jeurAddress,
			amountIn:    "500000000000", // 5e11 wei jEUR < 1e12
			tokenOut:    eurcAddress,
			expectedErr: ErrWrongSynthTokenRounding,
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

// TestCalcAmountOut_WrapperNonUnityRate covers the unwrap branch FixedRateLendingWrapper
// takes when conversionRate() != 1e18 ('collateralAmount = synthAmount.div(rate)/SCALING_FACTOR',
// a two-step floor with no divisibility revert) — not reachable with today's deployed
// rate (1e18) but present in the source and worth keeping green if the rate is ever changed.
func TestCalcAmountOut_WrapperNonUnityRate(t *testing.T) {
	t.Parallel()

	p := wrapperTestPool(t)
	extra := Extra{
		WrapperReserve:  uint256.MustFromDecimal("607000000000"),
		WrapperSynthCap: uint256.MustFromDecimal("605000000000000000000000"),
		WrapperRate:     uint256.MustFromDecimal("2000000000000000000"), // 2.0
	}
	p.Extra = mustMarshal(t, extra)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	// floor(100e18 * 1e18 / 2e18) / 1e12 = 50e18 / 1e12 = 50000000 (50 EURC)
	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("100000000000000000000")},
		TokenOut:      eurcAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, "50000000", result.TokenAmountOut.Amount.String())

	// floor(1 * 1e18 / 2e18) / 1e12 = 0 -- floors to zero rather than reverting,
	// unlike the rate == 1e18 branch.
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("1")},
		TokenOut:      eurcAddress,
	})
	assert.ErrorIs(t, err, ErrZeroAmountOut)
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

	// mint 1000 USDC -> 854.960655541189666835 jEUR
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
	assert.Equal(t, "932910532", redeemResult.TokenAmountOut.Amount.String())

	// mint capacity decreased by the minted amount
	assert.Equal(t, "499145039344458810333165", sim.extra.MaxSynthCap.Dec())
	assert.Equal(t, "866960655541189666835", sim.extra.TotalSynth.Dec())

	// Info.Reserves (display-only) moved in step: +amountIn on the input side,
	// -amountOut on the output side
	assert.Equal(t, "1014014680", sim.Info.Reserves[0].String())
	assert.Equal(t, "499145039344458810333165", sim.Info.Reserves[1].String())

	// the clone kept the pre-swap state, including Info.Reserves -- CloneState must
	// deep-copy the Reserves slice since UpdateBalance now writes it by index
	assert.Equal(t, "14014680", cloned.Info.Reserves[0].String())
	assert.Equal(t, "500000000000000000000000", cloned.Info.Reserves[1].String())
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

	// unwrap 600k jEUR -> 600k EURC, leaving 7k EURC / 5k jEUR of capacity
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
	assert.Equal(t, "5000000000000000000000", sim.extra.WrapperSynthCap.Dec())

	// a further 8k jEUR unwrap exceeds the remaining capacity
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: bignumber.NewBig10("8000000000000000000000")},
		TokenOut:      eurcAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientWrapReserve)

	// wrapping refills both the vault reserve and the synthetic-token headroom
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
	assert.Equal(t, "7000000000000000000000", sim.extra.WrapperSynthCap.Dec())
	assert.Equal(t, "9000000000", sim.Info.Reserves[0].String())
	assert.Equal(t, "100598000000000000000000000", sim.Info.Reserves[1].String())
}

func TestCalcAmountOut_WrapBoundedByVaultMaxDeposit(t *testing.T) {
	t.Parallel()

	p := wrapperTestPool(t)
	extra := Extra{
		WrapperReserve:    uint256.MustFromDecimal("607000000000"),
		WrapperSynthCap:   uint256.MustFromDecimal("605000000000000000000000"),
		WrapperRate:       uint256.MustFromDecimal("1000000000000000000"),
		WrapperMaxDeposit: uint256.MustFromDecimal("1000000000"), // vault will only accept 1000 more EURC
	}
	p.Extra = mustMarshal(t, extra)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	// within the vault's remaining deposit capacity
	result, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: eurcAddress, Amount: bignumber.NewBig10("1000000000")},
		TokenOut:      jeurAddress,
	})
	require.NoError(t, err)
	assert.Equal(t, "1000000000000000000000", result.TokenAmountOut.Amount.String())

	// one more wei of collateral exceeds the vault's maxDeposit -- would revert
	// inside the wrapper's delegatecall to the lending module's deposit()
	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: eurcAddress, Amount: bignumber.NewBig10("1000000001")},
		TokenOut:      jeurAddress,
	})
	assert.ErrorIs(t, err, ErrExceedsWrapCapacity)
}

// The vault's withdrawable liquidity is an independent, exact cap: on Base the
// wrapper's shares are worth ~631k EURC (previewRedeem) but only ~453k EURC is
// actually withdrawable (maxWithdraw), because a Morpho vault lends the rest out.
// Quoting into that gap reverts NotEnoughLiquidity() on-chain, so WrapperReserve
// must be maxWithdraw and must bind even when it is below the wrapper's own cap.
func TestUnwrap_BoundedByVaultWithdrawableLiquidity(t *testing.T) {
	t.Parallel()

	p := wrapperTestPool(t)
	extra := Extra{
		// vault liquidity (453k EURC) below the wrapper's own cap (605k jEUR)
		WrapperReserve:  uint256.MustFromDecimal("453364095963"),
		WrapperSynthCap: uint256.MustFromDecimal("605000000000000000000000"),
		WrapperRate:     uint256.MustFromDecimal("1000000000000000000"),
	}
	p.Extra = mustMarshal(t, extra)

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	atLimit := uint256.MustFromDecimal("453364095963000000000000") // maxWithdraw * 10^12
	overLimit := new(uint256.Int).Add(atLimit, uint256.MustFromDecimal("1000000000000"))

	res, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: atLimit.ToBig()},
		TokenOut:      eurcAddress,
	})
	require.NoError(t, err, "exactly maxWithdraw must quote")
	assert.Equal(t, "453364095963", res.TokenAmountOut.Amount.String())

	_, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
		TokenAmountIn: poolPkg.TokenAmount{Token: jeurAddress, Amount: overLimit.ToBig()},
		TokenOut:      eurcAddress,
	})
	assert.ErrorIs(t, err, ErrInsufficientWrapReserve,
		"one unit above maxWithdraw reverts NotEnoughLiquidity() on-chain")
}
