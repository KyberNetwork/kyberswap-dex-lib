package weighted

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/vault"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountOut := "12278617355364789838"
		expectedSwapFee := "2103630945830047"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap from token 1 to token 0 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountOut := "80912682117686948"
		expectedSwapFee := "2971053459918506"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("3. AmountIn is too small", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(799999), // less than MINIMUM_TRADE_AMOUNT (1000000)
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountInTooSmall, err)
	})

	t.Run("4. AmountOut is too small", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(991000), // less than MINIMUM_TRADE_AMOUNT (1000000)
		}
		tokenOut := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountIn := "68442734257727536"
		expectedSwapFee := "171106835644319"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap from token 1 to token 0 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountIn := "14710037031320992773"
		expectedSwapFee := "36775092578302482"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("3. AmountIn is too small", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(799999), // less than MINIMUM_TRADE_AMOUNT (1000000)
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountInTooSmall, err)
	})

	t.Run("4. AmountOut is too small", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
						"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
					},
				},
			},
			vault: vault.New(
				hooks.NewBaseHook(),
				shared.HooksConfig{},
				false, // isPoolInRecoveryMode
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},                                                                // decimalScalingFactors
				[]*uint256.Int{uint256.MustFromDecimal("1189479974914033532"), uint256.MustFromDecimal("1000890753869723446")},      // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("720118889801487374560"), uint256.MustFromDecimal("8876514414974844966787")}, // balancesLiveScaled18
				uint256.NewInt(2500000000000000),   // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			normalizedWeights: []*uint256.Int{uint256.NewInt(500000000000000000), uint256.NewInt(500000000000000000)},
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(999900), // less than MINIMUM_TRADE_AMOUNT (1000000)
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})
}
