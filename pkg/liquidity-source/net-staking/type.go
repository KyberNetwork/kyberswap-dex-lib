package netstaking

import "github.com/holiman/uint256"

// PoolExtra holds the block-by-block tracked state for the single
// NET/sNET/wsNET pool. It is stored as JSON in entity.Pool.Extra.
type PoolExtra struct {
	// Index is the sNET rebase index (9-decimal fixed point, same convention as gOHM).
	// wsNET.sNetToWs(x) = x * 1e18 / Index; wsNET.wsToSNet(x) = x * Index / 1e18.
	Index *uint256.Int `json:"index"`

	// NETReserve is NET.balanceOf(staking). Caps outbound NET: sNET->NET (unstake).
	NETReserve *uint256.Int `json:"netReserve"`

	// SNETStakingReserve is sNET.balanceOf(staking). Caps outbound sNET: NET->sNET
	// (stake); NET->wsNET's first leg (stake) is also capped by this.
	SNETStakingReserve *uint256.Int `json:"sNetStakingReserve"`

	// SNETWrapReserve is sNET.balanceOf(wrap). Caps outbound sNET: wsNET->sNET (unwrap).
	SNETWrapReserve *uint256.Int `json:"sNetWrapReserve"`
}

type Gas struct {
	Stake        int64
	Unstake      int64
	Wrap         int64
	Unwrap       int64
	StakeAndWrap int64
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
	NET         string `json:"net"`
	SNET        string `json:"sNet"`
	WSNET       string `json:"wsNet,omitempty"`

	// ApprovalAddress is the first-hop contract for the swap direction: the wsNET/wrap
	// contract for ActionWrap (sNET->wsNET) only, or the pool address (Staking) for
	// every other action. Unlike gohm (single contract == pool address for every
	// action), this pool spans two contracts, so the approval target varies by direction.
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

// SwapInfo is the per-swap data attached to CalcAmountOutResult.SwapInfo.
// Aggregator-encoding reads this from swap.Extra to build the executor payload
// without re-deriving the action from (tokenIn, tokenOut).
type SwapInfo struct {
	Action Action `json:"action"`
}
