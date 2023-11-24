package composable

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestRegularSwap(t *testing.T) {
	t.Run("1. Should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267407814265248164610048", 10)
		reserve1, _ := new(big.Int).SetString("6999791779383984752", 10)
		reserve2, _ := new(big.Int).SetString("3000000000000000000", 10)

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Tokens: []string{
					"0x00C2A4be503869Fa751c2DbcB7156cc970b5a8dA",
					"0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
					"0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		regularSimulator := &regularSimulator{
			Pool:              pool,
			swapFeePercentage: uint256.NewInt(100000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000052057863883934),
				uint256.NewInt(1000000000000000000),
			},

			bptIndex: 0,
			amp:      uint256.NewInt(1500000),
		}

		poolSimulator := &PoolSimulator{
			Pool:             pool,
			regularSimulator: regularSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
			Amount: big.NewInt(999791779383984752),
		}
		tokenOut := "0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA"

		// expected
		expectedAmountOut := "998507669837625986"

		// calculation
		result, err := poolSimulator.CalcAmountOut(tokenAmountIn, tokenOut)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. Should return OK", func(t *testing.T) {
		// data
		reserve0, _ := new(big.Int).SetString("2596148429267407814265248164610048", 10)
		reserve1, _ := new(big.Int).SetString("6999791779383984752", 10)
		reserve2, _ := new(big.Int).SetString("3000000000000000000", 10)

		pool := poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Tokens: []string{
					"0x00C2A4be503869Fa751c2DbcB7156cc970b5a8dA",
					"0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399",
					"0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
				},
				Reserves: []*big.Int{
					reserve0,
					reserve1,
					reserve2,
				},
			},
		}

		regularSimulator := &regularSimulator{
			Pool:              pool,
			swapFeePercentage: uint256.NewInt(100000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(10000000000),
				uint256.NewInt(10000520578),
				uint256.NewInt(10000000000),
			},

			bptIndex: 0,
			amp:      uint256.NewInt(1500000),
		}

		poolSimulator := &PoolSimulator{
			Pool:             pool,
			regularSimulator: regularSimulator,
		}

		// input
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xF71d0774B214c4cf51E33Eb3d30Ef98132e4DBaA",
			Amount: big.NewInt(23142175917219494),
		}
		tokenOut := "0xD4e7C1F3DA1144c9E2CfD1b015eDA7652b4a4399"

		// expected
		expectedAmountOut := "23155810259460675"

		// calculation
		result, err := poolSimulator.CalcAmountOut(tokenAmountIn, tokenOut)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}

func TestBptSwap(t *testing.T) {
	t.Run("1. Join swap should return OK", func(t *testing.T) {
		
	})

	t.Run("2. Join swap should return OK", func(t *testing.T) {})

	t.Run("3. Join swap should return OK", func(t *testing.T) {})

	t.Run("1. Exit swap should return OK", func(t *testing.T) {})

	t.Run("2. Exit swap should return OK", func(t *testing.T) {})

	t.Run("3. Join swap should return OK", func(t *testing.T) {})
}
