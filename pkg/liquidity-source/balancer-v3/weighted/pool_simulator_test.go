package weighted

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

		expectedAmountOut := "14588365766046319212"
		expectedSwapFee := "2500000000000000"

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

		expectedAmountOut := "68085611533624911"
		expectedSwapFee := "2500000000000000"

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
			Amount: big.NewInt(1300000), // less than MINIMUM_TRADE_AMOUNT (1000000)
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

	// Mock state from https://gnosisscan.io/tx/0x14579e3588ad7a76bfd850168baf41a581ed049c3a355a6c3c891cdccc2b0836
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0x272d6be442e30d7c87390edeb9b96f1e84cecd8d","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0x773cda0cade2a3d86e6d4e30699d40bb95174ff2","weight":1,"swappable":true},{"address":"0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644","weight":1,"swappable":true}],"extra":"{\"hooksConfig\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"staticSwapFeePercentage\":\"2500000000000000\",\"aggregateSwapFeePercentage\":\"100000000000000000\",\"normalizedWeights\":[\"500000000000000000\",\"500000000000000000\"],\"balancesLiveScaled18\":[\"718362766363614682950\",\"8898955182296732614690\"],\"decimalScalingFactors\":[\"1\",\"1\"],\"tokenRates\":[\"1189577407040530520\",\"1000892729180982664\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("999108067073568238", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644",
			Amount: amountIn,
		}
		tokenOut := "0x773cda0cade2a3d86e6d4e30699d40bb95174ff2"

		// expected
		expectedAmountOut := "67682487794870862"
		expectedSwapFee := "2497770167683920"

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

	t.Run("6. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0x2c6c34a046ae1bfb5543ffd32745cc5e2ac7fb34","exchange":"balancer-v3-weighted","type":"balancer-v3-weighted","timestamp":1740366843,"reserves":["92522708649454779998815","360573774832263481"],"tokens":[{"address":"0x3082cc23568ea640225c2467653db90e9250aaa0","weight":1,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","weight":1,"swappable":true}],"extra":"{\"hooksConfig\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"staticSwapFeePercentage\":\"5000000000000000\",\"aggregateSwapFeePercentage\":\"0\",\"normalizedWeights\":[\"750000000000000000\",\"250000000000000000\"],\"balancesLiveScaled18\":[\"92522708649454779998815\",\"360573774832263481\"],\"decimalScalingFactors\":[\"1\",\"1\"],\"tokenRates\":[\"1000000000000000000\",\"1000000000000000000\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\",\"isPoolInitialized\":true}","blockNumber":309271722}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("1000000000000000000", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x3082cc23568ea640225c2467653db90e9250aaa0",
			Amount: amountIn,
		}
		tokenOut := "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"

		// expected
		expectedAmountOut := "11632707084358"
		expectedSwapFee := "5000000000000000"

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

func TestCalcAmountIn(t *testing.T) {
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

	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0x272d6be442e30d7c87390edeb9b96f1e84cecd8d","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0x773cda0cade2a3d86e6d4e30699d40bb95174ff2","weight":1,"swappable":true},{"address":"0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644","weight":1,"swappable":true}],"extra":"{\"hooksConfig\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"staticSwapFeePercentage\":\"2500000000000000\",\"aggregateSwapFeePercentage\":\"100000000000000000\",\"normalizedWeights\":[\"500000000000000000\",\"500000000000000000\"],\"balancesLiveScaled18\":[\"718362766363614682950\",\"8898955182296732614690\"],\"decimalScalingFactors\":[\"1\",\"1\"],\"tokenRates\":[\"1189577407040530520\",\"1000892729180982664\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountOut, _ := new(big.Int).SetString("67682487794870862", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x773cda0cade2a3d86e6d4e30699d40bb95174ff2",
			Amount: amountOut,
		}
		tokenIn := "0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644"

		// expected
		expectedAmountIn := "999108067073574530"
		expectedSwapFee := "2497770167683936"

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
