package deepstateob

import (
	"math"

	"github.com/holiman/uint256"
)

// decodedNode is one packed DeepstateV1 node (bytes32): a leaf when
// tree(id,node) reports a zero leftNode, otherwise a branch. See
// DeepstateV1.sol's top-of-file NatSpec for the bit layout:
//
//	bits 224-255: signed int32 tick
//	bits  64-223: uint160 quantity (token0 units, raw/undecimaled)
//	bits  32-63:  uint32 same-tick branch correction code (0 for leaves)
//	bits   0-31:  uint32 nonce
type decodedNode struct {
	tick     int32
	quantity *uint256.Int
	nonce    uint32
}

var quantityMask = new(uint256.Int).Sub(
	new(uint256.Int).Lsh(uint256.NewInt(1), 160),
	uint256.NewInt(1),
)

// decodeNode extracts tick/quantity/nonce from a raw packed node word. It is
// used only for leaves: a branch's own word encodes an aggregate sort key,
// not a real tick, so callers must first confirm tree(id,node).leftNode == 0
// before decoding.
func decodeNode(node [32]byte) decodedNode {
	var word uint256.Int
	word.SetBytes(node[:])

	var tmp uint256.Int
	tmp.Rsh(&word, 224)
	tick := int32(uint32(tmp.Uint64()))

	quantity := new(uint256.Int).Rsh(&word, 64)
	quantity.And(quantity, quantityMask)

	nonce := uint32(word.Uint64())

	return decodedNode{tick: tick, quantity: quantity, nonce: nonce}
}

// tickPrice returns 2**(96*tick/2**31) as a float64, per DeepstateV1's
// NatSpec price definition -- token1-raw per token0-raw at that tick.
func tickPrice(tick int32) float64 {
	exponent := tickPriceExpNumerator * float64(tick) / tickPriceExpDenominator
	return math.Exp2(exponent)
}
