package mhocdesk

import "github.com/holiman/uint256"

// StaticExtra is set once by the list updater. Tokens[0] is always BASE and Tokens[1] QUOTE.
type StaticExtra struct {
	BaseDecimals  uint8  `json:"bd"`
	QuoteDecimals uint8  `json:"qd"`
	PriceScale    string `json:"ps"` // PRICE_SCALE(), decimal string
	KScale        uint32 `json:"ks"` // K_SCALE() (1..256): the unit of the card's kBid and kAsk fields
}

// Extra is everything PropAMM._compute reads besides the two token balances (which are the pool's Reserves),
// as of one block. The tracker reads it through the contract's public views, pinned to that block.
type Extra struct {
	// StateWord is stateWord(): the 32-byte price card (layout in card.go).
	StateWord *uint256.Int `json:"w"`

	FloorBase   *uint256.Int `json:"fb"` // floorBase(): BASE raw units a swap may never take the pool below
	FloorQuote  *uint256.Int `json:"fq"` // floorQuote()
	MaxNotional uint64       `json:"mn"` // maxNotional(): owner per-swap cap, whole QUOTE tokens

	// MaxPerUpdate is maxNotionalPerUpdate() = C, whole QUOTE tokens: the cap on the notional against one price card
	// and on the notional in one block. MaxPerWindow is maxNotionalPerWindow() = W, whole QUOTE tokens per calendar
	// minute; 0 means no trading.
	MaxPerUpdate uint64 `json:"mu"`
	MaxPerWindow uint32 `json:"mw"`

	// Counters is counters(): the raw counters with their tags. Each one is reset lazily by its own key, so the
	// simulator evaluates them for the block and timestamp of the simulated swap.
	Counters Counters `json:"c"`

	// ReadTimestamp is block.timestamp of the block the snapshot was read at.
	ReadTimestamp uint64 `json:"ts"`
}

// Counters mirrors counters(), in QUOTE raw units:
//
//	Fs (per card)  = CardFilled  if CardSeq == seq of the card, else 0
//	Fb (per block) = BlockFilled if FillBlock == uint32(block.number) and WindowID == block.timestamp/60, else 0
//	minute fill    = WindowFilled if WindowID == block.timestamp/60, else 0
type Counters struct {
	CardSeq      uint32       `json:"cs"`
	CardFilled   *uint256.Int `json:"cf"`
	FillBlock    uint32       `json:"bk"`
	BlockFilled  *uint256.Int `json:"bf"`
	WindowID     uint32       `json:"wi"`
	WindowFilled *uint256.Int `json:"wf"`
}

// SwapInfo carries what UpdateBalance needs to replay the contract's counter writes.
type SwapInfo struct {
	Notional    *uint256.Int `json:"notional"`    // Q, QUOTE raw units: what the swap adds to every counter
	BlockNumber uint64       `json:"blockNumber"` // block.number the swap was simulated at
	Timestamp   uint64       `json:"timestamp"`   // block.timestamp for the per-minute budget (windowNow)
}
