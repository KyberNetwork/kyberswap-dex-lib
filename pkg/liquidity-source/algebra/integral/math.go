package integral

import (
	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

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
