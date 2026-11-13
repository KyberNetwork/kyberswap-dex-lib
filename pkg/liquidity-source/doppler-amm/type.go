package doppleramm

// StaticExtra carries the DopplerAMM market identity. A market is a
// configuration hash, not a pool contract, so every read is keyed by MarketID;
// TokenX/TokenY fix which reserve is index 0.
type StaticExtra struct {
	MarketID string `json:"marketId"`
	TokenX   string `json:"tokenX"`
	TokenY   string `json:"tokenY"`
	AMM      string `json:"amm"`
	Quoter   string `json:"quoter"`
}

// Extra holds the fee rates resolved at refresh time. The module LP fee is not
// readable from state (it lives in the component's config pointer), so the
// tracker recovers it from one quoter probe; the protocol base fee comes from
// protocolFeeForMarket.
type Extra struct {
	// TokenX is repeated here so the encoder can derive inputIsX from the
	// route summary, which carries Extra but not StaticExtra.
	TokenX     string `json:"tokenX"`
	BaseFeeBps uint16 `json:"baseFeeBps"`
	LpFeeBps   uint16 `json:"lpFeeBps"`
}
