package skypsm

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// CeilDiv = (x - 1) / y + 1
func CeilDiv(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return nil, number.ErrDivByZero
	}

	if x.IsZero() {
		return number.Zero, nil
	}

	var res uint256.Int
	res.Sub(x, number.Number_1).Div(&res, y).
		Add(&res, number.Number_1)

	return &res, nil
}
