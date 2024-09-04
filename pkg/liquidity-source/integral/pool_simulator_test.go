package integral

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_token0            = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	_token1            = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	_reserve0, _       = new(big.Int).SetString("191264141949", 10)
	_reserve1, _       = new(big.Int).SetString("236717826701867952033", 10)
	_swapFee           = uint256.NewInt(100000000000000)
	_decimalsConverter = big.NewInt(1000000)
	_averagePrice      = uint256.NewInt(421279503935549)

	_amount0In  = big.NewInt(10000000)
	_amount1Out = big.NewInt(4212373759851553)

	_amount1In  = big.NewInt(1000000000000000)
	_amount0Out = big.NewInt(2373482)
)

func TestCalcAmountOut(t *testing.T) {
	extraBytes, err := json.Marshal(IntegralPair{
		SwapFee:           _swapFee,
		DecimalsConverter: _decimalsConverter,
		AveragePrice:      _averagePrice,
	})
	require.Nil(t, err)

	pool := entity.Pool{
		Address: "",
		Reserves: entity.PoolReserves{
			_reserve0.String(),
			_reserve1.String(),
		},
		Tokens: []*entity.PoolToken{
			{Address: _token0},
			{Address: _token1},
		},
		Extra: string(extraBytes),
	}

	t.Run("1. should return OK for token0 to token1 swap", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
			return simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  _token0,
					Amount: _amount0In,
				},
				TokenOut: _token1,
			})
		})

		require.Nil(t, err)
		assert.Equal(t, _amount1Out, result.TokenAmountOut.Amount)
	})

	t.Run("2. should return OK for token1 to token0 swap", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: _amount1In,
			},
			TokenOut: _token0,
		})
		require.Nil(t, err)
		assert.Equal(t, _amount0Out, result.TokenAmountOut.Amount)
	})

	t.Run("3. should return error when not enough liquidity", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		// Test for insufficient liquidity
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token0,
				Amount: new(big.Int).Add(_reserve1, _amount1In),
			},
			TokenOut: _token1,
		})
		require.NotNil(t, err)
		require.Nil(t, result)
	})

	t.Run("4. should return error for invalid token", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		// Test for insufficient liquidity
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "xxx", // invalid tokenIn
				Amount: new(big.Int).Add(_reserve1, _amount1In),
			},
			TokenOut: _token1,
		})
		require.NotNil(t, err)
		require.Nil(t, result)
	})
}

func TestUpdateBalance(t *testing.T) {
	extra := IntegralPair{
		SwapFee:           _swapFee,
		DecimalsConverter: _decimalsConverter,
		AveragePrice:      _averagePrice,
	}
	extraJson, _ := json.Marshal(extra)

	token0 := entity.PoolToken{
		Address:   _token0,
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   _token1,
		Swappable: true,
	}

	pool := entity.Pool{
		Reserves: entity.PoolReserves{_reserve0.String(), _reserve1.String()},
		Tokens:   []*entity.PoolToken{&token0, &token1},
		Extra:    string(extraJson),
	}

	poolSimulator, err := NewPoolSimulator(pool)
	require.Nil(t, err)

	tokenAmountIn := poolpkg.TokenAmount{
		Token:  _token0,
		Amount: _amount0In,
	}

	result, _ := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
		return poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      _token1,
			Limit:         nil,
		})
	})

	poolSimulator.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	expectedReserve0 := new(big.Int).Add(_reserve0, _amount0In)
	expectedReserve1 := new(big.Int).Sub(_reserve1, result.TokenAmountOut.Amount)

	expectedFee := new(big.Int).Div(new(big.Int).Mul(_amount0In, ToInt256(_swapFee)), ToInt256(precison))

	assert.Equal(t, new(big.Int).Sub(expectedReserve0, expectedFee), poolSimulator.Info.Reserves[0])
	assert.Equal(t, expectedReserve1, poolSimulator.Info.Reserves[1])
}
