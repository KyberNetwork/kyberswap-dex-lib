package math

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var SafeCast *safeCast

type safeCast struct{}

func (s *safeCast) ToInt256(value *uint256.Int) (*int256.Int, error) {
	if value.Sign() < 0 {
		return nil, ErrSafeCast
	}
	return (*int256.Int)(value), nil
}

func (s *safeCast) ToUint256(value *int256.Int) (*uint256.Int, error) {
	if value.IsNegative() {
		return nil, ErrSafeCast
	}
	return (*uint256.Int)(value), nil
}
