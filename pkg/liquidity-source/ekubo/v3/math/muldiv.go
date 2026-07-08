package math

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func MulDivOverflow(x, y, d *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if d.IsZero() {
		return nil, ErrDivZero
	}
	return lo.Ternary(roundUp, big256.MulDivUp, big256.MulDivDown)(new(uint256.Int), x, y, d), nil
}

func div(x, y *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if y.IsZero() {
		return nil, ErrDivZero
	}

	quotient, remainder := new(uint256.Int).DivMod(x, y, new(uint256.Int))
	if roundUp && !remainder.IsZero() {
		quotient.AddUint64(quotient, 1)
	}

	return quotient, nil
}
