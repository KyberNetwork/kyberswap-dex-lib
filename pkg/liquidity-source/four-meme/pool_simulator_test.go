package fourmeme

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func createTestPoolSimulator() *PoolSimulator {
	extra := Extra{
		GradThreshold: big.NewInt(0),
		BuyTax:        big.NewInt(5),  // 5% buy tax
		SellTax:       big.NewInt(10), // 10% sell tax
		KLast:         big.NewInt(1500000),
		ReserveA:      big.NewInt(1500),
		ReserveB:      big.NewInt(1000),
	}
	extraBytes, _ := json.Marshal(extra)

	staticExtra := StaticExtra{
		BondingAddress: "0xF66DeA7b3e897cD44A5a231c61B6B4423d613259",
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	entityPool := entity.Pool{
		Address:     "0xc321c3a7f730608b51e4747b72aeb18e0a3d32c4",
		Exchange:    string(valueobject.ExchangeVirtualFun),
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: "TokenA"}, {Address: "TokenB"}},
		Reserves:    []string{"1000", "1000"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	poolSimulator, err := NewPoolSimulator(entityPool)
	if err != nil {
		panic(err)
	}

	return poolSimulator
}

// func TestNewPoolSimulator(t *testing.T) {
// 	poolSimulator := createTestPoolSimulator()

// 	assert.NotNil(t, poolSimulator)
// 	assert.Equal(t, "0xc321c3a7f730608b51e4747b72aeb18e0a3d32c4", poolSimulator.Pool.Info.Address)
// 	assert.Equal(t, uint256.NewInt(5), poolSimulator.buyTax)
// 	assert.Equal(t, uint256.NewInt(10), poolSimulator.sellTax)
// 	assert.Equal(t, uint256.NewInt(1500000), poolSimulator.kLast)
// 	assert.Equal(t, uint256.NewInt(1500), poolSimulator.reserveA)
// 	assert.Equal(t, uint256.NewInt(1000), poolSimulator.reserveB)
// 	assert.Equal(t, "0xF66DeA7b3e897cD44A5a231c61B6B4423d613259", poolSimulator.bondingAddress)

// }

func TestCalcAmountOut_SellToken(t *testing.T) {
	poolSimulator := createTestPoolSimulator()

	amountIn, _ := uint256.FromBig(big.NewInt(187))
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "TokenA",
			Amount: amountIn.ToBig(),
		},
		TokenOut: "TokenB",
	}

	result, err := poolSimulator.CalcAmountOut(params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TokenB", result.TokenAmountOut.Token)
	assert.Equal(t, big.NewInt(100), result.TokenAmountOut.Amount)
}

func TestCalcAmountOut_BuyToken(t *testing.T) {
	poolSimulator := createTestPoolSimulator()

	amountIn, _ := uint256.FromBig(big.NewInt(100))
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "TokenB",
			Amount: amountIn.ToBig(),
		},
		TokenOut: "TokenA",
	}

	result, err := poolSimulator.CalcAmountOut(params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TokenA", result.TokenAmountOut.Token)
	assert.Equal(t, big.NewInt(131), result.TokenAmountOut.Amount)
}

func TestCalcAmountIn_SellExactOut(t *testing.T) {
	poolSimulator := createTestPoolSimulator()

	amountOut, _ := uint256.FromBig(big.NewInt(100))
	params := poolpkg.CalcAmountInParams{
		TokenAmountOut: poolpkg.TokenAmount{
			Token:  "TokenB",
			Amount: amountOut.ToBig(),
		},
		TokenIn: "TokenA",
	}

	result, err := poolSimulator.CalcAmountIn(params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TokenA", result.TokenAmountIn.Token)
	assert.Equal(t, big.NewInt(187), result.TokenAmountIn.Amount)

	// Swap back the calculated input amount and verify the output
	_params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "TokenA",
			Amount: result.TokenAmountIn.Amount,
		},
		TokenOut: "TokenB",
	}

	_result, err := poolSimulator.CalcAmountOut(_params)
	require.NoError(t, err)
	assert.NotNil(t, _result)
	assert.Equal(t, "TokenB", _result.TokenAmountOut.Token)
	assert.Equal(t, amountOut.ToBig(), _result.TokenAmountOut.Amount)
}

func TestCalcAmountIn_BuyExactOut(t *testing.T) {
	poolSimulator := createTestPoolSimulator()

	amountOut, _ := uint256.FromBig(big.NewInt(100))
	params := poolpkg.CalcAmountInParams{
		TokenAmountOut: poolpkg.TokenAmount{
			Token:  "TokenA",
			Amount: amountOut.ToBig(),
		},
		TokenIn: "TokenB",
	}

	result, err := poolSimulator.CalcAmountIn(params)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TokenB", result.TokenAmountIn.Token)
	assert.Equal(t, big.NewInt(74), result.TokenAmountIn.Amount)

	// Swap back the calculated input amount and verify the output
	_params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "TokenB",
			Amount: result.TokenAmountIn.Amount,
		},
		TokenOut: "TokenA",
	}

	_result, err := poolSimulator.CalcAmountOut(_params)
	require.NoError(t, err)
	assert.NotNil(t, _result)
	assert.Equal(t, "TokenA", _result.TokenAmountOut.Token)
	assert.Equal(t, amountOut.ToBig(), _result.TokenAmountOut.Amount)
}

// func TestUpdateBalance(t *testing.T) {
// 	poolSimulator := createTestPoolSimulator()

// 	newReserveA := uint256.NewInt(1200)
// 	newReserveB := uint256.NewInt(800)
// 	swapInfo := SwapInfo{
// 		IsBuy:       true,
// 		NewReserveA: newReserveA,
// 		NewReserveB: newReserveB,
// 		NewBalanceA: newReserveA,
// 		NewBalanceB: newReserveB,
// 	}

// 	params := poolpkg.UpdateBalanceParams{
// 		SwapInfo: swapInfo,
// 	}

// 	poolSimulator.UpdateBalance(params)

// 	assert.Equal(t, newReserveA.ToBig(), poolSimulator.Pool.Info.Reserves[0])
// 	assert.Equal(t, newReserveB.ToBig(), poolSimulator.Pool.Info.Reserves[1])
// 	assert.Equal(t, newReserveA, poolSimulator.reserveA)
// 	assert.Equal(t, newReserveB, poolSimulator.reserveB)
// }

func TestErrorCases(t *testing.T) {
	poolSimulator := createTestPoolSimulator()

	t.Run("Insufficient Input Amount", func(t *testing.T) {
		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "TokenA",
				Amount: big.NewInt(0),
			},
			TokenOut: "TokenB",
		}

		_, err := poolSimulator.CalcAmountOut(params)
		assert.Error(t, err)
		// assert.Equal(t, ErrInsufficientInputAmount, err)
	})

	t.Run("Insufficient Output Amount", func(t *testing.T) {
		params := poolpkg.CalcAmountInParams{
			TokenAmountOut: poolpkg.TokenAmount{
				Token:  "TokenA",
				Amount: big.NewInt(10000), // exceeds reserveOut
			},
			TokenIn: "TokenB",
		}

		_, err := poolSimulator.CalcAmountIn(params)
		assert.Error(t, err)
		// assert.Equal(t, ErrInsufficientOutputAmount, err)
	})
}

func TestGetMetaInfo(t *testing.T) {
	poolSimulator := createTestPoolSimulator()

	metaInfo := poolSimulator.GetMetaInfo("", "")
	poolMeta, ok := metaInfo.(PoolMeta)

	assert.True(t, ok)
	assert.Equal(t, uint64(0), poolMeta.BlockNumber)
}
