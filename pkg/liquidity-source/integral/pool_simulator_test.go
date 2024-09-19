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

	_xDecimals uint64 = 18
	_yDecimals uint64 = 6

	_reserve0, _ = new(big.Int).SetString("30396549939591301240", 10)
	_reserve1, _ = new(big.Int).SetString("33321339599", 10)

	_swapFee       = uint256.NewInt(500000000000000) // 5 ** 14
	_price, _      = uint256.FromDecimal("2406946062201516769030")
	_invertedPrice = uint256.NewInt(415422975055717)

	_amount0In  = big.NewInt(1000000000000000000)
	_amount1Out = big.NewInt(2405742589)

	_amount1In  = big.NewInt(1000000000)
	_amount0Out = big.NewInt(415215263568189141)

	_token0LimitMin = uint256.NewInt(40000000000000000)   // 0.04 wETH
	_token0LimitMax = uint256.NewInt(8385423175515936014) // ~9 wETH

	_token1LimitMin = uint256.NewInt(100000000)   // 100 USDC
	_token1LimitMax = uint256.NewInt(32366320801) // ~41180 USDC
)

func TestCalcAmountOut(t *testing.T) {
	extraBytes, err := json.Marshal(IntegralPair{
		IsEnabled:      true,
		X_Decimals:     _xDecimals,
		Y_Decimals:     _yDecimals,
		SwapFee:        _swapFee,
		Price:          _price,
		InvertedPrice:  _invertedPrice,
		Token0LimitMin: _token0LimitMin,
		Token0LimitMax: _token0LimitMax,
		Token1LimitMin: _token1LimitMin,
		Token1LimitMax: _token1LimitMax,
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

	// Test for swap limits
	t.Run("3. should return error when amountOut is below limit", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		// Test for tokenOut limit min
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: big.NewInt(1000), // This will result in an amountOut below the limit for token0
			},
			TokenOut: _token0,
		})
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrTR03)
	})

	t.Run("4. should return error when amountOut exceeds limit", func(t *testing.T) {
		simulator, err := NewPoolSimulator(pool)
		require.Nil(t, err)

		// Test for tokenOut limit max
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  _token1,
				Amount: big.NewInt(1000000000000000000), // This will result in an amountOut exceeds the limit for token0
			},
			TokenOut: _token0,
		})
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrTR3A)
	})

	// Test for disabled pool
	t.Run("5. should return error when pool is disabled", func(t *testing.T) {
		disabledExtraBytes, err := json.Marshal(IntegralPair{
			IsEnabled:      false,
			X_Decimals:     _xDecimals,
			Y_Decimals:     _yDecimals,
			SwapFee:        _swapFee,
			Price:          _price,
			InvertedPrice:  _invertedPrice,
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
		Price:          _price,
		Token0LimitMin: _token0LimitMin,
		Token0LimitMax: _token0LimitMax,
		Token1LimitMin: _token1LimitMin,
		Token1LimitMax: _token1LimitMax,
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
