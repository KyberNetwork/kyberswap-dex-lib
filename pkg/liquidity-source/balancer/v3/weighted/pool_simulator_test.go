package weighted

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/vault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

		expectedAmountOut := "75698355643402"
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

		expectedAmountIn := "76924349808904"
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

func TestWeightedPoolV2(t *testing.T) {
	t.Parallel()

	const (
		token0 = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
		token1 = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	)

	mustU256 := func(s string) *uint256.Int {
		v, err := uint256.FromDecimal(s)
		require.NoError(t, err)
		return v
	}

	buildSim := func(t *testing.T, chainID valueobject.ChainID, exchange string, bal0, bal1 string, weights [2]string, minBals []string) *base.PoolSimulator {
		t.Helper()
		extra := Extra{
			Extra: &shared.Extra{
				StaticSwapFeePercentage:    mustU256("3000000000000000"),
				AggregateSwapFeePercentage: mustU256("500000000000000000"),
				BalancesLiveScaled18:       []*uint256.Int{mustU256(bal0), mustU256(bal1)},
				DecimalScalingFactors:      []*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)},
				TokenRates:                 []*uint256.Int{mustU256("1000000000000000000"), mustU256("1000000000000000000")},
			},
			NormalizedWeights: []*uint256.Int{mustU256(weights[0]), mustU256(weights[1])},
		}
		if len(minBals) > 0 {
			extra.MinTokenBalances = []*uint256.Int{mustU256(minBals[0]), mustU256(minBals[1])}
		}

		extraBytes, err := json.Marshal(extra)
		require.NoError(t, err)
		staticBytes, err := json.Marshal(shared.StaticExtra{BufferTokens: []string{"", ""}})
		require.NoError(t, err)

		p := entity.Pool{
			Address:     "0x2222222222222222222222222222222222222222",
			Exchange:    exchange,
			Type:        DexType,
			BlockNumber: 42000000,
			Reserves:    entity.PoolReserves{bal0, bal1},
			Tokens: []*entity.PoolToken{
				{Address: token0, Decimals: 18, Swappable: true},
				{Address: token1, Decimals: 18, Swappable: true},
			},
			Extra:       string(extraBytes),
			StaticExtra: string(staticBytes),
		}

		sim, err := NewPoolSimulator(pool.FactoryParams{
			EntityPool: p,
			ChainID:    chainID,
		})
		require.NoError(t, err)
		return sim
	}

	t.Run("ExactIn +1 balance rounding differs from v1 and matches WeightedPool v2", func(t *testing.T) {
		weights := [2]string{"800000000000000000", "200000000000000000"}
		v1Sim := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "10000000", "10000000000000000000", weights, nil)
		v2Sim := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "10000000", "10000000000000000000", weights, []string{"1000000", "1000000"})

		params := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(2_000_000)},
			TokenOut:      token1,
		}

		v1Res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return v1Sim.CalcAmountOut(params)
		})
		require.NoError(t, err)
		require.Equal(t, "5167812005680028660", v1Res.TokenAmountOut.Amount.String())

		v2Res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return v2Sim.CalcAmountOut(params)
		})
		require.NoError(t, err)
		require.Equal(t, "5167811684339948710", v2Res.TokenAmountOut.Amount.String())
	})

	t.Run("ExactOut +1 balance rounding differs from v1 and matches WeightedPool v2", func(t *testing.T) {
		weights := [2]string{"500000000000000000", "500000000000000000"}
		v1Sim := buildSim(t, valueobject.ChainIDEthereum, valueobject.ExchangeFrontierWeighted, "50000000000000", "1100000000000", weights, nil)
		v2Sim := buildSim(t, valueobject.ChainIDEthereum, valueobject.ExchangeFrontierWeighted, "50000000000000", "1100000000000", weights, []string{"1000000", "1000000000000"})

		params := pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: token1, Amount: big.NewInt(100_000_000_000)},
			TokenIn:        token0,
		}

		v1Res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return v1Sim.CalcAmountIn(params)
		})
		require.NoError(t, err)
		require.Equal(t, "5015045135407", v1Res.TokenAmountIn.Amount.String())

		v2Res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
			return v2Sim.CalcAmountIn(params)
		})
		require.NoError(t, err)
		require.Equal(t, "5015045135408", v2Res.TokenAmountIn.Amount.String())
	})

	t.Run("enforces minTokenBalances on indexIn after +1 adjustment", func(t *testing.T) {
		weights := [2]string{"500000000000000000", "500000000000000000"}
		minBals := []string{"1000000000000", "1000000"}
		params := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(10_000_000)},
			TokenOut:      token1,
		}

		// balanceIn = min - 2 -> balanceIn + 1 = min - 1 < min -> reverts on v2, succeeds on v1.
		v1Low := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "999999999998", "50000000000000000000", weights, nil)
		v2Low := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "999999999998", "50000000000000000000", weights, minBals)

		v1Res, err := v1Low.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, "498495030005500", v1Res.TokenAmountOut.Amount.String())

		_, err = v2Low.CalcAmountOut(params)
		require.ErrorIs(t, err, ErrTokenBalanceBelowMin)

		// balanceIn = min - 1 -> balanceIn + 1 == min -> succeeds on v2.
		v2Boundary := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "999999999999", "50000000000000000000", weights, minBals)
		v2Res, err := v2Boundary.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, "498495030004550", v2Res.TokenAmountOut.Amount.String())
	})

	t.Run("enforces minTokenBalances on indexOut for ExactIn and ExactOut", func(t *testing.T) {
		weights := [2]string{"500000000000000000", "500000000000000000"}
		minBals := []string{"1000000", "1000000000000"}
		v1Sim := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "50000000000000", "1100000000000", weights, nil)
		v2Sim := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "50000000000000", "1100000000000", weights, minBals)

		exactInParams := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(10_000_000_000_000)},
			TokenOut:      token1,
		}
		v1Out, err := v1Sim.CalcAmountOut(exactInParams)
		require.NoError(t, err)
		require.Equal(t, "182874770718", v1Out.TokenAmountOut.Amount.String())

		_, err = v2Sim.CalcAmountOut(exactInParams)
		require.ErrorIs(t, err, ErrTokenBalanceBelowMin)

		exactOutParams := pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: token1, Amount: big.NewInt(100_000_000_001)},
			TokenIn:        token0,
		}
		v1In, err := v1Sim.CalcAmountIn(exactOutParams)
		require.NoError(t, err)
		require.Equal(t, "5015045135463", v1In.TokenAmountIn.Amount.String())

		_, err = v2Sim.CalcAmountIn(exactOutParams)
		require.ErrorIs(t, err, ErrTokenBalanceBelowMin)
	})

	t.Run("Frontier GetMetaInfo resolves BatchRouter on Ethereum and Base", func(t *testing.T) {
		weights := [2]string{"500000000000000000", "500000000000000000"}
		for _, chainID := range []valueobject.ChainID{valueobject.ChainIDEthereum, valueobject.ChainIDBase} {
			sim := buildSim(t, chainID, valueobject.ExchangeFrontierWeighted, "50000000000000", "50000000000000", weights, []string{"1000000", "1000000"})
			meta, ok := sim.GetMetaInfo(token0, token1).(shared.PoolMetaInfo)
			require.True(t, ok)
			require.Equal(t, "0x7d08bc34e444848983227e266f61146a7e935a94", meta.ApprovalAddress)
			require.Equal(t, uint64(42000000), meta.BlockNumber)
		}
	})

	t.Run("CloneState and UpdateBalance preserve v2 minimum balance enforcement", func(t *testing.T) {
		weights := [2]string{"500000000000000000", "500000000000000000"}
		sim := buildSim(t, valueobject.ChainIDBase, valueobject.ExchangeFrontierWeighted, "50000000000000", "1100000000000", weights, []string{"1000000", "1000000000000"})
		cloned := sim.CloneState()

		firstParams := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(3_000_000_000_000)},
			TokenOut:      token1,
		}
		firstRes, err := cloned.CalcAmountOut(firstParams)
		require.NoError(t, err)
		cloned.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  firstParams.TokenAmountIn,
			TokenAmountOut: *firstRes.TokenAmountOut,
			SwapInfo:       firstRes.SwapInfo,
		})

		// Second identical swap pushes token1 below 1e12 on cloned sim, while original sim is untouched.
		_, err = cloned.CalcAmountOut(firstParams)
		require.ErrorIs(t, err, ErrTokenBalanceBelowMin)

		origRes, err := sim.CalcAmountOut(firstParams)
		require.NoError(t, err)
		require.Equal(t, firstRes.TokenAmountOut.Amount, origRes.TokenAmountOut.Amount)
	})

	// Live snapshot of Frontier Ethereum 50WBTC-50WETH (WeightedPool v2) at block 26455234:
	// https://etherscan.io/address/0xbe4f51b275f3d4d6a0130988be79b25523549c7a
	// Verified against BatchRouter (0x7D08bc34e444848983227E266F61146A7e935A94) querySwapExactIn/querySwapExactOut.
	t.Run("Live Ethereum Frontier 50WBTC-50WETH snapshot matches on-chain BatchRouter at block 26455234", func(t *testing.T) {
		const (
			wbtc = "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
			weth = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
		)
		poolJSON := `{"address":"0xbe4f51b275f3d4d6a0130988be79b25523549c7a","exchange":"frontier-weighted","type":"balancer-v3-weighted","timestamp":1795164767,"reserves":["112341","39497665194129220"],"tokens":[{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","swappable":true}],"extra":"{\"hook\":{},\"fee\":\"1000000000000000\",\"aggrFee\":\"0\",\"balsE18\":[\"1123410000000000\",\"39497665194129220\"],\"decs\":[\"10000000000\",\"1\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\"],\"buffs\":[null,null],\"normalizedWeights\":[\"500000000000000000\",\"500000000000000000\"],\"minTokenBalances\":[\"10000000000\",\"1000000\"]}","staticExtra":"{\"buffs\":[\"\",\"\"]}","blockNumber":26455234}`

		var p entity.Pool
		require.NoError(t, json.Unmarshal([]byte(poolJSON), &p))

		sim, err := NewPoolSimulator(pool.FactoryParams{
			EntityPool: p,
			ChainID:    valueobject.ChainIDEthereum,
		})
		require.NoError(t, err)

		// 20000 sat WBTC -> WETH: on-chain BatchRouter.querySwapExactIn returns 5964006851359203 (v1 would over-quote by +5 wei).
		out20k, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: wbtc, Amount: big.NewInt(20000)},
				TokenOut:      weth,
			})
		})
		require.NoError(t, err)
		require.Equal(t, "5964006851359203", out20k.TokenAmountOut.Amount.String())

		// 2246 sat WBTC -> WETH exact-in and round-trip exact-out match on-chain BatchRouter to the wei.
		out2246, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wbtc, Amount: big.NewInt(2246)},
			TokenOut:      weth,
		})
		require.NoError(t, err)
		require.Equal(t, "773427887884527", out2246.TokenAmountOut.Amount.String())

		inRes, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: weth, Amount: out2246.TokenAmountOut.Amount},
			TokenIn:        wbtc,
		})
		require.NoError(t, err)
		require.Equal(t, "2246", inRes.TokenAmountIn.Amount.String())

		meta, ok := sim.GetMetaInfo(wbtc, weth).(shared.PoolMetaInfo)
		require.True(t, ok)
		require.Equal(t, "0x7d08bc34e444848983227e266f61146a7e935a94", meta.ApprovalAddress)
		require.Equal(t, uint64(26455234), meta.BlockNumber)
	})
}
