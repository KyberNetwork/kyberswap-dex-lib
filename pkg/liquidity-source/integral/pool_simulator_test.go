package integral

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	_token0 = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48" // USDC
	_token1 = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2" // wETH

	_xDecimals uint8 = 18
	_yDecimals uint8 = 6

	_reserve0, _ = new(big.Int).SetString("30396549939591301240", 10)
	_reserve1, _ = new(big.Int).SetString("33321339599", 10)

	_swapFee       = uint256.NewInt(500000000000000) // 5 ** 14
	_price, _      = uint256.FromDecimal("2406946062201516769030")
	_invertedPrice = uint256.NewInt(415422975055717)

	_amount0In  = big.NewInt(1000000000000000000)
	_amount1Out = big.NewInt(2405742589)

	_amount1In  = big.NewInt(1000000000)
	_amount0Out = big.NewInt(415215263568189141)

	_token0LimitMin = uint256.NewInt(40000000000000000) // 0.04 wETH
	_token1LimitMin = uint256.NewInt(100000000)         // 100 USDC

	_token0LimitMaxMultiplier = uint256.NewInt(950000000000000000)
	_token1LimitMaxMultiplier = uint256.NewInt(950000000000000000) // 0.95
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()

	extraBytes, err := json.Marshal(Extra{
		IsEnabled:                true,
		SwapFee:                  _swapFee,
		Price:                    _price,
		InvertedPrice:            _invertedPrice,
		Token0LimitMin:           _token0LimitMin,
		Token1LimitMin:           _token1LimitMin,
		Token0LimitMaxMultiplier: _token0LimitMaxMultiplier,
		Token1LimitMaxMultiplier: _token1LimitMaxMultiplier,
	})
	require.Nil(t, err)

	pool := entity.Pool{
		Address: "",
		Reserves: entity.PoolReserves{
			_reserve0.String(),
			_reserve1.String(),
		},
		Tokens: []*entity.PoolToken{
			{Address: _token0, Decimals: 18, Swappable: true},
			{Address: _token1, Decimals: 6, Swappable: true},
		},
		Extra: string(extraBytes),
	}

	sim, err := NewPoolSimulator(pool)
	require.Nil(t, err)
	limit := swaplimit.NewInventory(DexTypeIntegral, sim.CalculateLimit())

	t.Run("1. should return OK for token0 to token1 swap", func(t *testing.T) {
		cloned := sim.CloneState()
		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return cloned.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  _token0,
					Amount: _amount0In,
				},
				TokenOut: _token1,
				Limit:    limit,
			})
		})

		require.Nil(t, err)
		assert.Equal(t, _amount1Out, result.TokenAmountOut.Amount)
	})

	t.Run("2. should return OK for token1 to token0 swap", func(t *testing.T) {
		cloned := sim.CloneState()
		result, err := cloned.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: _amount1In,
			},
			TokenOut: _token0,
			Limit:    limit,
		})
		require.Nil(t, err)
		assert.Equal(t, _amount0Out, result.TokenAmountOut.Amount)
	})

	t.Run("3. should return error when amountOut is below limit", func(t *testing.T) {
		cloned := sim.CloneState()
		result, err := cloned.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: big.NewInt(1000), // This will result in an amountOut below the limit for token0
			},
			TokenOut: _token0,
			Limit:    limit,
		})
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrTR03)
	})

	t.Run("4. should return error when amountOut exceeds limit", func(t *testing.T) {
		cloned := sim.CloneState()
		result, err := cloned.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: big.NewInt(1000000000000000000), // This will result in an amountOut exceeds the limit for token0
			},
			TokenOut: _token0,
			Limit:    limit,
		})
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrTR3A)
	})

	// Test for disabled pool
	t.Run("5. should return error when pool is disabled", func(t *testing.T) {
		disabledExtraBytes, err := json.Marshal(Extra{
			IsEnabled:                false,
			SwapFee:                  _swapFee,
			Price:                    _price,
			InvertedPrice:            _invertedPrice,
			Token0LimitMin:           _token0LimitMin,
			Token1LimitMin:           _token1LimitMin,
			Token0LimitMaxMultiplier: _token0LimitMaxMultiplier,
			Token1LimitMaxMultiplier: _token1LimitMaxMultiplier,
		})
		require.Nil(t, err)

		disabledPool := entity.Pool{
			Address: "",
			Reserves: entity.PoolReserves{
				_reserve0.String(),
				_reserve1.String(),
			},
			Tokens: []*entity.PoolToken{
				{Address: _token0},
				{Address: _token1},
			},
			Extra: string(disabledExtraBytes),
		}

		p, err := NewPoolSimulator(disabledPool)
		require.Nil(t, err)

		result, err := p.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token0,
				Amount: _amount0In,
			},
			TokenOut: _token1,
			Limit:    limit,
		})
		require.NotNil(t, err)
		require.Nil(t, result)
	})
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	extraBytes, err := json.Marshal(Extra{
		IsEnabled:                true,
		SwapFee:                  _swapFee,
		Price:                    _price,
		Token0LimitMin:           _token0LimitMin,
		Token1LimitMin:           _token1LimitMin,
		Token0LimitMaxMultiplier: _token0LimitMaxMultiplier,
		Token1LimitMaxMultiplier: _token1LimitMaxMultiplier,
	})
	require.Nil(t, err)

	token0 := entity.PoolToken{
		Address:   _token0,
		Decimals:  _xDecimals,
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   _token1,
		Decimals:  _yDecimals,
		Swappable: true,
	}

	pool := entity.Pool{
		Reserves: entity.PoolReserves{_reserve0.String(), _reserve1.String()},
		Tokens:   []*entity.PoolToken{&token0, &token1},
		Extra:    string(extraBytes),
	}

	poolSimulator, err := NewPoolSimulator(pool)
	require.Nil(t, err)
	limit := swaplimit.NewInventory(DexTypeIntegral, poolSimulator.CalculateLimit())
	require.NotNil(t, limit)

	tokenAmountIn := poolpkg.TokenAmount{
		Token:  _token0,
		Amount: _amount0In,
	}

	result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      _token1,
			Limit:         limit,
		})
	})

	require.Nil(t, err)

	poolSimulator.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
		SwapLimit:      limit,
	})

	expectedReserve0 := new(big.Int).Add(_reserve0, _amount0In)
	expectedReserve1 := new(big.Int).Sub(_reserve1, result.TokenAmountOut.Amount)

	assert.Equal(t, expectedReserve0, limit.GetLimit(_token0))
	assert.Equal(t, expectedReserve1, limit.GetLimit(_token1))
}

type UpdateBalanceTestSuite struct {
	suite.Suite

	pools map[string]string
	sims  map[string]*PoolSimulator

	limit *swaplimit.Inventory
}

func (ts *UpdateBalanceTestSuite) SetupSuite() {
	ts.pools = map[string]string{
		"WETH-USDT": `{"address":"0x048f0e7ea2cfd522a4a058d1b1bdd574a0486c46","swapFee":0.00055,"exchange":"integral","type":"integral","timestamp":1753734990,"reserves":["19597574281727075672","27200982862"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"relayerAddress\":\"0xd17b3c9784510E33cD5B87b490E79253BcD81e2E\",\"isEnabled\":true,\"price\":\"3791811598071743552476\",\"invertedPrice\":\"264484194114580\",\"swapFee\":\"550000000000000\",\"t0LiMi\":\"1200000000000000000\",\"t0LiMa\":\"18617695567640721888\",\"t1LiMi\":\"5000000000\",\"t1LiMa\":\"25840933718\",\"t0LiMaMu\":\"950000000000000000\",\"t1LiMaMu\":\"950000000000000000\"}","blockNumber":23020064}`,
		"WBTC-WETH": `{"address":"0x37f6df71b40c50b2038329cabf5fda3682df1ebf","swapFee":0.0005,"exchange":"integral","type":"integral","timestamp":1753734990,"reserves":["140058859","19597574281727075672"],"tokens":[{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"relayerAddress\":\"0xd17b3c9784510E33cD5B87b490E79253BcD81e2E\",\"isEnabled\":true,\"price\":\"31146077957603836264\",\"invertedPrice\":\"31146077957603836264\",\"swapFee\":\"500000000000000\",\"t0LiMi\":\"7000000\",\"t0LiMa\":\"25840933718\",\"t1LiMi\":\"1200000000000000000\",\"t1LiMa\":\"18617695567640721888\",\"t0LiMaMu\":\"950000000000000000\",\"t1LiMaMu\":\"950000000000000000\"}","blockNumber":23020064}`,
	}

	ts.sims = map[string]*PoolSimulator{}
	for k, p := range ts.pools {
		var ep entity.Pool
		err := json.Unmarshal([]byte(p), &ep)
		ts.Require().Nil(err)

		sim, err := NewPoolSimulator(ep)
		ts.Require().Nil(err)
		ts.Require().NotNil(sim)

		ts.sims[k] = sim
	}

	ts.limit = swaplimit.NewInventory(DexTypeIntegral, map[string]*big.Int{
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": bignum.NewBig("19597574281727075672"),
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": bignum.NewBig("140058859"),
		"0xdac17f958d2ee523a2206206994597c13d831ec7": bignum.NewBig("27200982862"),
	})
}

func (ts *UpdateBalanceTestSuite) TestUpdateBalance() {
	ts.T().Parallel()

	swaps := []struct {
		name     string
		pool     string
		tokenIn  string
		tokenOut string
		amountIn string

		expectedAmountOut string
		expectedError     error
	}{
		{
			name:              "USDT -> WETH",
			pool:              "WETH-USDT",
			tokenIn:           "0xdac17f958d2ee523a2206206994597c13d831ec7",
			tokenOut:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:          "59000000000",
			expectedAmountOut: "15595984940661201879",
		},
		{
			name:          "WBTC -> WETH",
			pool:          "WBTC-WETH",
			tokenIn:       "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			tokenOut:      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn:      "50925483",
			expectedError: ErrTR3A,
		},
	}

	ts.T().Run("UpdateBalance", func(t *testing.T) {
		for _, swap := range swaps {
			sim := ts.sims[swap.pool]
			require.NotNil(t, sim, "Pool simulator for %s not found", swap.pool)

			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  swap.tokenIn,
					Amount: bignum.NewBig(swap.amountIn),
				},
				TokenOut: swap.tokenOut,
				Limit:    ts.limit,
			})

			if swap.expectedError == nil {
				require.NotNil(t, res)
				require.Equal(t, swap.expectedAmountOut, res.TokenAmountOut.Amount.String())
				sim.UpdateBalance(poolpkg.UpdateBalanceParams{
					TokenAmountIn: poolpkg.TokenAmount{
						Token:  swap.tokenIn,
						Amount: bignum.NewBig(swap.amountIn),
					},
					TokenAmountOut: *res.TokenAmountOut,
					SwapInfo:       res.SwapInfo,
					SwapLimit:      ts.limit,
				})
				require.Equal(t, swap.expectedAmountOut, res.TokenAmountOut.Amount.String())
			} else {
				require.ErrorContains(t, err, swap.expectedError.Error())
			}
		}
	})
}

func TestPoolSimulatorTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdateBalanceTestSuite))
}
