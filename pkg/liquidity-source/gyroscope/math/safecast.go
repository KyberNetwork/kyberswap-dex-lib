package math

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var SafeCast *safecast

type safecast struct{}

func init() {
	SafeCast = &safecast{}
}

func (s *safecast) ToInt256(value *uint256.Int) (*int256.Int, error) {
	v, err := int256.FromBig(value.ToBig())
	if err != nil {
		return nil, err
	}
	if v.IsNegative() {
		return nil, ErrSafeCast
	}
	return v, nil
}

func (s *safecast) ToUint256(value *int256.Int) (*uint256.Int, error) {
	if value.IsNegative() {
		return nil, ErrSafeCast
	}
	return uint256.MustFromBig(value.ToBig()), nil
}
