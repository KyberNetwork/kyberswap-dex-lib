package liquiditybookv21

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShiftDivWide: x << 128 over a denominator, rounded down and up, against big.Int, for x wider
// than 128 bits, where the 512-bit division of Uint256x256Math runs.
func TestShiftDivWide(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	widths := []uint{129, 160, 200, 250, 255}
	checked := 0
	for range 5000 {
		x := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), widths[rng.Intn(len(widths))]))
		x.SetBit(x, 128, 1)
		d := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 129+uint(rng.Intn(127))))
		d.Add(d, big.NewInt(1))
		shifted := new(big.Int).Lsh(x, 128)
		down, rem := new(big.Int).QuoRem(shifted, d, new(big.Int))
		if new(big.Int).Rsh(shifted, 256).Cmp(d) >= 0 {
			continue // the quotient does not fit 256 bits: Uint256x256Math reverts
		}
		up := new(big.Int).Set(down)
		if rem.Sign() > 0 {
			up.Add(up, big.NewInt(1))
		}
		ux, ud := uint256.MustFromBig(x), uint256.MustFromBig(d)

		got, err := shiftDivRoundDown(ux, scaleOffset, ud)
		require.NoError(t, err)
		assert.Equal(t, down.String(), got.Dec(), "%s << 128 / %s", x, d)
		got, err = shiftDivRoundUp(ux, scaleOffset, ud)
		require.NoError(t, err)
		if up.BitLen() <= 256 {
			assert.Equal(t, up.String(), got.Dec(), "%s << 128 / %s, rounded up", x, d)
		}
		assert.Equal(t, d.String(), ud.Dec(), "the denominator is left as it was")
		checked++
	}
	require.Greater(t, checked, 1000)
}
