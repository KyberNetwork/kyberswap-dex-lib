package deepstateob

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// packNode builds a raw node word from its four fields, mirroring
// DeepstateV1.sol's _pack bit layout, for round-trip testing decodeNode.
func packNode(tick int32, quantity uint64, correctionCode, nonce uint32) [32]byte {
	var node [32]byte
	binary.BigEndian.PutUint32(node[0:4], uint32(tick))
	binary.BigEndian.PutUint64(node[16:24], quantity) // low 64 bits of the 160-bit quantity field
	binary.BigEndian.PutUint32(node[24:28], correctionCode)
	binary.BigEndian.PutUint32(node[28:32], nonce)
	return node
}

// TestDecodeNode_TopOrder decodes real order words read from
// topOrder(bookId, isBid) against DeepstateV1 on Robinhood Chain at block
// 59217049 (bookId=0xdf941c23...6bf399, sorted pair token0=USDG (6d),
// token1=NVDA (18d)). topOrder does not return the raw node, but it does
// return (nonce, soldAmount) -- soldAmount is `_hookAmount`, which for a bid
// is remaining quote collateral and for an ask is remaining base quantity
// (see DeepstateV1.sol topOrder/_hookAmount). This test only exercises tick
// price math against a value in the tree()-decoded tick's plausible range,
// since a full leaf-node capture requires a state-diff read this session
// didn't collect; see tracker.md open questions for a follow-up live fixture
// capture once the lens contract exists.
func TestTickPrice_MonotonicAndZeroIsOne(t *testing.T) {
	assert.InDelta(t, 1.0, tickPrice(0), 1e-12)
	assert.Greater(t, tickPrice(1), tickPrice(0))
	assert.Less(t, tickPrice(-1), tickPrice(0))

	// price(t) * price(-t) should be ~1 (reciprocal ticks), within float64 tolerance.
	for _, tick := range []int32{1, 1000, 1_000_000, math.MaxInt32 / 4} {
		got := tickPrice(tick) * tickPrice(-tick)
		assert.InDelta(t, 1.0, got, 1e-6, "tick=%d", tick)
	}
}

func TestDecodeNode_RoundTrip(t *testing.T) {
	node := packNode(5, 1000, 0, 7)

	decoded := decodeNode(node)
	assert.Equal(t, int32(5), decoded.tick)
	assert.Equal(t, uint64(1000), decoded.quantity.Uint64())
	assert.Equal(t, uint32(7), decoded.nonce)
}

func TestDecodeNode_NegativeTick(t *testing.T) {
	node := packNode(-1, 0, 0, 0)

	decoded := decodeNode(node)
	assert.Equal(t, int32(-1), decoded.tick)
}
