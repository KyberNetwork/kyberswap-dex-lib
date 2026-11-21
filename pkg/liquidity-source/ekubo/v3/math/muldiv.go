package math

import (
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func MulDivOverflow(x, y, d *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if d.IsZero() {
		return nil, ErrDivZero
	}

	var result uint256.Int
	if roundUp {
		if err := utils.MulDivRoundingUpV2(x, y, d, &result); err != nil {
			return nil, ErrMulDivOverflow
		}
	} else if _, overflow := result.MulDivOverflow(x, y, d); overflow {
		return nil, ErrMulDivOverflow
	}

	return &result, nil
}

func div(x, y *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if y.IsZero() {
		return nil, ErrDivZero
	}

	quotient, remainder := new(uint256.Int).DivMod(x, y, new(uint256.Int))
	if roundUp && !remainder.IsZero() {
		quotient.AddUint64(quotient, 1)
	}

	return quotient, nil
}
