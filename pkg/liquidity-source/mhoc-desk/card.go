package mhocdesk

import "github.com/holiman/uint256"

// The price card (stateWord(), slot 0) is the only place the contract's bit layout appears; a new layout changes
// this file only. Layout, from the least significant bit:
//
//	[0,32) validUntil | [32,64) seq | [64,88) qMax | [88,104) kAsk | [104,120) kBid | [120,136) dAsk
//	| [136,152) dBid | [152,216) price | [216,248) validUntilBlock | [248,256) flags (bit 0 paused)

// flagPausedBit is bit 0 of the flags byte (the top byte of the state word).
const flagPausedBit = 248

// cardFields is the decoded price card.
type cardFields struct {
	paused          bool
	price           *uint256.Int // 64 bits: QUOTE raw units per whole BASE token, times PRICE_SCALE
	dBid            uint64       // 16 bits, 1e-6
	dAsk            uint64       // 16 bits, 1e-6
	kBid            uint64       // 16 bits: size slippage for takers who sell BASE, in units of K_SCALE
	kAsk            uint64       // 16 bits: size slippage for takers who buy BASE, in units of K_SCALE
	qMax            uint64       // 24 bits, whole QUOTE tokens
	seq             uint32       // 32 bits
	validUntil      uint64       // 32 bits, unix seconds
	validUntilBlock uint64       // 32 bits, the last block number in which the card may trade
}

func decodeCard(w *uint256.Int) cardFields {
	var t uint256.Int
	field := func(shift, bits uint) uint64 {
		t.Rsh(w, shift)
		if bits == 64 {
			return t.Uint64()
		}
		return t.Uint64() & (1<<bits - 1)
	}
	price := uint256.NewInt(field(152, 64))
	return cardFields{
		paused:          field(flagPausedBit, 1) == 1,
		price:           price,
		dBid:            field(136, 16),
		dAsk:            field(120, 16),
		kBid:            field(104, 16),
		kAsk:            field(88, 16),
		qMax:            field(64, 24),
		seq:             uint32(field(32, 32)),
		validUntil:      field(0, 32),
		validUntilBlock: field(216, 32),
	}
}

// live reports whether the card may trade in block (blockNumber, now): both expiries must hold. The whole block
// number is compared with the 32-bit field, as the contract does.
func (c *cardFields) live(blockNumber, now uint64) bool {
	return now <= c.validUntil && blockNumber <= c.validUntilBlock
}
