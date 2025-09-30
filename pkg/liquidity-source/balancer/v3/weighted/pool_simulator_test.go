package weighted

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/vault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xdaba3d8ccf79ef289a7e2dbce51871b39ea445a2","exchange":"balancer-v3-weighted","type":"balancer-v3-weighted","timestamp":1757384774,"reserves":["54968478987261442046","2951446654676525480148856"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x77146784315ba81904d654466968e3a7c196d1f3","symbol":"TREE","decimals":18,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"8000000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"56305404803570006222\",\"2951446654676525480148856\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1024321681096877127\",\"1000000000000000000\"],\"buffs\":[{\"dRate\":[\"976255\",\"976255817341\",\"976255817341645373\",\"976255817341645373456045\",\"976255817341645373456045753577\"],\"rRate\":[\"1024321\",\"1024321681096\",\"1024321681096877127\",\"1024321681096877127977750\",\"1024321681096877127977750950000\"]},null],\"normalizedWeights\":[\"200000000000000000\",\"800000000000000000\"]}","staticExtra":"{\"buffs\":[\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"\"]}","blockNumber":23322539}`), &entityPool)
	poolSim    = lo.Must(NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool}))
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	t.Run("1. Swap from token 0 to token 1 successful", func(t *testing.T) {
		reserves := make([]*big.Int, 2)
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		tokenAmountIn := pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0x77146784315ba81904d654466968e3a7c196d1f3"

		expectedAmountOut := "12858515052701213115427"
		expectedSwapFee := "7810046538733162"

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
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

		tokenAmountIn := pool.TokenAmount{
			Token:  "0x77146784315ba81904d654466968e3a7c196d1f3",
			Amount: big.NewInt(1e18),
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountOut := "75698355643337"
		expectedSwapFee := "8000000000000000"

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
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

		tokenAmountIn := pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(799999), // less than MinTradeAmount (1000000)
		}
		tokenOut := "0x77146784315ba81904d654466968e3a7c196d1f3"

		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
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

		tokenAmountIn := pool.TokenAmount{
			Token:  "0x77146784315ba81904d654466968e3a7c196d1f3",
			Amount: big.NewInt(1300000), // less than MinTradeAmount (1000000)
		}
		tokenOut := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})

	// Mock state from https://gnosisscan.io/tx/0x14579e3588ad7a76bfd850168baf41a581ed049c3a355a6c3c891cdccc2b0836
	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0x272d6be442e30d7c87390edeb9b96f1e84cecd8d","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0x773cda0cade2a3d86e6d4e30699d40bb95174ff2","weight":1,"swappable":true},{"address":"0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644","weight":1,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"2500000000000000\",\"aggrFee\":\"100000000000000000\",\"normalizedWeights\":[\"500000000000000000\",\"500000000000000000\"],\"balsE18\":[\"718362766363614682950\",\"8898955182296732614690\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1189577407040530520\",\"1000892729180982664\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

		var entityPool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &entityPool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("999108067073568238", 10)

		tokenAmountIn := pool.TokenAmount{
			Token:  "0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644",
			Amount: amountIn,
		}
		tokenOut := "0x773cda0cade2a3d86e6d4e30699d40bb95174ff2"

		// expected
		expectedAmountOut := "67682487794870862"
		expectedSwapFee := "2497770167683920"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return s.CalcAmountOut(pool.CalcAmountOutParams{
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
		poolStr := `{"address":"0x2c6c34a046ae1bfb5543ffd32745cc5e2ac7fb34","exchange":"balancer-v3-weighted","type":"balancer-v3-weighted","timestamp":1740366843,"reserves":["92522708649454779998815","360573774832263481"],"tokens":[{"address":"0x3082cc23568ea640225c2467653db90e9250aaa0","weight":1,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","weight":1,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"5000000000000000\",\"aggrFee\":\"0\",\"normalizedWeights\":[\"750000000000000000\",\"250000000000000000\"],\"balsE18\":[\"92522708649454779998815\",\"360573774832263481\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\",\"isPoolInitialized\":true}","blockNumber":309271722}`

		var entityPool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &entityPool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
		assert.Nil(t, err)

		amountIn, _ := new(big.Int).SetString("1000000000000000000", 10)

		tokenAmountIn := pool.TokenAmount{
			Token:  "0x3082cc23568ea640225c2467653db90e9250aaa0",
			Amount: amountIn,
		}
		tokenOut := "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"

		// expected
		expectedAmountOut := "11632707084358"
		expectedSwapFee := "5000000000000000"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return s.CalcAmountOut(pool.CalcAmountOutParams{
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
		reserves[0], _ = new(big.Int).SetString("720118889801352582380", 10)
		reserves[1], _ = new(big.Int).SetString("8876513774745869289662", 10)

		tokenAmountOut := pool.TokenAmount{
			Token:  "0x77146784315ba81904d654466968e3a7c196d1f3",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		expectedAmountIn := "76924349808837"
		expectedSwapFee := "600782751970"

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(pool.CalcAmountInParams{
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

		tokenAmountOut := pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(1e18),
		}
		tokenIn := "0x77146784315ba81904d654466968e3a7c196d1f3"

		expectedAmountIn := "13358934186453660015697"
		expectedSwapFee := "106871473491629280126"

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(pool.CalcAmountInParams{
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

		tokenAmountOut := pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(799999), // less than MinTradeAmount (1000000)
		}
		tokenIn := "0x77146784315ba81904d654466968e3a7c196d1f3"

		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(pool.CalcAmountInParams{
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

		tokenAmountOut := pool.TokenAmount{
			Token:  "0x77146784315ba81904d654466968e3a7c196d1f3",
			Amount: big.NewInt(1000000), // less than MinTradeAmount (1000000)
		}
		tokenIn := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

		_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: tokenAmountOut,
				TokenIn:        tokenIn,
			})
		})

		assert.Error(t, err)
		assert.Equal(t, vault.ErrAmountOutTooSmall, err)
	})

	t.Run("5. should return OK", func(t *testing.T) {
		poolStr := `{"address":"0x272d6be442e30d7c87390edeb9b96f1e84cecd8d","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1735816509,"reserves":["619469949959861143118","1841897390394044699179"],"tokens":[{"address":"0x773cda0cade2a3d86e6d4e30699d40bb95174ff2","weight":1,"swappable":true},{"address":"0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644","weight":1,"swappable":true}],"extra":"{\"hook\":{},\"fee\":\"2500000000000000\",\"aggrFee\":\"100000000000000000\",\"normalizedWeights\":[\"500000000000000000\",\"500000000000000000\"],\"balsE18\":[\"718362766363614682950\",\"8898955182296732614690\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1189577407040530520\",\"1000892729180982664\"],\"isVaultPaused\":false,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false}","staticExtra":"{\"vault\":\"0xba1333333333a1ba1108e8412f11850a5c319ba9\",\"defaultHook\":\"\"}","blockNumber":21536418}`

		var entityPool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &entityPool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
		assert.Nil(t, err)

		amountOut, _ := new(big.Int).SetString("67682487794870862", 10)

		tokenAmountOut := pool.TokenAmount{
			Token:  "0x773cda0cade2a3d86e6d4e30699d40bb95174ff2",
			Amount: amountOut,
		}
		tokenIn := "0x7c16f0185a26db0ae7a9377f23bc18ea7ce5d644"

		// expected
		expectedAmountIn := "999108067073574531"
		expectedSwapFee := "2497770167683936"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return s.CalcAmountIn(pool.CalcAmountInParams{
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
