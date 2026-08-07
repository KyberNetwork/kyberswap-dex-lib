package flap

import "github.com/holiman/uint256"

// StaticExtra is set once by the list updater and never touched by the tracker. PortalAddress is
// carried here (rather than read from Config at simulator-construction time) because
// pool.RegisterFactory0's entity.Pool-only constructor signature has no other way to reach per-DEX
// config - this is the same reason CLAUDE.md calls out StaticExtra as the place to persist immutable
// metadata the simulator needs.
type StaticExtra struct {
	// QuoteToken is the raw quote token address reported by the board API before native-wrapping,
	// e.g. the zero address when the pair quotes in the chain's native token.
	QuoteToken string `json:"quoteToken"`

	// PortalAddress is where swaps execute and where tokenIn must be approved.
	PortalAddress string `json:"portalAddress"`
}

// Extra is refreshed by the tracker on every cycle from Portal.getTokenV8(token), Portal.getFeeRate()
// and Portal.enableTaxOnBondingCurve(). CirculatingSupply is the only field UpdateBalance mutates.
type Extra struct {
	Status TokenStatus `json:"status"`
	Curve  Curve       `json:"curve"`

	CirculatingSupply *uint256.Int `json:"circulatingSupply"`
	DexSupplyThresh   *uint256.Int `json:"dexSupplyThresh"`

	// BuyFeeBps/SellFeeBps are Portal's own protocol fee from getFeeRate(), observed on-chain as
	// 100/100 (basis points out of bpsDenominator, i.e. 1%/1%).
	BuyFeeBps  uint64 `json:"buyFeeBps"`
	SellFeeBps uint64 `json:"sellFeeBps"`

	// BuyTaxBps/SellTaxBps are the launched token's own transfer tax, from getTokenV8's
	// buyTaxRate/sellTaxRate (same bps scale, cross-checked live against the board API's
	// tax.buyTaxBps/sellTaxBps for a live tax token). Only actually deducted on curve trades when
	// TaxOnBondingCurveEnabled is true (Portal.enableTaxOnBondingCurve(), a global switch) - verified
	// true on-chain. This is on top of, not instead of, BuyFeeBps/SellFeeBps: it's the token
	// contract's own transfer-tax mechanism, unrelated to Portal's protocol fee.
	BuyTaxBps                uint64 `json:"buyTaxBps"`
	SellTaxBps               uint64 `json:"sellTaxBps"`
	TaxOnBondingCurveEnabled bool   `json:"taxOnBondingCurveEnabled"`
}

// SwapInfo is produced by CalcAmountOut and consumed by UpdateBalance.
type SwapInfo struct {
	NewCirculatingSupply *uint256.Int `json:"-"`
	NewStatus            TokenStatus  `json:"-"`
}
