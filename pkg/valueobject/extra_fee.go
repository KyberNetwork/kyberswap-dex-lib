package valueobject

import (
	"math/big"
)

var (
	// BasisPoint is one hundredth of 1 percentage point
	// https://en.wikipedia.org/wiki/Basis_point
	BasisPoint   = big.NewInt(10000)
)
