package btr_test

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/btr"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The router ships simulators between services via msgpack, so a decoded
// simulator must quote the same as the original, including consumed input.
func TestMsgpackRoundTrip(t *testing.T) {
	const a, b = "0x754704bc059f8c67012fed69bc8a327a5aafb603", "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	extra, err := json.Marshal(btr.Extra{
		Flags: []uint16{btr.SwapEnabledBit, btr.SwapEnabledBit},
		Ladders: []btr.Ladder{{From: 0, To: 1, Points: []btr.Point{
			{In: *uint256.NewInt(1000), Out: *uint256.NewInt(990)},
			{In: *uint256.NewInt(2000), Out: *uint256.NewInt(1900)},
		}}},
	})
	require.NoError(t, err)
	sim, err := btr.NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: btr.CreditPoolCore, Exchange: btr.DexType, Type: btr.DexType, Extra: string(extra),
		Tokens:   []*entity.PoolToken{{Address: a}, {Address: b}},
		Reserves: entity.PoolReserves{"5000", "5000"},
	}})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: a, Amount: big.NewInt(1000)},
		TokenAmountOut: pool.TokenAmount{Token: b, Amount: big.NewInt(990)},
	})

	enc, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{"p": sim})
	require.NoError(t, err)
	dec, err := msgpack.DecodePoolSimulatorsMap(enc)
	require.NoError(t, err)

	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: a, Amount: big.NewInt(500)}, TokenOut: b}
	want, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	got, err := dec["p"].CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
	require.Equal(t, "455", got.TokenAmountOut.Amount.String(), "500 more after 1000 used: 1900-990 -> slope 0.91")
}
