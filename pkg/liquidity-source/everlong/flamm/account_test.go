package everlongflamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type mmMulDivRow struct {
	Header bool        `json:"header"`
	X      uint256.Int `json:"x"`
	Y      uint256.Int `json:"y"`
	D      uint256.Int `json:"d"`
	Down   finCall     `json:"down"`
	Up     finCall     `json:"up"`
}

// TestMMMulDivOZ replays OpenZeppelin (compat v4) Math.mulDiv, floor and Rounding.Up, over every triple of 15
// boundary values plus 400 seeded ones: a zero denominator is Panic(0x12) under a product that fits and the bare
// `require(denominator > prod1)` otherwise, and the rounded-up `result += 1` is Panic(0x11) at type(uint256).max
// (testdata/gen/GateIntEdges.t.sol).
func TestMMMulDivOZ(t *testing.T) {
	t.Parallel()
	var rows []mmMulDivRow
	loadGzFixture(t, "mm_muldiv_edges.json.gz", &rows)
	require.Equal(t, 15*15*15+400, len(rows)-1)
	mm := &finMismatches{t: t, area: "mulDiv"}
	for _, r := range rows[1:] {
		z, err := mmMulDivOZ(&r.X, &r.Y, &r.D)
		r.Down.checkWords(t, mm, "mulDiv("+r.X.Dec()+","+r.Y.Dec()+","+r.D.Dec()+")", []uint256.Int{z}, err)
		z, err = mmMulDivOZUp(&r.X, &r.Y, &r.D)
		r.Up.checkWords(t, mm, "mulDivUp("+r.X.Dec()+","+r.Y.Dec()+","+r.D.Dec()+")", []uint256.Int{z}, err)
	}
	require.Zero(t, mm.n)
}
