package libv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	DecimalMathOne = new(uint256.Int).Set(number.Number_1e18)
)

// DecimalMathMul https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DecimalMath.sol#L25
func DecimalMathMul(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDiv(SafeMul(target, d), number.Number_1e18)
}

// DecimalMathMulCeil https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DecimalMath.sol#L29
func DecimalMathMulCeil(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDivCeil(SafeMul(target, d), number.Number_1e18)
}

// DecimalMathDivFloor https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DecimalMath.sol#L33
func DecimalMathDivFloor(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDiv(SafeMul(target, number.Number_1e18), d)
}

// DecimalMathDivCeil https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/lib/DecimalMath.sol#L37
func DecimalMathDivCeil(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDivCeil(SafeMul(target, number.Number_1e18), d)
}
