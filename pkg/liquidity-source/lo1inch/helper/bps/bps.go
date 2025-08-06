package bps

import "math/big"

func FromFraction(val int, base *big.Int) uint16 {
	bps := new(big.Int).SetInt64(int64(val) * 10_000)
	return uint16(bps.Div(bps, base).Int64())
}

func FromPercent(val int, base *big.Int) uint16 {
	bps := new(big.Int).SetInt64(int64(val) * 100)
	return uint16(bps.Div(bps, base).Int64())
}

func ToFraction(val int, base *big.Int) *big.Int {
	tmp := new(big.Int).SetInt64(int64(val))
	tmp.Mul(tmp, base)
	return tmp.Div(tmp, big.NewInt(10_000))
}

func ToPercent(val int, base *big.Int) *big.Int {
	tmp := new(big.Int).SetInt64(int64(val))
	tmp.Mul(tmp, base)
	return tmp.Div(tmp, big.NewInt(100))
}
