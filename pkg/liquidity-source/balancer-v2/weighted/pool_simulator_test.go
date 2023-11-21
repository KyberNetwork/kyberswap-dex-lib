package weighted

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func Test_CalcAmountOut(t *testing.T) {
	t.Run("1. should return OK", func(t *testing.T) {
		// input
		s := PoolSimulatorV1{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0xac3E018457B222d93114458476f3E3416Abbe38F",
						"0xae78736Cd615f374D3085123A210448E74Fc6393",
						"0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
					},
					Reserves: []*big.Int{
						big.NewInt(331125),
						big.NewInt(320633),
						big.NewInt(348846),
					},
				},
			},

			swapFeePercentage: uint256.NewInt(3000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(333300000000000000),
				uint256.NewInt(333300000000000000),
				uint256.NewInt(333400000000000000),
			},
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xac3E018457B222d93114458476f3E3416Abbe38F",
			Amount: big.NewInt(3311),
		}
		tokenOut := "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"

		// expected
		amountOut := "3442"

		// calculation
		result, err := s.CalcAmountOut(tokenAmountIn, tokenOut)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, amountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. should return OK", func(t *testing.T) {
		// input
		reserve0, _ := new(big.Int).SetString("3360160080014532471350474", 10)
		reserve1, _ := new(big.Int).SetString("1112301324508754708737", 10)

		s := PoolSimulatorV1{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0x5a8F45b943A7E6a4BEA463A98de68940A153c78a",
						"0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223",
					},
					Reserves: []*big.Int{
						reserve0,
						reserve1,
					},
				},
			},

			swapFeePercentage: uint256.NewInt(1000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(800000000000000000),
				uint256.NewInt(200000000000000000),
			},
		}

		amountIn, _ := new(big.Int).SetString("60160080014532471350474", 10)
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x5a8F45b943A7E6a4BEA463A98de68940A153c78a",
			Amount: amountIn,
		}
		tokenOut := "0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223"

		// expected
		amountOut := "76143667376405160244"

		// calculation
		result, err := s.CalcAmountOut(tokenAmountIn, tokenOut)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, amountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return OK", func(t *testing.T) {
		// input
		reserve0, _ := new(big.Int).SetString("3360160080014532471350474", 10)
		reserve1, _ := new(big.Int).SetString("1112301324508754708737", 10)

		s := PoolSimulatorV1{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0x5a8F45b943A7E6a4BEA463A98de68940A153c78a",
						"0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223",
					},
					Reserves: []*big.Int{
						reserve0,
						reserve1,
					},
				},
			},

			swapFeePercentage: uint256.NewInt(1000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(800000000000000000),
				uint256.NewInt(200000000000000000),
			},
		}

		amountIn, _ := new(big.Int).SetString("6016008001453247", 10)
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223",
			Amount: amountIn,
		}
		tokenOut := "0x5a8F45b943A7E6a4BEA463A98de68940A153c78a"

		// expected
		amountOut := "4538893010907736440"

		// calculation
		result, err := s.CalcAmountOut(tokenAmountIn, tokenOut)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, amountOut, result.TokenAmountOut.Amount.String())
	})
}
