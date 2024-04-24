package msgpencode

import "github.com/holiman/uint256"

func EncodeUint256(x *uint256.Int) []byte {
	if x == nil {
		return nil
	}
	return x.Bytes()
}

func DecodeUint256(b []byte) *uint256.Int {
	if b == nil {
		return nil
	}
	return new(uint256.Int).SetBytes(b)
}

func EncodeUint256NonPtr(x uint256.Int) []byte {
	return EncodeUint256(&x)
}

func DecodeUint256NonPtr(b []byte) uint256.Int {
	u := DecodeUint256(b)
	if u == nil {
		return *uint256.NewInt(0)
	}
	return *u
}
