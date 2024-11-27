package deltaswapv1

import (
	"github.com/holiman/uint256"
)

func Max(a *uint256.Int, b *uint256.Int) *uint256.Int {
	if a.Gt(b) {
		return a
	}
	return b
}
