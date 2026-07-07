package gohm

import "github.com/holiman/uint256"

// PoolExtra holds the block-by-block tracked state for the single gOHM pool.
// It is stored as JSON in entity.Pool.Extra.
type PoolExtra struct {
	// Index is the sOHM rebase index — the sole pricing primitive.
	// Stored as uint256 string (raw, not scaled). Unit: sOHM-decimals (9).
	// gOHM.balanceFrom(gOHMAmount) = gOHMAmount * Index / 1e18
	// gOHM.balanceTo(OHMAmount)    = OHMAmount  * 1e18  / Index
	Index *uint256.Int `json:"index"`

	// WarmupPeriod is the current warmup epoch count. When > 0, staking is
	// non-atomic and the pool is not routable.
	WarmupPeriod uint64 `json:"warmupPeriod"`

	// OHMReserve is OHM.balanceOf(staking) — the OHM the staking contract holds.
	// Caps outbound OHM: unstake (sOHM->OHM) and gOHM->OHM both require
	// amount <= OHMReserve (enforced on-chain by require(amount_ <= OHM.balanceOf(this))).
	OHMReserve *uint256.Int `json:"ohmReserve"`

	// SOHMReserve is sOHM.balanceOf(staking) — the sOHM the staking contract holds.
	// Caps outbound sOHM: stake (OHM->sOHM) and gOHM->sOHM (unwrap) both require
	// amount <= SOHMReserve (staking transfers out sOHM 1:1 / via safeTransfer).
	SOHMReserve *uint256.Int `json:"sohmReserve"`
}

type Gas struct {
	Stake   int64
	Unstake int64
	Wrap    int64
	Unwrap  int64
}

type PoolMeta struct {
	BlockNumber uint64 `json:"bN"`
	OHM         string `json:"ohm"`
	SOHM        string `json:"sohm"`
	GOHM        string `json:"gohm"`
}

// SwapInfo is the per-swap data attached to CalcAmountOutResult.SwapInfo.
// Aggregator-encoding reads this from swap.Extra to build the executor payload
// without re-deriving the action from (tokenIn, tokenOut).
type SwapInfo struct {
	Action Action `json:"action"`
}
