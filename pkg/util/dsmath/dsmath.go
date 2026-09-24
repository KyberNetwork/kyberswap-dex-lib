package dsmath

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	WAD = number.Number_1e18
)

// WDiv
// ((x * WAD) + (y / 2)) / y
func WDiv(x *uint256.Int, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(
		new(uint256.Int).Add(
			new(uint256.Int).Mul(x, WAD),
			new(uint256.Int).Div(y, number.Number_2),
		),
		y,
	)
}

// WMul
// ((x * y) + (WAD / 2)) / WAD;
func WMul(x *uint256.Int, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(
		new(uint256.Int).Add(
			new(uint256.Int).Mul(x, y),
			new(uint256.Int).Div(WAD, number.Number_2),
		),
		WAD,
	)
}

// ToWad
// Convert x to WAD (18 decimals) from d decimals.
func ToWAD(x *uint256.Int, d uint8) *uint256.Int {
	if d < 18 {
		return new(uint256.Int).Mul(x, number.TenPow(18-d))
	}

	if d > 18 {
		return new(uint256.Int).Div(x, number.TenPow(d-18))
	}

	return new(uint256.Int).Set(x)
}

// FromWAD
// Convert x from WAD (18 decimals) to d decimals.
func FromWAD(x *uint256.Int, d uint8) *uint256.Int {
	if d < 18 {
		return new(uint256.Int).Div(x, number.TenPow(18-d))
	}

	if d > 18 {
		return new(uint256.Int).Mul(x, number.TenPow(d-18))
	}

	return new(uint256.Int).Set(x)
}
