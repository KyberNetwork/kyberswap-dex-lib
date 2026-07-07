package calc

import (
	"github.com/holiman/uint256"
)

func Min(x, y *uint256.Int) *uint256.Int {
	if x.Cmp(y) <= 0 {
		return x
	}
	return y
}
