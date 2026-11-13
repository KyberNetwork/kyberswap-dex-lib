package smardex

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestCloneStateIsolatesUpdateBalance(t *testing.T) {
	t.Parallel()
	extra := SmardexPair{
		PairFee: PairFee{
			FeesLP:   feesLP,
			FeesPool: feesPool,
			FeesBase: uint256.MustFromBig(FEES_BASE),
		},
		FictiveReserve: FictiveReserve{FictiveReserve0: resFicT0, FictiveReserve1: resFicT1},
		PriceAverage: PriceAverage{
			PriceAverage0:             priceAvT0,
			PriceAverage1:             priceAvT1,
			PriceAverageLastTimestamp: uint256.NewInt(uint64(TIMESTAMP_JAN_2020)),
		},
		FeeToAmount: FeeToAmount{Fees0: uint256.NewInt(0), Fees1: uint256.NewInt(0)},
	}
	extraJson, _ := json.Marshal(extra)

	sim, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{resT0.String(), resT1.String()},
		Tokens: []*entity.PoolToken{
			{Address: "token0", Swappable: true},
			{Address: "token1", Swappable: true},
		},
		Extra: string(extraJson),
	})
	require.NoError(t, err)

	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: "token0", Amount: amountInT0.ToBig()},
		TokenOut:      "token1",
	}
	result, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	origReserves := sim.GetReserves()
	origRes0, origRes1 := origReserves[0].String(), origReserves[1].String()
	origFic0, origFic1 := sim.FictiveReserve.FictiveReserve0.String(), sim.FictiveReserve.FictiveReserve1.String()
	origPriceAv0 := sim.PriceAverage.PriceAverage0.String()
	origFee0 := sim.FeeToAmount.Fees0.String()

	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  params.TokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	assert.Equal(t, origRes0, sim.GetReserves()[0].String(), "UpdateBalance on clone changed original reserve0")
	assert.Equal(t, origRes1, sim.GetReserves()[1].String(), "UpdateBalance on clone changed original reserve1")
	assert.Equal(t, origFic0, sim.FictiveReserve.FictiveReserve0.String())
	assert.Equal(t, origFic1, sim.FictiveReserve.FictiveReserve1.String())
	assert.Equal(t, origPriceAv0, sim.PriceAverage.PriceAverage0.String())
	assert.Equal(t, origFee0, sim.FeeToAmount.Fees0.String())

	assert.NotEqual(t, origRes0, clone.GetReserves()[0].String(), "clone reserve0 should change after UpdateBalance")
	assert.NotEqual(t, origRes1, clone.GetReserves()[1].String(), "clone reserve1 should change after UpdateBalance")
}
