package integral

import (
	"github.com/holiman/uint256"
)

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapOracleV3.sol#L141
func (p *PoolSimulator) tradeY(yAfter, xBefore, yBefore *uint256.Int) (*uint256.Int, error) {
	yAfterInt := ToInt256(yAfter)
	xBeforeInt := ToInt256(xBefore)
	yBeforeInt := ToInt256(yBefore)
	averagePriceInt := ToInt256(p.AveragePrice)

	xTradedInt := MulInt256(SubInt256(yAfterInt, yBeforeInt), p.DecimalsConverter)

	xAfterInt := SubInt256(xBeforeInt, NegFloorDiv(xTradedInt, averagePriceInt))

	if xAfterInt.Cmp(ZERO) < 0 {
		return nil, ErrT028
	}

	return ToUint256(xAfterInt), nil
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapOracleV3.sol#L121
func (p *PoolSimulator) tradeX(xAfter, xBefore, yBefore *uint256.Int) (*uint256.Int, error) {
	xAfterInt := ToInt256(xAfter)
	xBeforeInt := ToInt256(xBefore)
	yBeforeInt := ToInt256(yBefore)
	averagePriceInt := ToInt256(p.AveragePrice)

	yTradedInt := MulInt256(SubInt256(xAfterInt, xBeforeInt), averagePriceInt)

	yAfterInt := SubInt256(yBeforeInt, NegFloorDiv(yTradedInt, p.DecimalsConverter))

	if yAfterInt.Cmp(ZERO) < 0 {
		return nil, ErrT027
	}

	return ToUint256(yAfterInt), nil
}
