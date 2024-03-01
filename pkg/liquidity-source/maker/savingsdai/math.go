package savingsdai

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

var (
	ErrOverflow = errors.New("overflow")
)

func rpow(x, n, base *uint256.Int) (*uint256.Int, error) {
	if x.IsZero() {
		if n.IsZero() {
			return base, nil
		}

		return number.Zero, nil
	}

	z := x
	if new(uint256.Int).Mod(n, number.Number_2).IsZero() {
		z = base
	}

	half := new(uint256.Int).Div(base, number.Number_2)
	for i := new(uint256.Int).Div(n, number.Number_2); i.Gt(number.Zero); i = new(uint256.Int).Div(i, number.Number_2) {
		xx := new(uint256.Int).Mul(x, x)

		if !new(uint256.Int).Div(xx, x).Eq(x) {
			return nil, ErrOverflow
		}

		xxRound := new(uint256.Int).Add(xx, half)
		if xxRound.Lt(xx) {
			return nil, ErrOverflow
		}

		x = new(uint256.Int).Div(xxRound, base)
		if !new(uint256.Int).Mod(i, number.Number_2).IsZero() {
			zx := new(uint256.Int).Mul(z, x)

			if !x.IsZero() && !new(uint256.Int).Div(zx, x).Eq(z) {
				return nil, ErrOverflow
			}

			zxRound := new(uint256.Int).Add(zx, half)
			if zxRound.Lt(zx) {
				return nil, ErrOverflow
			}

			z = new(uint256.Int).Div(zxRound, base)
		}
	}

	return z, nil
}

func rmul(x, y *uint256.Int) (*uint256.Int, error) {
	z, overflow := new(uint256.Int).MulOverflow(x, y)
	if overflow {
		return nil, ErrOverflow
	}
	return z, nil
}
