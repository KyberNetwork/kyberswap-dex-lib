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
	t.Parallel()
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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountOut := "847659059612802449"
		expectedSwapFee := "20000000000000"

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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountOut := "1179670809463600235"
		expectedSwapFee := "20000000000000"

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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(840000), // less than MINIMUM_TRADE_AMOUNT (1000000)
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

	// Mock state from https://etherscan.io/tx/0x92f38cf0d3c11276c220d4790c57a1f316b99a5e676e182b81d576b71f779012
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9","weight":1,"swappable":true},{"address":"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29","weight":1,"swappable":true}],"extra":"{\"hook\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"fee\":\"20000000000000\",\"aggrFee\":\"100000000000000000\",\"ampParam\":\"5000000\",\"balsE18\":[\"625134427981060649446\",\"2193655709385971229274\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009146942992102450\",\"1190985849893040213\"]}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("1189123158260799643", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: amountIn,
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		// expected
		expectedAmountOut := "1008017884740935660"
		expectedSwapFee := "23782463165215"

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
	t.Parallel()
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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountIn := "1179719723071294912"
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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
		}

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountIn := "847694017912779289"
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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
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
			vault: vault.New(hooks.NewNoOpHook(), shared.HooksConfig{},
				[]*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				[]*uint256.Int{uint256.NewInt(1009108897721464489), uint256.NewInt(1190879275654308905)},
				[]*uint256.Int{uint256.MustFromDecimal("694069210892948295209"),
					uint256.MustFromDecimal("2124492373418339554414")}, uint256.NewInt(20000000000000),
				uint256.NewInt(500000000000000000)),
			currentAmp: uint256.NewInt(5000000),
			buffers:    make([]*shared.ExtraBuffer, 2),
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

	// Mock state from https://etherscan.io/tx/0x92f38cf0d3c11276c220d4790c57a1f316b99a5e676e182b81d576b71f779012
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9","weight":1,"swappable":true},{"address":"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29","weight":1,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"20000000000000\",\"aggrFee\":\"100000000000000000\",\"ampParam\":\"5000000\",\"balsE18\":[\"625134427981060649446\",\"2193655709385971229274\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009146942992102450\",\"1190985849893040213\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountOut, _ := new(big.Int).SetString("1008017884740935660", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: amountOut,
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		// expected
		expectedAmountIn := "1189123158260799642"
		expectedSwapFee := "23782463165215"

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
