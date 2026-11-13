package fraxswap

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestCloneStateIsolatesUpdateBalance(t *testing.T) {
	t.Parallel()
	sim := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Tokens:   []string{"a", "b"},
			Reserves: []*big.Int{big.NewInt(20), big.NewInt(20)},
		}},
		Fee: big.NewInt(9997),
	}
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "a", Amount: big.NewInt(20)},
		TokenOut:      "b",
	}
	result, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  params.TokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
	})

	assert.Equal(t, "20", sim.GetReserves()[0].String(), "UpdateBalance on clone changed original reserve0")
	assert.Equal(t, "20", sim.GetReserves()[1].String(), "UpdateBalance on clone changed original reserve1")
	assert.Equal(t, "39", clone.GetReserves()[0].String(), "clone reserve0 should grow by fee-adjusted amountIn")
	assert.Equal(t, "11", clone.GetReserves()[1].String(), "clone reserve1 should drop by amountOut")
}
