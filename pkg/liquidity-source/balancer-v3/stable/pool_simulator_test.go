package stable

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/vault"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountOut := "1000347432097063736"
		expectedSwapFee := "23602591917350"

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
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountOut := "999611352568542440"
		expectedSwapFee := "16947291272107"

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
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
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
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
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

	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735803357,"reserves":["687804073931103275644","1783969556654743519024"],"tokens":[{"address":"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9","weight":1,"swappable":true},{"address":"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29","weight":1,"swappable":true}],"extra":"{\"hooksConfig\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"staticSwapFeePercentage\":\"20000000000000\",\"aggregateSwapFeePercentage\":\"500000000000000000\",\"amplificationParameter\":\"5000000\",\"balancesLiveScaled18\":[\"694069210892948295209\",\"2124492373418339554414\"],\"decimalScalingFactors\":[\"1\",\"1\"],\"tokenRates\":[\"1009108897721464489\",\"1190879275654308905\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21535329}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(73183418984294781),
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		// expected
		expectedAmountOut := "73208894800314286"
		expectedSwapFee := "1727318373403"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountIn := "1179719723071294910"
		expectedSwapFee := "23594394461425"

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

	t.Run("1. Swap from token 1 to token 0 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountIn := "847694017912779287"
		expectedSwapFee := "16953880358256"

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
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
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
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},                            // tokenRates
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"), uint256.MustFromDecimal("2124492373418339554414")}, // balancesLiveScaled18
				uint256.NewInt(20000000000000),     // swapFeePercentage
				uint256.NewInt(500000000000000000), // aggregateSwapFeePercentage
			),
			currentAmp: uint256.NewInt(5000000),
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(840000), // less than MINIMUM_TRADE_AMOUNT (1000000)
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

	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735803357,"reserves":["687804073931103275644","1783969556654743519024"],"tokens":[{"address":"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9","weight":1,"swappable":true},{"address":"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29","weight":1,"swappable":true}],"extra":"{\"hooksConfig\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"staticSwapFeePercentage\":\"20000000000000\",\"aggregateSwapFeePercentage\":\"500000000000000000\",\"amplificationParameter\":\"5000000\",\"balancesLiveScaled18\":[\"694069210892948295209\",\"2124492373418339554414\"],\"decimalScalingFactors\":[\"1\",\"1\"],\"tokenRates\":[\"1009108897721464489\",\"1190879275654308905\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21535329}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		// expected
		expectedAmountIn := "847694017912779287"
		expectedSwapFee := "16953880358256"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return s.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		// assert
		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})
}
