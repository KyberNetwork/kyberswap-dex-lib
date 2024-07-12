package ambient

import (
	"fmt"
	"math/big"
)

const (
	LOT_SIZE_BITS uint8 = 10
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

/* @notice Given a positive and negative delta lots value net out the raw liquidity
 *         delta. */
func netLotsOnLiquidity(incrLots *big.Int, decrLots *big.Int) *big.Int {
	// 	 return lotToNetLiq(incrLots) - lotToNetLiq(decrLots);
	tmp := lotToNetLiq(incrLots)
	tmp.Sub(tmp, lotToNetLiq(decrLots))
	return tmp
}

/* @notice Given an amount of lots of liquidity converts to a signed raw liquidity
 *         delta. (Which by definition is always positive.) */
func lotToNetLiq(lots *big.Int) *big.Int {
	//     return int128(lotsToLiquidity(lots));
	return lotsToLiquidity(lots)
}

// /* @notice Given a number of lots of liquidity converts to raw liquidity value. */
var (
	KNOCKOUT_FLAG_MASK, _ = new(big.Int).SetString("0x1", 16)
)

func lotsToLiquidity(lots *big.Int) *big.Int {
	//	    uint96 realLots = lots & ~KNOCKOUT_FLAG_MASK;
	realLots := new(big.Int).And(lots, new(big.Int).Not(KNOCKOUT_FLAG_MASK))

	//	    return uint128(realLots) << LOT_SIZE_BITS;
	return realLots.Lsh(realLots, uint(LOT_SIZE_BITS))
}

// / @notice Add a signed liquidity delta to liquidity and revert if it overflows or underflows
// / @param x The liquidity before change
// / @param y The delta by which liquidity should be changed
// / @return z The liquidity delta
func addDelta(x *big.Int, y *big.Int) *big.Int {
	// if (y < 0) {
	//         require((z = x - uint128(-y)) < x);
	//     }
	if y.Cmp(big0) == -1 {
		z := new(big.Int).Set(x)
		negY := new(big.Int).Neg(y)
		z.Sub(z, negY)
		return z
	}

	// require((z = x + uint128(y)) >= x);
	z := new(big.Int).Add(x, y)

	return z
}

/* @notice Same as minusDelta, but operates on lots of liquidity rather than outright
*         liquiidty. */
func minusLots(x *big.Int, y *big.Int) *big.Int {
	// z = x - y;
	return new(big.Int).Sub(x, y)
}
