package efraprop

// StaticExtra holds the immutable routing data of one market.
type StaticExtra struct {
	Venue string `json:"venue"`
	// Taker is passed to quoteExactIn as the address that will call swapExactIn, so the
	// probe already includes the gap discount the venue grants that address.
	Taker string `json:"taker"`
}

// PoolMeta is what the encoder needs: approve the venue, then call swapExactIn on it.
type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	Venue           string `json:"venue"`
	ApprovalAddress string `json:"approvalAddress"`
}
