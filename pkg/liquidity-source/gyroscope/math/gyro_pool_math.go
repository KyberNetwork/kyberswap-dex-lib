package math

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var GyroPoolMath *gyroPoolMath

type gyroPoolMath struct {
}

func init() {
	GyroPoolMath = &gyroPoolMath{}
}

func _sqrt(input, tolerance *uint256.Int) *uint256.Int {
	if input.Eq(number.Zero) {
		return number.Zero
	}

}
