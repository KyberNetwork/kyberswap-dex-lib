package integral

import (
	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

// addDelta adds the signed liquidity delta d, negated when neg, to z in place.
func addDelta(z *uint256.Int, d *int256.Int, neg bool) error {
	var abs uint256.Int
	abs.Abs((*uint256.Int)(d))
	if (d.Sign() >= 0) != neg || d.IsZero() {
		if _, overflow := z.AddOverflow(z, &abs); overflow {
			return ErrLiquidityAdd
		}
		return nil
	}
	if _, underflow := z.SubOverflow(z, &abs); underflow {
		return ErrLiquiditySub
	}
	return nil
}

func ToUInt256(x *int256.Int) (*uint256.Int, error) {
	var res = new(uint256.Int)
	if err := v3Utils.ToUInt256(x, res); err != nil {
		return nil, err
	}

	return res, nil
}
