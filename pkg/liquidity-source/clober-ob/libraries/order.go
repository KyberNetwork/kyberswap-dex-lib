package cloberlib

import (
	"math/big"

	"github.com/holiman/uint256"
)

var (
	mask24 = uint256.MustFromHex("0xffffff").ToBig()
)

func DecodeOrderId(orderId *big.Int) (string, Tick) {
	// [192 bits: bookId][24 bits: tick][40 bits: index]
	bookId := new(big.Int).Rsh(orderId, 64)

	// tick = (id >> 40) & 0xffffff
	tick := new(big.Int).Rsh(orderId, 40)
	tick.And(tick, mask24)

	return bookId.String(), Tick(tick.Uint64())
}
