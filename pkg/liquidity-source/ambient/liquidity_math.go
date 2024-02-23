package ambient

import (
	"fmt"
	"math/big"
)

// @notice Add an unsigned liquidity delta to liquidity and revert if it overflows or underflows
// @param x The liquidity before change
// @param y The delta by which liquidity should be changed
// @return z The liquidity delta
func addLiq(x, y *big.Int) (*big.Int, error) {
	// require((z = x + y) >= x);
	z := new(big.Int).Add(x, y)
	if z.Cmp(x) == -1 {
		return nil, fmt.Errorf("z is smaller than x")
	}

	return z, nil
}
