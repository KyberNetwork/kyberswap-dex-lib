package msgpencode

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func EncodeInt256(x *int256.Int) []byte {
	if x == nil {
		return nil
	}
	xU256 := (*uint256.Int)(x)
	return xU256.Bytes()
}

func DecodeInt256(b []byte) *int256.Int {
	if b == nil {
		return nil
	}
	x := new(uint256.Int)
	x.SetBytes(b)
	return (*int256.Int)(x)
}

func EncodeInt256NonPtr(x int256.Int) []byte {
	return EncodeInt256(&x)
}

func DecodeInt256NonPtr(b []byte) int256.Int {
	u := DecodeInt256(b)
	if u == nil {
		return *int256.NewInt(0)
	}
	return *u
}
