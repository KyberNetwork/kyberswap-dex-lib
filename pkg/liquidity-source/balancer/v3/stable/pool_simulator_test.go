package stable

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/vault"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1751293016,"reserves":["687804073931103275644","1783969556654743519024"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"}],"extra":"{\"hook\":{},\"fee\":\"20000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"694069210892948295209\",\"2124492373418339554414\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009108897721464489\",\"1190879275654308905\"],\"buffs\":[{\"rate\":\"2000000000000000000\"},{\"rate\":\"2000000000000000000\"}],\"surge\":{},\"ampParam\":\"5000000\"}","staticExtra":"{\"buffs\":[\"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9\",\"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29\"]}","blockNumber":22817774}`), &entityPool)
	poolSim    = lo.Must(NewPoolSimulator(entityPool))
)

func TestUnderlyingTokenCalcAmountOut(t *testing.T) {
	t.Parallel()

	t.Run("1. Swap from underlying token 0 to underlying token 1 successful", func(t *testing.T) {
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(2e18),
		}
		tokenOut := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		expectedAmountOut := "1695318119225604898"
		expectedSwapFee := "20000000000000"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap from underlying token 1 to underlying token 0 successful", func(t *testing.T) {
		s := poolSim

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(2e18),
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountOut := "2359341618927200470"
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
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(799999), // less than MinTradeAmount (1000000)
		}
		tokenOut := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountInTooSmall, err)
	})

	t.Run("4. AmountOut is too small", func(t *testing.T) {
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(1680000), // less than MinTradeAmount (1000000)
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})

	// // // Mock state from https://etherscan.io/tx/0x92f38cf0d3c11276c220d4790c57a1f316b99a5e676e182b81d576b71f779012
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","weight":1,"swappable":true},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","weight":1,"swappable":true}],"extra":"{\"hook\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"fee\":\"20000000000000\",\"aggrFee\":\"100000000000000000\",\"ampParam\":\"5000000\",\"balsE18\":[\"625134427981060649446\",\"2193655709385971229274\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009146942992102450\",\"1190985849893040213\"],\"buffs\":[{\"rate\":\"1000000000000000000\"},{\"rate\":\"1000000000000000000\"}]}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\",\"buffs\":[\"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9\",\"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29\"]}","blockNumber":21536418}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("1189123158260799643", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: amountIn,
		}
		tokenOut := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

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

func TestUnderlyingTokenCalcAmountIn(t *testing.T) {
	t.Parallel()
	t.Run("1. Swap from underlying token 0 to underlyingtoken 1 successful", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(2e18),
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountIn := "2359439446142589824"
		expectedSwapFee := "23594394461425"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap from underlying token 1 to underlyingtoken 0 successful", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(2e18),
		}
		tokenIn := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		expectedAmountIn := "1695388035825558578"
		expectedSwapFee := "16953880358256"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("3. AmountIn is too small", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(799999), // less than MinTradeAmount (1000000)
		}
		tokenIn := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountInTooSmall, err)
	})

	t.Run("4. AmountOut is too small", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(1680000), // less than MinTradeAmount (1000000)
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})

	// Mock state from https://etherscan.io/tx/0x92f38cf0d3c11276c220d4790c57a1f316b99a5e676e182b81d576b71f779012
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","weight":1,"swappable":true},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","weight":1,"swappable":true}],"extra":"{\"hook\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"fee\":\"20000000000000\",\"aggrFee\":\"100000000000000000\",\"ampParam\":\"5000000\",\"balsE18\":[\"625134427981060649446\",\"2193655709385971229274\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009146942992102450\",\"1190985849893040213\"],\"buffs\":[{\"rate\":\"1000000000000000000\"},{\"rate\":\"1000000000000000000\"}]}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\",\"buffs\":[\"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9\",\"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29\"]}","blockNumber":21536418}`

		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool)
		assert.Nil(t, err)

		amountOut, _ := new(big.Int).SetString("1008017884740935660", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: amountOut,
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

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

func TestWrappedTokenCalcAmountOut(t *testing.T) {
	t.Parallel()

	t.Run("1. Swap from wrapped token 0 to wrappedtoken 1 successful", func(t *testing.T) {
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountOut := "847659059612802449"
		expectedSwapFee := "20000000000000"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap from wrapped token 1 to wrapped token 0 successful", func(t *testing.T) {
		s := poolSim

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
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(799999), // less than MinTradeAmount (1000000)
		}
		tokenOut := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountInTooSmall, err)
	})

	t.Run("4. AmountOut is too small", func(t *testing.T) {
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(840000), // less than MinTradeAmount (1000000)
		}
		tokenOut := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})
}

func TestWrappedTokenCalcAmountIn(t *testing.T) {
	t.Parallel()
	t.Run("1. Swap from wrapped token 0 to wrapped token 1 successful", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountIn := "1179719723071294912"
		expectedSwapFee := "23594394461425"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap from wrapped token 1 to wrapped token 0 successful", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountIn := "847694017912779289"
		expectedSwapFee := "16953880358256"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("3. AmountIn is too small", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(799999), // less than MinTradeAmount (1000000)
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountInTooSmall, err)
	})

	t.Run("4. AmountOut is too small", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(840000), // less than MinTradeAmount (1000000)
		}
		tokenIn := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})
}

func TestBufferSwaps(t *testing.T) {
	t.Parallel()

	t.Run("1. Swap, calcAmountOut, from underlying token 0 to wrapped token 0 successful", func(t *testing.T) {
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9"

		expectedAmountOut := "500000000000000000"
		expectedSwapFee := "0"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap, calcAmountIn, from underlying token 0 to wrapped token 0 successful", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9",
			Amount: big.NewInt(0.5e18),
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountIn := "1000000000000000000"
		expectedSwapFee := "0"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("1. Swap, calcAmountOut, from wrapped token 1 to underlying token 1 successful", func(t *testing.T) {
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		expectedAmountOut := "2000000000000000000"
		expectedSwapFee := "0"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})

	t.Run("2. Swap, calcAmountIn, from wrapped token 1 to underlying token 1 successful", func(t *testing.T) {
		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(2e18),
		}
		tokenIn := "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"

		expectedAmountIn := "1000000000000000000"
		expectedSwapFee := "0"

		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Nil(t, err)

		assert.Equal(t, expectedAmountIn, result.TokenAmountIn.Amount.String())
		assert.Equal(t, expectedSwapFee, result.Fee.Amount.String())
	})
}

func TestCanSwapTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Underlying Swap",
			input:    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expected: []string{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9", "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29"},
		},
		{
			name:     "Wrapped Swap",
			input:    "0x775f661b0bd1739349b9a2a3ef60be277c5d2d29",
			expected: []string{"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := poolSim.CanSwapTo(tc.input)
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}
