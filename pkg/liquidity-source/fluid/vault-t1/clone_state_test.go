package vaultT1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCloneStateIsolatesUpdateBalance(t *testing.T) {
	t.Parallel()
	const (
		tokenIn  = "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"
		tokenOut = "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"
	)
	sim := &PoolSimulator{
		Info: poolpkg.PoolInfo{
			Address:     "0x40D9b8417E6E1DcD358f04E3328bCEd061018A82",
			Exchange:    "fluid-vault-t1",
			Type:        "fluid-vault-t1",
			Tokens:      []string{tokenIn, tokenOut},
			Reserves:    []*big.Int{bignumber.NewBig("86232802856618560"), bignumber.NewBig("97976286699627227")},
			BlockNumber: 20812089,
			SwapFee:     big.NewInt(0),
		},
		Ratio: bignumber.NewBig("1136183487651849280183370224"),
	}
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Amount: bignumber.NewBig("10000000000000000"), Token: tokenIn},
		TokenOut:      tokenOut,
	}
	result, err := sim.CalcAmountOut(params)
	assert.NoError(t, err)

	origRes0 := sim.GetReserves()[0].String()
	origRes1 := sim.GetReserves()[1].String()

	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  params.TokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	assert.Equal(t, origRes0, sim.GetReserves()[0].String(), "UpdateBalance on clone changed original reserve0")
	assert.Equal(t, origRes1, sim.GetReserves()[1].String(), "UpdateBalance on clone changed original reserve1")
	assert.NotEqual(t, origRes0, clone.GetReserves()[0].String(), "clone reserve0 should change after UpdateBalance")
	assert.NotEqual(t, origRes1, clone.GetReserves()[1].String(), "clone reserve1 should change after UpdateBalance")
}
