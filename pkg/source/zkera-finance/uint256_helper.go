package zkerafinance

import (
	"math/big"

	"github.com/holiman/uint256"
)

func u256ToBig(v *uint256.Int) *big.Int {
	if v == nil {
		return nil
	}

	return v.ToBig()
}
