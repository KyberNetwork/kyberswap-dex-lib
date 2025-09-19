package math

import (
	"errors"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrSqrtFailed = errors.New("_sqrt FAILED")
)

var GyroPoolMath *gyroPoolMath

type gyroPoolMath struct {
}

// Sqrt
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/libraries/GyroPoolMath.sol#L121
func (l *gyroPoolMath) Sqrt(input *uint256.Int) (*uint256.Int, error) {
	var sqrt uint256.Int
	return sqrt.Sqrt(sqrt.Mul(input, big256.BONE)), nil
}
