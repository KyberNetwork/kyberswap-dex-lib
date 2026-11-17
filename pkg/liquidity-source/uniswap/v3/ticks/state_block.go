package ticks

import (
	"github.com/goccy/go-json"
)

// TicksBasedStateBlock is the block R of the latest read of a ticks-based pool's tracked state
// (slot0, liquidity, hook state, ...). A ticks-based tracker reads that state at latest, while the
// pool's BlockNumber is the tick-sync cursor H (the last applied log) and must not change, so the
// two differ. A consumer that replays a quote at the quoted block needs R.
//
// A tracker sets it to the block of its state read on every write, so it also tells when the
// pool's state was last read. Embed it in an Extra, to store R, and in a meta, to expose it, so
// both use the key "sB". StateBlock 0 means unknown: an Extra written before the field existed, or
// a state read that exists at no block (eth_call overrides).
type TicksBasedStateBlock struct {
	StateBlock uint64 `json:"sB,omitempty"`
}

// TicksBasedStateBlockOf returns the StateBlock of a pool's raw Extra, 0 if the Extra has none or
// is not JSON.
func TicksBasedStateBlockOf(extra string) uint64 {
	var info TicksBasedStateBlock
	_ = json.Unmarshal([]byte(extra), &info)
	return info.StateBlock
}

// ForMeta returns the TicksBasedStateBlock for a pool's meta: the block the pool's state is valid
// at, which is never below the pool's BlockNumber. It is zero, so omitted from the JSON, when the
// StateBlock is unknown: BlockNumber is then the best block available.
func (s TicksBasedStateBlock) ForMeta(blockNumber uint64) TicksBasedStateBlock {
	if s.StateBlock == 0 {
		return TicksBasedStateBlock{}
	}
	return TicksBasedStateBlock{StateBlock: max(s.StateBlock, blockNumber)}
}
