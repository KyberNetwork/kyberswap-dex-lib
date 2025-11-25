package cloberlib

import (
	"math/big"
)

func DecodeOrderId(orderId *big.Int) (string, Tick) {
	bookId := new(big.Int).Rsh(orderId, 64)
	tickUint := new(big.Int).Rsh(orderId, 40).Uint64() & 0xffffff

	tick := Tick(tickUint)
	if tickUint >= 0x800000 {
		tick -= 0x1000000
	}

	return bookId.String(), tick
}
