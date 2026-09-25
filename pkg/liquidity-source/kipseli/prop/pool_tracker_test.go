package prop

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

func TestSampleBound(t *testing.T) {
	balance := big.NewInt(100)

	assert.Equal(t, big.NewInt(30), sampleBound(balance, maxIn(balance, big.NewInt(130))),
		"cap room tighter than the balance bounds the grid")
	assert.Equal(t, balance, sampleBound(balance, maxIn(balance, big.NewInt(1000))))
	assert.Equal(t, balance, sampleBound(balance, maxIn(balance, bignumber.MaxUint256)),
		"the quote token's max-uint cap means uncapped")
	assert.Nil(t, maxIn(balance, big.NewInt(90)), "over cap: no room to report, fall back to balance")
}

func TestApplyBuffer(t *testing.T) {
	outs := bigs(10_000, 12_345)
	(&PoolTracker{cfg: &Config{Buffer: 9_990}}).applyBuffer(outs)
	assert.Equal(t, bigs(9_990, 12_332), outs)

	untouched := bigs(10_000)
	(&PoolTracker{cfg: &Config{}}).applyBuffer(untouched)
	assert.Equal(t, bigs(10_000), untouched)
}
