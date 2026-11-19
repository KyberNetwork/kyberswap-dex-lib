package everlongflamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestLegacyGrid(t *testing.T) {
	g := legacyGrid(8)
	require.Len(t, g, 30)
	require.Equal(t, "100000", g[0].String()) // 10^(8-3)
}

// A legacy ladder carries one path-free cell and a second-fill charge, or the simulator refuses it.
func TestLegacyLadderCurve(t *testing.T) {
	l := Ladder{In: 0, Out: 1, Cells: []Cell{{Gas: legacyGasSell}}, Second: legacyGasSell,
		Points: []Point{{uint256.NewInt(1000), uint256.NewInt(600)}, {uint256.NewInt(2000), uint256.NewInt(1200)}}}
	c, err := newCurve(l)
	require.NoError(t, err)
	var q, used, out uint256.Int
	q.SetUint64(2000)
	_, err = c.quote(&q, 0, &used, &out)
	require.NoError(t, err)
	require.Equal(t, uint64(1200), out.Uint64())
	require.Equal(t, legacyGasSell, c.charge(&out, false))
}
