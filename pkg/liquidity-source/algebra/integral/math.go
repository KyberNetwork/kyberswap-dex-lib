package integral

import (
	"errors"

	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func unsafeDivRoundingUp(x, y *uint256.Int) (*uint256.Int, error) {
	if y.Sign() == 0 {
		return nil, errors.New("division by zero")
	}

	quotient := new(uint256.Int).Div(x, y)

	remainder := new(uint256.Int).Mod(x, y)
	if remainder.Sign() > 0 {
		quotient.Add(quotient, uONE)
	}

	return quotient, nil
}

func addDelta(x *uint256.Int, y *int256.Int) (*uint256.Int, error) {
	uY, err := ToUInt256(y)
	if err != nil {
		return nil, err
	}

	var res *uint256.Int
	if y.Sign() < 0 {
		res := new(uint256.Int).Sub(x, uY.Neg(uY))
		if res.Cmp(x) >= 0 {
			return nil, ErrLiquiditySub
		}
	} else {
		res := new(uint256.Int).Add(x, uY)
		if res.Cmp(x) < 0 {
			return nil, ErrLiquidityAdd
		}
	}

	return res, nil
}

func ToUInt256(x *int256.Int) (*uint256.Int, error) {
	var res = new(uint256.Int)
	if err := v3Utils.ToUInt256(x, res); err != nil {
		return nil, err
	}

	return res, nil
}

func ToInt256(x *uint256.Int) (*int256.Int, error) {
	var res *int256.Int
	if err := v3Utils.ToInt256(x, res); err != nil {
		return nil, err
	}

	return res, nil
}
