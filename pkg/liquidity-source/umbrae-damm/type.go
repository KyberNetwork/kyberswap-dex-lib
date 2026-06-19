package umbraedamm

import "math/big"

// Extra is the mutable per-pool state the tracker refreshes each block.
//
//   - FeeBps is the current dynamic fee in basis points, snapshotted from currentFeeBps(). The
//     contract recomputes this from a volatility accumulator on every swap; KyberSwap re-tracks on
//     new blocks, so the simulator prices against the snapshot — the value getAmountOut() uses at
//     the tracked block.
//   - FeeToken is the side the fee is always charged in (WETH on the U1/WETH pair). The fee is
//     taken on the input when feeToken is the input side, otherwise on the output.
type Extra struct {
	FeeBps   uint64 `json:"feeBps"`
	FeeToken string `json:"feeToken"`
}

// SwapInfo carries the pair-side reserve deltas from CalcAmountOut to UpdateBalance so the latter
// never recomputes swap math. Fees exit reserves into accumulators (K stays constant), so the
// reserve deltas differ from the user-facing in/out amounts.
type SwapInfo struct {
	ReserveInDelta  *big.Int `json:"-"`
	ReserveOutDelta *big.Int `json:"-"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
