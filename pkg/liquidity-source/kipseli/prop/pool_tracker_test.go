package prop

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

func bigs(vals ...int64) []*big.Int {
	out := make([]*big.Int, len(vals))
	for i, v := range vals {
		out[i] = big.NewInt(v)
	}
	return out
}

// SwapImpl caps every quote at the wallet's tokenOut balance. Past the first
// probe that reaches it the curve is flat, and a spline over that plateau
// would sell ever-larger inputs for the same output.
func TestTrimAtWalletCap(t *testing.T) {
	points := bigs(1, 2, 3, 4)
	outs := bigs(10, 20, 25, 25)
	trimAtWalletCap(outs, big.NewInt(25))

	assert.Equal(t, []ladder.Point{{1, 10}, {2, 20}, {3, 25}}, ladder.CollectLadder(points, outs),
		"keeps the first capped probe, drops the plateau behind it")

	untouched := bigs(10, 20)
	trimAtWalletCap(untouched, big.NewInt(0))
	assert.Equal(t, bigs(10, 20), untouched, "an empty wallet reading is no cap")
}

// One-sided zeroing (ladder.ZeroUnquotedReserves) only touches the unquoted
// direction's output token. The live direction must keep the grid its own
// ladder guides, so it can't read the zeroed token's reserve as an input bound.
func TestSamplePoints_OneSidedNoQuote(t *testing.T) {
	extra, _ := json.Marshal(Extra{Ladders: [2][]ladder.Point{{{1e6, 1e6}, {2e6, 2e6}, {3e6, 2.1e6}}, nil}})
	p := entity.Pool{
		Tokens:   []*entity.PoolToken{{Decimals: 6}, {Decimals: 18}},
		Reserves: entity.PoolReserves{"5000000", "9000000"},
		Extra:    string(extra),
	}
	before := samplePoints(p)

	p.Reserves = ladder.ZeroUnquotedReserves(p.Reserves, [2][]ladder.Point{{{1, 1}}, nil})
	after := samplePoints(p)

	assert.Equal(t, entity.PoolReserves{"0", "9000000"}, p.Reserves)
	assert.Equal(t, before[0], after[0], "live 0->1 grid ignores the zeroed token0 reserve")
	assert.NotEqual(t, ladder.BuildDecimalsSweep(6), after[0], "live grid stays guided by its ladder")
	assert.Equal(t, ladder.BuildDecimalsSweep(18), after[1], "unquoted 1->0 falls back to the decimals sweep")
}

func TestApplyBuffer(t *testing.T) {
	outs := bigs(10_000, 12_345)
	(&PoolTracker{cfg: &Config{Buffer: 9_990}}).applyBuffer(outs)
	assert.Equal(t, bigs(9_990, 12_332), outs)

	untouched := bigs(10_000)
	(&PoolTracker{cfg: &Config{}}).applyBuffer(untouched)
	assert.Equal(t, bigs(10_000), untouched)
}
