package thogprop

import "strings"

// priceWord names one of the three packed price surfaces exactQuote reads.
type priceWord int

const (
	priceWordV1 priceWord = iota
	priceWordV2
	priceWordV3
)

// tokenMeta is one row of ThogAMM's fixed 8-token registry. Index, category,
// priceWord and priceShift are the protocol's own bit-packing layout (not
// derivable from getTokens()) and are load-bearing for every shift in
// math.go -- see THOGAMM_AGGREGATOR_INTEGRATION.md's token metadata table.
type tokenMeta struct {
	Index    int
	Address  string // lowercase, no checksum
	Category int
	Decimals uint8
	Word     priceWord
	Shift    uint
}

// tokenIdxXAUt is the index that requires riskV3Ready and uses the p1/p2
// XAUT-specific shift constants (136 for token shift, 126 for category
// shift) instead of the regular 40+idx*10 / cat*10 formula.
const tokenIdxXAUt = 7

// tokenTable is ThogAMM's fixed token registry, in the exact index order the
// doc requires getTokens(poolId) to return. Mismatched order silently
// corrupts every price/spread/covariance shift -- pool_list_updater.go
// verifies this order against the live contract before trusting it.
var tokenTable = []tokenMeta{
	{Index: 0, Address: "0x754704bc059f8c67012fed69bc8a327a5aafb603", Category: 0, Decimals: 6, Word: priceWordV2, Shift: 0},    // USDC
	{Index: 1, Address: "0x00000000efe302beaa2b3e6e1b18d08d69a9012a", Category: 0, Decimals: 6, Word: priceWordV2, Shift: 48},   // AUSD
	{Index: 2, Address: "0xe7cd86e13ac4309349f30b3435a9d337750fc82d", Category: 0, Decimals: 6, Word: priceWordV2, Shift: 96},   // USDT0
	{Index: 3, Address: "0x3bd359c1119da7da1d913d1c4d2b7c461115433a", Category: 1, Decimals: 18, Word: priceWordV1, Shift: 207}, // WMON
	{Index: 4, Address: "0xee8c0e9f1bffb4eb878d8f15f368a02a35481242", Category: 2, Decimals: 18, Word: priceWordV1, Shift: 63},  // WETH
	{Index: 5, Address: "0x0555e30da8f98308edb960aa94c0db47230d2b9c", Category: 3, Decimals: 8, Word: priceWordV1, Shift: 111},  // WBTC
	{Index: 6, Address: "0xd18b7ec58cdf4876f6afebd3ed1730e4ce10414b", Category: 3, Decimals: 8, Word: priceWordV1, Shift: 159},  // cbBTC
	{Index: 7, Address: "0x01bff41798a0bcf287b996046ca68b395dbc1071", Category: 4, Decimals: 6, Word: priceWordV3, Shift: 0},    // XAUt0
}

func tokenAddresses() []string {
	addrs := make([]string, len(tokenTable))
	for i, t := range tokenTable {
		addrs[i] = t.Address
	}
	return addrs
}

func tokenByAddress(addr string) (tokenMeta, bool) {
	addr = strings.ToLower(addr)
	for _, t := range tokenTable {
		if t.Address == addr {
			return t, true
		}
	}
	return tokenMeta{}, false
}

// categoryShift returns r1's category-width bit offset for a token's
// category: cat*10, except XAUT (category 4) which lives at bit 126.
func categoryShift(category int) uint {
	if category == 4 {
		return 126
	}
	return uint(category) * 10
}

// tokenShift returns r1's per-token-width bit offset: 40+idx*10, except
// XAUt0 (index 7) which lives at bit 136.
func tokenShift(index int) uint {
	if index == tokenIdxXAUt {
		return 136
	}
	return 40 + uint(index)*10
}
