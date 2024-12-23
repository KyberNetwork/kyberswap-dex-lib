package integral

import (
	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func unsafeDivRoundingUp(x, y *uint256.Int) *uint256.Int {
	if y.IsZero() {
		panic("division by zero")
	}

	quotient, remainder := new(uint256.Int).DivMod(x, y, new(uint256.Int))
	if remainder.Sign() > 0 {
		quotient.AddUint64(quotient, 1)
	}

	return quotient
}

func addDelta(x *uint256.Int, y *int256.Int) (*uint256.Int, error) {
	if y.Sign() >= 0 {
		uY, err := ToUInt256(y)
		if err != nil {
			return nil, err
		}
		res := new(uint256.Int).Add(x, uY)
		if res.Cmp(x) < 0 {
			return nil, ErrLiquidityAdd
		}
		return res, nil
	}

	uY, err := ToUInt256(new(int256.Int).Neg(y))
	if err != nil {
		return nil, err
	}

	res := new(uint256.Int).Sub(x, uY)
	if res.Cmp(x) >= 0 {
		return nil, ErrLiquiditySub
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
	var res = new(int256.Int)
	if err := v3Utils.ToInt256(x, res); err != nil {
		return nil, err
	}

	return res, nil
}
