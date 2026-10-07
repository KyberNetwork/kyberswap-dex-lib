package biconomyprop

import "github.com/holiman/uint256"

type StaticExtra struct {
	// Venue is the PropAMMVenue this pool belongs to. One venue serves every pair, so the pool
	// address is synthetic and swaps always target Venue.
	Venue string `json:"venue"`
}

// Board is one maker's board for one direction, as PropAMMVenue.board(mm, tokenIn, tokenOut)
// returns it: cumulative sizes, the price each level fills at when read, the tokenIn already
// filled, the depth a fill may still take (capped by the board's per-block limit) and the expiry.
// A dark, expired or exhausted board has Remaining == 0.
type Board struct {
	Sizes     []*uint256.Int `json:"sizes"`
	Prices    []*uint256.Int `json:"prices"` // tokenOut wei per tokenIn wei, 1e18-scaled
	Filled    *uint256.Int   `json:"filled"`
	Remaining *uint256.Int   `json:"remaining"`
	ExpiresAt uint64         `json:"expiresAt"`
}

// MemberExtra is one registered maker. Dir0 quotes token0 -> token1, Dir1 token1 -> token0.
type MemberExtra struct {
	Maker string `json:"maker"`
	Dir0  Board  `json:"dir0"`
	Dir1  Board  `json:"dir1"`
}

type Extra struct {
	// Members keeps the venue's registry order, which breaks price ties.
	Members []MemberExtra `json:"members"`
	FeeBps  uint64        `json:"feeBps"`
}

type memberTake struct {
	Member   int          `json:"member"`
	AmountIn *uint256.Int `json:"amountIn"`
}

type SwapInfo struct {
	DirIndex  int          `json:"dirIndex"`
	Takes     []memberTake `json:"takes"`
	AmountOut *uint256.Int `json:"amountOut"`
}

type MetaInfo struct {
	Venue       string `json:"venue"`
	BlockNumber uint64 `json:"blockNumber"`
}
