package gohm

import (
	"github.com/holiman/uint256"
)

var number_1e18 = uint256.NewInt(1e18)

func balanceFrom(gOHMAmount, index *uint256.Int) (*uint256.Int, error) {
	if index.IsZero() {
		return nil, ErrIndexZero
	}
	result := new(uint256.Int).Mul(gOHMAmount, index)
	result.Div(result, number_1e18)
	return result, nil
}

func balanceTo(ohmAmount, index *uint256.Int) (*uint256.Int, error) {
	if index.IsZero() {
		return nil, ErrIndexZero
	}
	result := new(uint256.Int).Mul(ohmAmount, number_1e18)
	result.Div(result, index)
	return result, nil
}
