package shared

import (
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

func FromBig(v *big.Int, _ int) *uint256.Int {
	r, _ := uint256.FromBig(v)
	return r
}

func FromBigs(v []*big.Int) []*uint256.Int {
	return lo.Map(v, FromBig)
}
