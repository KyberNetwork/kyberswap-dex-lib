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
	_token0 = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48" // USDC
	_token1 = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2" // wETH

	_xDecimals uint64 = 6
	_yDecimals uint64 = 18

	_reserve0, _ = new(big.Int).SetString("427606417957", 10)
	_reserve1, _ = new(big.Int).SetString("134129304160258568649", 10)

	_swapFee      = uint256.NewInt(550000000000000) // 10 ** 14
	_averagePrice = uint256.NewInt(436677406974646)
	_spotPrice    = uint256.NewInt(436776207402818)

	_amount0In  = big.NewInt(10000000000)
	_amount1Out = big.NewInt(4364372344008099447)

	_amount1In, _  = new(big.Int).SetString("10000000000000000000", 10)
	_amount0Out, _ = new(big.Int).SetString("22882427729", 10)

	_token0LimitMin = uint256.NewInt(5000000000)          // 5000 USDC
	_token1LimitMin = uint256.NewInt(1200000000000000000) // 1.2 wETH
)

func TestCalcAmountOut(t *testing.T) {
	extraBytes, err := json.Marshal(IntegralPair{
		IsEnabled:      true,
		X_Decimals:     _xDecimals,
		Y_Decimals:     _yDecimals,
		SwapFee:        _swapFee,
		SpotPrice:      _spotPrice,
		AveragePrice:   _averagePrice,
		Token0LimitMin: _token0LimitMin,
		Token1LimitMin: _token1LimitMin,
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

	// Test for swap limits
	t.Run("4. should return error when amountOut is below limits", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		// Test for token0 limit
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: big.NewInt(1000), // This will result in an amountOut below the limit for token0
			},
			TokenOut: _token0,
		})
		require.NotNil(t, err)
		require.Nil(t, result)
	})

	// Test for disabled pool
	t.Run("5. should return error when pool is disabled", func(t *testing.T) {
		disabledExtraBytes, err := json.Marshal(IntegralPair{
			IsEnabled:      false,
			X_Decimals:     _xDecimals,
			Y_Decimals:     _yDecimals,
			SwapFee:        _swapFee,
			SpotPrice:      _spotPrice,
			AveragePrice:   _averagePrice,
			Token0LimitMin: _token0LimitMin,
			Token1LimitMin: _token1LimitMin,
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

		simulator, err := NewPoolSimulator(disabledPool)
		require.Nil(t, err)

		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token0,
				Amount: _amount0In,
			},
			TokenOut: _token1,
		})
		require.NotNil(t, err)
		require.Nil(t, result)
	})
}

func TestUpdateBalance(t *testing.T) {
	extraBytes, err := json.Marshal(IntegralPair{
		IsEnabled:      true,
		X_Decimals:     _xDecimals,
		Y_Decimals:     _yDecimals,
		SwapFee:        _swapFee,
		SpotPrice:      _spotPrice,
		AveragePrice:   _averagePrice,
		Token0LimitMin: _token0LimitMin,
		Token1LimitMin: _token1LimitMin,
	})
	require.Nil(t, err)

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
		Extra:    string(extraBytes),
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

	assert.Equal(t, expectedReserve0, poolSimulator.Info.Reserves[0])
	assert.Equal(t, expectedReserve1, poolSimulator.Info.Reserves[1])
}
