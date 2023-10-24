package liquiditybookv21

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func transformSubgraphBins(
	subgraphBins []binSubgraphResp,
	unitX *big.Float,
	unitY *big.Float,
) ([]bin, error) {
	ret := make([]bin, 0, len(subgraphBins))
	for _, b := range subgraphBins {
		id, err := strconv.ParseUint(b.BinID, 10, 32)
		if err != nil {
			return nil, err
		}

		reserveX, ok := new(big.Float).SetString(b.ReserveX)
		if !ok {
			return nil, ErrInvalidReserve
		}
		reserveXInt, _ := new(big.Float).Mul(reserveX, unitX).Int(nil)

		reserveY, ok := new(big.Float).SetString(b.ReserveY)
		if !ok {
			return nil, ErrInvalidReserve
		}
		reserveYInt, _ := new(big.Float).Mul(reserveY, unitY).Int(nil)

		totalSupply, _ := new(big.Int).SetString(b.TotalSupply, 10)

		ret = append(ret, bin{
			ID:          uint32(id),
			ReserveX:    reserveXInt,
			ReserveY:    reserveYInt,
			TotalSupply: totalSupply,
		})
	}

	return ret, nil
}

func buildQueryGetBins(pairAddress string, skip int) string {
	q := fmt.Sprintf(`
	lbpair(id: "%s") {
		tokenX { decimals }
		tokenY { decimals }
		bins(where: {totalSupply_not: "0"}, orderBy: binId, orderDirection: asc, first: 1000, skip: %d) {
		  binId
		  reserveX
		  reserveY
		  totalSupply
		}
	  }
	  _meta { block { timestamp } }
	`, pairAddress, skip)

	return q
}

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/math/Uint128x128Math.sol#L95
func pow(x *big.Int, y *big.Int) (*big.Int, error) {
	var (
		invert bool
		absY   *big.Int
		result = big.NewInt(0)
	)

	if y.Cmp(bignumber.ZeroBI) == 0 {
		return scale, nil
	}

	absY = new(big.Int).Abs(y)
	if y.Sign() < 0 {
		invert = true
	}

	u, _ := new(big.Int).SetString("100000", 16)
	if absY.Cmp(u) < 0 {
		result = scale
		v, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
		squared := x
		if x.Cmp(v) > 0 {
			not0 := new(big.Int).Sub(
				new(big.Int).Lsh(bignumber.One, 256),
				bignumber.One,
			)
			squared = new(big.Int).Div(not0, squared)
			invert = !invert
		}

		for i := 0x1; i <= 0x80000; i <<= 1 {
			ii := big.NewInt(int64(i))
			if new(big.Int).And(absY, ii).Cmp(bignumber.ZeroBI) != 0 {
				result = new(big.Int).Lsh(
					new(big.Int).Mul(result, squared),
					128,
				)
			}
			if i < 0x80000 {
				squared = new(big.Int).Lsh(
					new(big.Int).Mul(squared, squared),
					128,
				)
			}
		}
	}

	if result.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrPowUnderflow
	}

	if invert {
		v := new(big.Int).Sub(new(big.Int).Lsh(bignumber.One, 256), bignumber.One)
		result = new(big.Int).Div(v, result)
	}

	return result, nil
}

func shiftDivRoundUp(x *big.Int, offset uint8, denominator *big.Int) (*big.Int, error) {
	result, err := shiftDivRoundDown(x, offset, denominator)
	if err != nil {
		return nil, err
	}
	if denominator.Cmp(bignumber.ZeroBI) == 0 {
		return result, nil
	}
	v := new(big.Int).Mod(
		new(big.Int).Mul(x, new(big.Int).Lsh(bignumber.One, uint(offset))),
		denominator,
	)
	if v.Cmp(bignumber.ZeroBI) != 0 {
		result = new(big.Int).Add(result, bignumber.One)
	}

	return result, nil
}

func shiftDivRoundDown(x *big.Int, offset uint8, denominator *big.Int) (*big.Int, error) {
	var (
		prod0, prod1 *big.Int
	)

	prod0 = new(big.Int).Lsh(x, uint(offset))
	prod1 = new(big.Int).Rsh(x, uint(256-int(offset)))

	return _getEndOfDivRoundDown(x, x, denominator, prod0, prod1)
}

func _getEndOfDivRoundDown(
	x *big.Int,
	y *big.Int,
	denominator *big.Int,
	prod0 *big.Int,
	prod1 *big.Int,
) (*big.Int, error) {
	if prod1.Cmp(bignumber.ZeroBI) == 0 {
		return new(big.Int).Div(prod0, denominator), nil
	}

	if prod0.Cmp(denominator) >= 0 {
		return nil, ErrMulDivOverflow
	}

	var remainder *big.Int
	if denominator.Cmp(bignumber.ZeroBI) == 0 {
		remainder = big.NewInt(0)
	} else {
		remainder = new(big.Int).Mod(
			new(big.Int).Mul(x, y),
			denominator,
		)
	}
	prod1 = new(big.Int).Sub(prod1, gt(remainder, prod0))
	prod0 = new(big.Int).Sub(prod0, remainder)

	bitwiseNotDenominator := new(big.Int).Not(denominator)
	lpotdod := new(big.Int).And(denominator, new(big.Int).Add(bitwiseNotDenominator, bignumber.One))
	denominator = new(big.Int).Div(denominator, lpotdod)
	prod0 = new(big.Int).Div(prod0, lpotdod)
	lpotdod = new(big.Int).Add(
		new(big.Int).Div(
			new(big.Int).Sub(
				bignumber.ZeroBI,
				lpotdod,
			),
			lpotdod,
		),
		bignumber.One,
	)

	prod0 = new(big.Int).Or(
		prod0,
		new(big.Int).Mul(prod1, lpotdod),
	)

	inverse := new(big.Int).Mul(new(big.Int).Mul(denominator, denominator), big.NewInt(9))
	for i := 0; i < 6; i++ {
		inverse = new(big.Int).Mul(
			inverse,
			new(big.Int).Sub(
				bignumber.Two,
				new(big.Int).Mul(
					denominator,
					inverse,
				),
			),
		)
	}

	result := new(big.Int).Mul(prod0, inverse)
	return result, nil
}

func gt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return bignumber.One
	}
	return bignumber.ZeroBI
}

func mulShiftRoundUp(x *big.Int, y *big.Int, offset uint8) (*big.Int, error) {
	result, err := mulShiftRoundDown(x, y, offset)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		new(big.Int).Lsh(bignumber.One, uint(offset)),
	)
	if v.Cmp(bignumber.ZeroBI) != 0 {
		result = new(big.Int).Add(result, bignumber.One)
	}
	return result, nil
}

func mulShiftRoundDown(x *big.Int, y *big.Int, offset uint8) (*big.Int, error) {
	prod0, prod1 := _getMulProds(x, y)
	result := big.NewInt(0)
	if prod0.Cmp(bignumber.ZeroBI) != 0 {
		result = new(big.Int).Rsh(prod0, uint(offset))
	}
	if prod1.Cmp(bignumber.ZeroBI) != 0 {
		if prod1.Cmp(new(big.Int).Lsh(bignumber.One, uint(offset))) >= 0 {
			return nil, ErrMulShiftOverflow
		}
		result = new(big.Int).Add(
			result,
			new(big.Int).Lsh(prod1, uint(256-int(offset))),
		)
	}

	return result, nil
}

func _getMulProds(x *big.Int, y *big.Int) (*big.Int, *big.Int) {
	not0 := new(big.Int).Sub(
		new(big.Int).Lsh(bignumber.One, 256),
		bignumber.One,
	)
	mm := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		not0,
	)
	prod0 := new(big.Int).Mul(x, y)
	prod1 := new(big.Int).Sub(
		new(big.Int).Sub(mm, prod0),
		lt(mm, prod0),
	)
	return prod0, prod1
}

func lt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return bignumber.One
	}
	return bignumber.ZeroBI
}

func getFeeAmount(amount *big.Int, totalFee *big.Int) *big.Int {
	denominator := new(big.Int).Sub(precison, totalFee)
	result := new(big.Int).Div(
		new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Mul(amount, totalFee),
				denominator,
			),
			bignumber.One,
		),
		denominator,
	)
	return result
}

func getFeeAmountFrom(amountWithFees *big.Int, totalFee *big.Int) *big.Int {
	result := new(big.Int).Div(
		new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Mul(amountWithFees, totalFee),
				precison,
			),
			bignumber.One,
		),
		precison,
	)
	return result
}
