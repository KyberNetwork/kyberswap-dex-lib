package calc

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

// MulDivFloor performs multiplication first and then division, flooring the result.
func MulDivFloor(a, b, c *uint256.Int) *uint256.Int {
	res, _ := v3Utils.MulDiv(a, b, c)
	return res
}

// MulDivCeil performs multiplication first and then division, ceiling the result.
func MulDivCeil(a, b, c *uint256.Int) *uint256.Int {
	res, _ := v3Utils.MulDivRoundingUp(a, b, c)
	return res
}
