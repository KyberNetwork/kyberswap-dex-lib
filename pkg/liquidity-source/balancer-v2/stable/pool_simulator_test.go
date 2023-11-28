package stable

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. should return error balance didnt converge", func(t *testing.T) {
		reserves := make([]*big.Int, 3)
		reserves[0], _ = new(big.Int).SetString("9999991000000000000", 10)
		reserves[1], _ = new(big.Int).SetString("99999910000000000056", 10)
		reserves[2], _ = new(big.Int).SetString("8897791020011100123456", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(50000000000000),
			amp:               uint256.NewInt(5000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1), uint256.NewInt(100)},

			poolType:        poolTypeStable,
			poolTypeVersion: 1,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: new(big.Int).SetUint64(99999910000000),
		}
		tokenOut := "0x6b175474e89094c44da98b954eedeac495271d0f"
		_, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.ErrorIs(t, err, math.ErrStableGetBalanceDidntConverge)
	})

	t.Run("2. should return OK", func(t *testing.T) {
		// input
		reserves := make([]*big.Int, 3)
		reserves[0], _ = new(big.Int).SetString("9999991000000000000000", 10)
		reserves[1], _ = new(big.Int).SetString("9999991000000000005613", 10)
		reserves[2], _ = new(big.Int).SetString("13288977911102200123456", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(50000000000000),
			amp:               uint256.NewInt(1390000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1), uint256.NewInt(100)},

			poolType:        poolTypeStable,
			poolTypeVersion: 1,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x6b175474e89094c44da98b954eedeac495271d0f",
			Amount: new(big.Int).SetUint64(12000000000000000000),
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		// expected
		expected := "1000000000000000000"

		// actual
		result, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return OK", func(t *testing.T) {
		// input
		reserves := make([]*big.Int, 3)
		reserves[0], _ = new(big.Int).SetString("9999991000000000013314124321", 10)
		reserves[1], _ = new(big.Int).SetString("9999991000000123120010005613", 10)
		reserves[2], _ = new(big.Int).SetString("1328897131447911102200123456", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(53332221119995),
			amp:               uint256.NewInt(1390000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(100)},

			poolType:        poolTypeStable,
			poolTypeVersion: 1,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: new(big.Int).SetUint64(12111222333444555666),
		}
		tokenOut := "0x6b175474e89094c44da98b954eedeac495271d0f"

		// expected
		expected := "590000000000000000"

		// actual
		result, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.TokenAmountOut.Amount.String())
	})
}
