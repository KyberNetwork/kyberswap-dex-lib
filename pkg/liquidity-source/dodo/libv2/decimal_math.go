package libv2

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	DecimalMathOne  = new(uint256.Int).Set(number.Number_1e18)
	DecimalMathOne2 = new(uint256.Int).Set(new(uint256.Int).Exp(number.Number_1e18, number.Number_2))
)

// DecimalMathMulFloor https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DecimalMath.sol#L25
func DecimalMathMulFloor(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDiv(SafeMul(target, d), number.Number_1e18)
}

// DecimalMathMulCeil https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DecimalMath.sol#L29
func DecimalMathMulCeil(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDivCeil(SafeMul(target, d), number.Number_1e18)
}

// DecimalMathDivFloor https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DecimalMath.sol#L33
func DecimalMathDivFloor(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDiv(SafeMul(target, number.Number_1e18), d)
}

// DecimalMathDivCeil https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DecimalMath.sol#L37
func DecimalMathDivCeil(target *uint256.Int, d *uint256.Int) *uint256.Int {
	return SafeDivCeil(SafeMul(target, number.Number_1e18), d)
}

// DecimalMathReciprocalFloor https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/DecimalMath.sol#L41
func DecimalMathReciprocalFloor(target *uint256.Int) *uint256.Int {
	return SafeDiv(DecimalMathOne2, target)
}
