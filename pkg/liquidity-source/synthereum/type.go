package synthereum

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// PoolItem describes one statically embedded pool.
// Token order convention: index 0 = collateral (USDC/EURC), index 1 = synthetic token (jEUR).
type PoolItem struct {
	ID       string             `json:"id"`
	PoolType string             `json:"poolType"`
	Vault    string             `json:"vault,omitempty"`
	Tokens   []entity.PoolToken `json:"tokens"`
}

// StaticExtra holds immutable pool metadata.
type StaticExtra struct {
	PoolType string `json:"poolType"`
	// Vault is the ERC4626 Morpho vault where the fixed-rate wrapper deposits its collateral (wrapper pools only).
	Vault string `json:"vault,omitempty"`
}

// Extra holds mutable pool state refreshed by the tracker.
//
// For multi-lp pools, the tracker probes the on-chain quoter with one whole unit of
// the input token and stores the results; the simulator prices linearly from these
// probes (mint/redeem are linear in amount for a fixed oracle price and fee).
type Extra struct {
	// multi-lp pool state
	MintProbeIn    *uint256.Int `json:"mintProbeIn,omitempty"`    // collateral probe amount (1 whole collateral unit)
	MintProbeOut   *uint256.Int `json:"mintProbeOut,omitempty"`   // synthTokensReceived for MintProbeIn (net of fee)
	MintProbeFee   *uint256.Int `json:"mintProbeFee,omitempty"`   // feePaid (collateral) for MintProbeIn
	RedeemProbeIn  *uint256.Int `json:"redeemProbeIn,omitempty"`  // synthetic probe amount (1 whole synth unit)
	RedeemProbeOut *uint256.Int `json:"redeemProbeOut,omitempty"` // collateralAmountReceived for RedeemProbeIn (net of fee)
	RedeemProbeFee *uint256.Int `json:"redeemProbeFee,omitempty"` // feePaid (collateral) for RedeemProbeIn
	FeePercentage  *uint256.Int `json:"feePercentage,omitempty"`  // 1e18-scaled fee percentage
	MaxSynthCap    *uint256.Int `json:"maxSynthCap,omitempty"`    // maxTokensCapacity(): max synth mintable
	TotalSynth     *uint256.Int `json:"totalSynth,omitempty"`     // totalSyntheticTokens(): max synth redeemable

	// wrapper pool state
	WrapperReserve *uint256.Int `json:"wrapperReserve,omitempty"` // collateral redeemable from the vault for unwraps
}

type Gas struct {
	Mint   int64
	Redeem int64
	Wrap   int64
	Unwrap int64
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
}
