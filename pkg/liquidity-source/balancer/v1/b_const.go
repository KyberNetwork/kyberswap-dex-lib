package balancerv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var BConst *bConst

type bConst struct {
	BONE           *uint256.Int
	MIN_BPOW_BASE  *uint256.Int
	MAX_BPOW_BASE  *uint256.Int
	BPOW_PRECISION *uint256.Int
	MAX_IN_RATIO   *uint256.Int
}

func init() {
	BConst = &bConst{
		BONE:           number.Number_1e18,
		MIN_BPOW_BASE:  number.Number_1,
		MAX_BPOW_BASE:  new(uint256.Int).Sub(new(uint256.Int).Mul(number.Number_2, number.Number_1e18), number.Number_1),
		BPOW_PRECISION: new(uint256.Int).Div(number.Number_1e18, number.TenPow(10)),
		MAX_IN_RATIO:   new(uint256.Int).Div(number.Number_1e18, number.Number_2),
	}
}
