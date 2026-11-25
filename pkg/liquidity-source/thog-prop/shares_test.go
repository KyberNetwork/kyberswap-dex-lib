package thogprop

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestShareConversionsAndDustMatchIntegerReference(t *testing.T) {
	state := State{Tokens: []Token{{Address: "0x90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48"}}}
	for _, multiplier := range []uint256.Int{
		u(1), powers[18], u(2_000_000_000_000_000_000),
		u(2_000_000_000_000_000_001), u(3_100_000_000_000_000_001), maxInventory,
	} {
		state.Shares[0].Multiplier = multiplier
		for _, units := range []uint256.Int{{}, u(1), powers[18], maxInventory} {
			m := arithmetic{}
			got := state.amountForUnits(&m, 0, units)
			want := new(big.Int).Quo(new(big.Int).Mul(units.ToBig(), multiplier.ToBig()), powers[18].ToBig())
			require.NoError(t, m.err)
			require.Zero(t, want.Cmp(got.ToBig()))
			got = state.transferUnits(&m, 0, units)
			want.Quo(new(big.Int).Mul(units.ToBig(), powers[18].ToBig()), multiplier.ToBig())
			require.NoError(t, m.err)
			require.Zero(t, want.Cmp(got.ToBig()))
		}
		dust := state.transferDust(0)
		want := new(big.Int).Quo(new(big.Int).Add(multiplier.ToBig(), new(big.Int).Sub(powers[18].ToBig(), big.NewInt(1))), powers[18].ToBig())
		if want.Cmp(big.NewInt(2)) < 0 {
			want.SetInt64(2)
		}
		require.Zero(t, want.Cmp(dust.ToBig()))
	}
	state.Shares[0].Multiplier = powers[18]
	identityMath := arithmetic{}
	require.Equal(t, maxUint, state.amountForUnits(&identityMath, 0, maxUint))
	require.Equal(t, maxUint, state.transferUnits(&identityMath, 0, maxUint))
	require.NoError(t, identityMath.err)
	// A zero multiplier is the pool's marker for an ordinary ERC-20: amounts are
	// units and transfers need no rounding allowance.
	state.Shares[0].Multiplier.Clear()
	ordinary := arithmetic{}
	require.False(t, state.shareBased(0))
	require.Equal(t, u(7), state.amountForUnits(&ordinary, 0, u(7)))
	require.Equal(t, u(7), state.transferUnits(&ordinary, 0, u(7)))
	require.True(t, new(state.transferDust(0)).IsZero())
	require.NoError(t, ordinary.err)
}
