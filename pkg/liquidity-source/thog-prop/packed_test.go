package thogprop

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestPackedFieldsMatchIntegerMask(t *testing.T) {
	// Cover every valid width/offset, including both limb boundaries and the
	// aligned 64/128-bit price and inventory fields used by quote preparation.
	words := []uint256.Int{
		{}, maxUint, {1, 2, 3, 4},
		{0xfedcba9876543210, 0x0123456789abcdef, 0x55aa55aa55aa55aa, 0xaa55aa55aa55aa55},
	}
	for _, word := range words {
		for shift := uint(0); shift < 256; shift++ {
			shifted := new(big.Int).Rsh(word.ToBig(), shift)
			for width := uint(1); width <= 256-shift; width++ {
				mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), width), big.NewInt(1))
				want, overflow := uint256.FromBig(new(big.Int).And(shifted, mask))
				require.False(t, overflow)
				if got := bits(&word, shift, width); got != *want {
					t.Fatalf("word %x shift %d width %d: got %x, want %x", word, shift, width, got, *want)
				}
			}
		}
	}
}

func TestShareConversionsPreserveExactIdentityAndDustBoundaries(t *testing.T) {
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
		m := arithmetic{}
		dust := state.transferDust(&m, 0)
		want := new(big.Int).Quo(new(big.Int).Add(multiplier.ToBig(), new(big.Int).Sub(powers[18].ToBig(), big.NewInt(1))), powers[18].ToBig())
		if want.Cmp(big.NewInt(2)) < 0 {
			want.SetInt64(2)
		}
		require.NoError(t, m.err)
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
	require.True(t, new(state.transferDust(&ordinary, 0)).IsZero())
	require.NoError(t, ordinary.err)
}
