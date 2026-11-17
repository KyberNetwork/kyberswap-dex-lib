package uniswapv3

import (
	"math"
	"testing"
)

// onchainWalk is one swap measured on an Ethereum fork at block 26307077 (2026-10-30): the gas the pool spent
// (its swap frame, the tokens it moves and the pay-in transfer; the executor's own share left out) by a
// debug_traceCall of the route's build, and the swap-loop iterations dex-lib counts for the same swap.
type onchainWalk struct {
	pool       string
	ticks      int // CrossInitTickLoops
	words      int // CrossEmptyWordLoops
	poolGasUse int64
}

// v3Walks are the uniswap v3 swaps of sampled routes that walked 40 or more tick-bitmap words.
// Every word costs 7,350-7,500 gas on-chain: a cold
// SLOAD of the word, getSqrtRatioAtTick, computeSwapStep and the loop's own bookkeeping, about three times the
// 2,500 the first estimate (#1591) charged. Swaps through such pools were priced at a third of their gas, and
// the routes holding them ran 18-22% over their estimate.
var v3Walks = []onchainWalk{
	{"0xb3453652", 0, 585, 4_409_905},
	{"0x38c0e310", 4, 476, 3_692_961},
	{"0xbe80225f", 8, 516, 4_091_184},
	{"0x1b9d58be", 1, 66, 625_869},
	{"0x8bb9cd88", 46, 43, 1_522_906},
}

func walkGas(g Gas, w onchainWalk) int64 {
	return g.BaseGas + g.CrossInitTickGas*int64(w.ticks) + g.CrossEmptyWordGas*int64(w.words)
}

func TestEmptyWordGasMatchesTheChain(t *testing.T) {
	for _, w := range v3Walks {
		got := walkGas(defaultGas, w)
		if off := math.Abs(float64(got-w.poolGasUse)) / float64(w.poolGasUse); off > 0.10 {
			t.Errorf("%s (%d ticks, %d words): priced %d, spent %d on-chain (%.0f%% off)", w.pool, w.ticks, w.words,
				got, w.poolGasUse, off*100)
		}
	}
}
