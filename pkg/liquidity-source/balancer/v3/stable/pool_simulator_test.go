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
	_          = json.Unmarshal([]byte(`{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1757384774,"reserves":["413690937750307354847","2327922153052532612174"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","symbol":"wstETH","decimals":18,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"20000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"423752596810938377498\",\"2833759539011177586101\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1024321681096877127\",\"1217291366592888852\"],\"buffs\":[{\"dRate\":[\"976255\",\"976255817341\",\"976255817341645373\",\"976255817341645373456045\",\"976255817341645373456045753577\"],\"rRate\":[\"1024321\",\"1024321681096\",\"1024321681096877127\",\"1024321681096877127977750\",\"1024321681096877127977750950000\"]},{\"dRate\":[\"996629\",\"996629442697\",\"996629442697471179\",\"996629442697471179789157\",\"996629442697471179789157582365\"],\"rRate\":[\"1003381\",\"1003381956380\",\"1003381956380303285\",\"1003381956380303285385258\",\"1003381956380303285385258382000\"]}],\"surge\":{},\"ampParam\":\"5000000\"}","staticExtra":"{\"buffs\":[\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\"]}","blockNumber":23322539}`), &entityPool)
	poolSim    = lo.Must(NewPoolSimulator(entityPool))
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()

	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		expectedAmountOut := "825444678056933274"
		expectedSwapFee := "19525116346832"

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

	t.Run("2. Swap from token 1 to token 0 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

		s := poolSim

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountOut := "1211410413259017388"
		expectedSwapFee := "19932588853950"

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
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(825000), // less than MinTradeAmount (1000000)
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

	// Mock state from https://etherscan.io/tx/0x92f38cf0d3c11276c220d4790c57a1f316b99a5e676e182b81d576b71f779012
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","weight":1,"swappable":true},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","weight":1,"swappable":true}],"extra":"{\"hook\":{\"enableHookAdjustedAmounts\":false,\"shouldCallComputeDynamicSwapFee\":false,\"shouldCallBeforeSwap\":false,\"shouldCallAfterSwap\":false},\"fee\":\"20000000000000\",\"aggrFee\":\"100000000000000000\",\"ampParam\":\"5000000\",\"balsE18\":[\"625134427981060649446\",\"2193655709385971229274\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009146942992102450\",\"1190985849893040213\"]}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

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

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountIn := "1211469115773124612"
		expectedSwapFee := "23654075436065"

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

	t.Run("2. Swap from token 1 to token 0 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		expectedAmountIn := "825483459061033533"
		expectedSwapFee := "16454022395199"

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
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

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
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("687804073931103275644", 10)
		reserves[1], _ = new(big.Int).SetString("1783969556654743519024", 10)

		tokenAmountOut := poolpkg.TokenAmount{
			Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			Amount: big.NewInt(825000), // less than MinTradeAmount (1000000)
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
		poolStr := `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","weight":1,"swappable":true},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","weight":1,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"20000000000000\",\"aggrFee\":\"100000000000000000\",\"ampParam\":\"5000000\",\"balsE18\":[\"625134427981060649446\",\"2193655709385971229274\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009146942992102450\",\"1190985849893040213\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

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
