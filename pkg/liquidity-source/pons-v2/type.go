package ponsv2

import (
	"math/big"

	"github.com/holiman/uint256"
)

// StaticExtra carries per-curve values that are immutable once
// PonsV2BondingCurve.initialize() has run (i.e. by the time the factory's
// TokenLaunched event fires, since initialize() runs earlier in the same
// launchToken() transaction). Discovery (pool_factory.go) is a pure log
// decode with no RPC calls, so these are left zero-valued there; PoolTracker
// fetches and (re-)populates all four every GetNewPoolState pass instead of
// only once, since that's the first point any of them are actually known.
type StaticExtra struct {
	// FeeBps is the immutable protocol/creator/buyback trade fee, in basis
	// points, always charged on the quote leg (PonsV2BondingCurve.feeBps()).
	FeeBps uint16 `json:"feeBps"`
	// CreatorTaxBps is the immutable creator-chosen tax layered on top of
	// FeeBps, also always charged on the quote leg
	// (PonsV2BondingCurve.creatorTaxBps()).
	CreatorTaxBps uint16 `json:"creatorTaxBps"`
	// ReservedTokens is the token allocation this curve will never sell
	// below, withheld to seed the post-graduation Uniswap V4 pool
	// (PonsV2BondingCurve.reservedTokens()). sellableTokens() ==
	// max(TokenReserve-ReservedTokens, 0).
	ReservedTokens *uint256.Int `json:"reservedTokens"`
	// IsNativeQuote mirrors PonsV2BondingCurve.isNativeQuote(): true when
	// pairToken() == address(0), meaning quote-side trades move native ETH
	// rather than an ERC-20.
	IsNativeQuote bool `json:"isNativeQuote"`
}

// Extra carries the block-by-block mutable curve state, refreshed on every
// tracker pass.
type Extra struct {
	// QuoteReserve is PonsV2BondingCurve.getReserves()'s quoteReserve_:
	// phantomQuote + trackedQuote - quoteFeeBalance - creatorTaxBalance.
	// Already nets out pending fees/tax and folds in the virtual reserve, so
	// nothing else needs to be tracked for the quote leg.
	QuoteReserve *uint256.Int `json:"quoteReserve"`
	// TokenReserve is PonsV2BondingCurve.getReserves()'s tokenReserve_
	// (== trackedTokens), the FULL token balance including the
	// ReservedTokens floor. The curve's own constant-product math always
	// prices against this full value; only the resulting tokensOut is
	// clamped against sellableTokens() (TokenReserve - ReservedTokens).
	TokenReserve *uint256.Int `json:"tokenReserve"`
	// Graduated mirrors PonsV2BondingCurve.graduated(). Once true, buy()
	// reverts unconditionally on-chain; sell() also reverts once
	// sellableTokens()==0 even before this flips (see readyToGraduate in
	// pool_simulator.go), so both must be rejected by the simulator.
	Graduated bool `json:"graduated"`
}

// SwapInfo is the swap-specific state CalcAmountOut hands to UpdateBalance.
// Both buy and sell only ever move QuoteReserve/TokenReserve by the amounts
// actually applied on-chain (buy's clamp-and-refund means "amount offered"
// can differ from "amount actually spent").
type SwapInfo struct {
	IsBuy bool `json:"isBuy"`
	// IsNativeQuote mirrors StaticExtra.IsNativeQuote (also isNativeQuote()
	// on-chain). Carried on SwapInfo, not just StaticExtra, so the
	// aggregator-encoding layer can decide native msg.value handling (buy)
	// / native payout (sell) per swap without a second lookup: on-chain,
	// PonsV2BondingCurve.buy() requires msg.value == quoteIn only when
	// isNativeQuote() is true, and this must be distinguished from a curve
	// whose pairToken happens to be the real WETH ERC-20 contract (which
	// still takes a normal transferFrom, not native value).
	//
	// Deliberately NOT tagged json:"-": aggregator-encoding's
	// util.AnyToStruct round-trips swap.Extra through json.Marshal/Unmarshal,
	// which drops "-"-tagged fields; a "-" tag here would make this always
	// decode to false downstream (matches why IsBuy above also has a real tag).
	IsNativeQuote   bool         `json:"isNativeQuote"`
	NewQuoteReserve *uint256.Int `json:"-"`
	NewTokenReserve *uint256.Int `json:"-"`
	NewGraduated    bool         `json:"-"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"blockNumber"`
}

// curveReservesResult mirrors PonsV2BondingCurve.getReserves()'s two return
// values for ethrpc decoding. Fields must stay *big.Int (not *uint256.Int):
// go-ethereum's reflection-based abi.set only knows how to populate
// *big.Int for a uint256 output; converting to *uint256.Int happens
// afterward in pool_tracker.go.
type curveReservesResult struct {
	QuoteReserve *big.Int
	TokenReserve *big.Int
}
