package flap

import "github.com/holiman/uint256"

// StaticExtra is set once by the list updater and never touched by the tracker. PortalAddress is
// carried here (rather than read from Config at simulator-construction time) because
// pool.RegisterFactory0's entity.Pool-only constructor signature has no other way to reach per-DEX
// config - this is the same reason CLAUDE.md calls out StaticExtra as the place to persist immutable
// metadata the simulator needs.
type StaticExtra struct {
	// PortalAddress is where swaps execute and where tokenIn must be approved.
	PortalAddress string `json:"pa"`
	HasNative     bool   `json:"hn"`
}

// Extra is refreshed by the tracker on every cycle from Portal.getTokenV8(token), Portal.getFeeRate()
// and Portal.enableTaxOnBondingCurve(). CirculatingSupply is the only field UpdateBalance mutates.
// JSON tags are abbreviated - this is stored per-pool in Redis.
type Extra struct {
	Status TokenStatus `json:"st"`
	Curve  Curve       `json:"cv"`

	CirculatingSupply *uint256.Int `json:"cs"`
	DexSupplyThresh   *uint256.Int `json:"dst"`

	// BuyFeeBps/SellFeeBps are Portal's own protocol fee from getFeeRate(), observed on-chain as
	// 100/100 (basis points out of bpsDenominator, i.e. 1%/1%).
	BuyFeeBps  uint64 `json:"bfb"`
	SellFeeBps uint64 `json:"sfb"`

	// BuyTaxBps/SellTaxBps are the launched token's own transfer tax, from getTokenV8's
	// buyTaxRate/sellTaxRate (same bps scale, cross-checked live against the board API's
	// tax.buyTaxBps/sellTaxBps for a live tax token). Only actually deducted on curve trades when
	// TaxOnBondingCurveEnabled is true (Portal.enableTaxOnBondingCurve(), a global switch) - verified
	// true on-chain. This is on top of, not instead of, BuyFeeBps/SellFeeBps: it's the token
	// contract's own transfer-tax mechanism, unrelated to Portal's protocol fee.
	BuyTaxBps                uint64 `json:"btb"`
	SellTaxBps               uint64 `json:"stb"`
	TaxOnBondingCurveEnabled bool   `json:"tobc"`
}

// SwapInfo is produced by CalcAmountOut and consumed by UpdateBalance.
type SwapInfo struct {
	NewCirculatingSupply *uint256.Int `json:"-"`
	NewStatus            TokenStatus  `json:"-"`
}

// PoolMeta is exposed to encoding so it can build the executor calldata / know where to approve.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	HasNative       bool   `json:"hasNative"`
	BlockNumber     uint64 `json:"blockNumber"`
}
