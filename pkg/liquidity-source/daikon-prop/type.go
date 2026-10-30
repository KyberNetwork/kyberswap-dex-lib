package daikonprop

import "github.com/holiman/uint256"

type StaticExtra struct {
	Router    string `json:"router"`
	Manager   string `json:"manager"`
	PartnerID uint64 `json:"partnerId"`
	FeePpm    uint64 `json:"feePpm"`
}

// Extra is PAMMManager.getMarketState plus the custody inventory. Tokens are [base, quote].
type Extra struct {
	BaseDecimals  uint8 `json:"baseDec"`
	QuoteDecimals uint8 `json:"quoteDec"`
	// Price is quote per whole base, WAD-scaled. ObservedAtMs is the publisher clock.
	Price        *uint256.Int `json:"price"`
	ObservedAtMs uint64       `json:"observedAtMs"`
	TTL          uint64       `json:"ttl"`
	Spread1Bps   uint64       `json:"s1"`
	Spread2Bps   uint64       `json:"s2"`
	// Remaining tier capacities in native quote units; buy and sell are independent.
	BuyC1  *uint256.Int `json:"buyC1"`
	BuyC2  *uint256.Int `json:"buyC2"`
	SellC1 *uint256.Int `json:"sellC1"`
	SellC2 *uint256.Int `json:"sellC2"`
	// VaultBase and VaultQuote are what custody can pay out: min(balance, allowance to the router).
	VaultBase  *uint256.Int `json:"vaultBase"`
	VaultQuote *uint256.Int `json:"vaultQuote"`
	// Paused covers the pair pause and the manager kill switch.
	Paused bool `json:"paused"`
}

// MetaInfo feeds the encoder: swap(..., abi.encode(PartnerID, FeePpm)) on Router.
type MetaInfo struct {
	Router          string `json:"router"`
	ApprovalAddress string `json:"approvalAddress"`
	PartnerID       uint64 `json:"partnerId"`
	FeePpm          uint64 `json:"feePpm"`
	BlockNumber     uint64 `json:"blockNumber"`
}

// SwapInfo records the capacity a swap consumed, in quote units.
type SwapInfo struct {
	BuyBase bool         `json:"buy"`
	C1Used  *uint256.Int `json:"c1"`
	C2Used  *uint256.Int `json:"c2"`
}

// normalize replaces absent ints with zero so the simulator never dereferences nil.
func (e *Extra) normalize() {
	for _, p := range []**uint256.Int{&e.Price, &e.BuyC1, &e.BuyC2, &e.SellC1, &e.SellC2, &e.VaultBase, &e.VaultQuote} {
		if *p == nil {
			*p = new(uint256.Int)
		}
	}
}
