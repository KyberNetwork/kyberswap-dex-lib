package synthereum

import (
	"github.com/holiman/uint256"
)

// StaticExtra holds immutable pool metadata.
type StaticExtra struct {
	PoolType string `json:"poolType"`
	// Vault is the ERC4626 Morpho vault where the fixed-rate wrapper deposits its collateral (wrapper pools only).
	Vault string `json:"vault,omitempty"`
}

// Extra holds mutable pool state refreshed by the tracker.
//
// For multi-lp pools, mint/redeem are computed with the pool's own exact integer
// formula (SynthereumMultiLpLiquidityPoolLib._calculateMint/_calculateRedeem, both
// PreciseUnitMath-floor-rounded) from the tracked oracle Price and FeePercentage —
// not approximated from a probed quote — so the simulator matches the on-chain
// getMintTradeInfo/getRedeemTradeInfo output to the wei for any input size.
type Extra struct {
	// multi-lp pool state
	Price         *uint256.Int `json:"price,omitempty"`         // 1e18-scaled priceFeed rate (finder.PriceFeed.getLatestPrice(priceFeedIdentifier()))
	FeePercentage *uint256.Int `json:"feePercentage,omitempty"` // 1e18-scaled fee percentage
	MaxSynthCap   *uint256.Int `json:"maxSynthCap,omitempty"`   // maxTokensCapacity(): max synth mintable
	TotalSynth    *uint256.Int `json:"totalSynth,omitempty"`    // totalSyntheticTokens(): max synth redeemable

	// wrapper pool state
	WrapperReserve    *uint256.Int `json:"wrapperReserve,omitempty"`    // vault.maxWithdraw(wrapper): collateral actually withdrawable now; unwrapping above it reverts NotEnoughLiquidity()
	WrapperSynthCap   *uint256.Int `json:"wrapperSynthCap,omitempty"`   // totalSyntheticTokens() on the wrapper itself: the binding on-chain unwrap cap ('Synth tokens amount too high' revert guard)
	WrapperRate       *uint256.Int `json:"wrapperRate,omitempty"`       // conversionRate(): 1e18-scaled: when == 1e18 unwrap requires an exact multiple of the scaling factor on-chain (reverts otherwise, does not floor)
	WrapperMaxDeposit *uint256.Int `json:"wrapperMaxDeposit,omitempty"` // vault.maxDeposit(wrapper), collateral units: wrap()'s underlying vault deposit reverts above this
}

type Gas struct {
	Mint   int64
	Redeem int64
	Wrap   int64
	Unwrap int64
}

// PoolMeta is what the executor-side encoder receives. PoolType and IsCollateralIn
// together select which of the four on-chain entry points the swap encodes to:
// multi-lp mint/redeem, or wrapper wrap/unwrap. Token index 0 is always the
// collateral and index 1 the synthetic, so the direction reduces to which side the
// input token sits on.
//
// No approvalAddress here on purpose: the spender is always the pool, so the encoder
// resolves it from its own dex table (useApproveMaxDexes) rather than having every
// quote carry it. See GetApprovalAddress.
type PoolMeta struct {
	BlockNumber    uint64 `json:"blockNumber"`
	PoolType       string `json:"poolType"`
	IsCollateralIn bool   `json:"isCollateralIn"`
}
