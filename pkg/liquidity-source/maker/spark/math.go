package spark

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
			return base.Clone(), nil
		}

		return number.Zero.Clone(), nil
	}

	var (
		z       uint256.Int
		xx      uint256.Int
		xxRound uint256.Int
		zx      uint256.Int
		zxRound uint256.Int
		i       uint256.Int
		temp    uint256.Int
	)

	if temp.Mod(n, number.Number_2).IsZero() {
		z.Set(base)
	}

	half := new(uint256.Int).Div(base, number.Number_2)
	for ; i.Gt(number.Zero); i.Div(&i, number.Number_2) {
		xx.Mul(x, x)

		if !temp.Div(&xx, x).Eq(x) {
			return nil, ErrOverflow
		}

		xxRound.Add(&xx, half)
		if xxRound.Lt(&xx) {
			return nil, ErrOverflow
		}

		x = new(uint256.Int).Div(&xxRound, base)
		if !temp.Mod(&i, number.Number_2).IsZero() {
			zx.Mul(&z, x)

			if !x.IsZero() && !temp.Div(&zx, x).Eq(&z) {
				return nil, ErrOverflow
			}

			zxRound.Add(&zx, half)
			if zxRound.Lt(&zx) {
				return nil, ErrOverflow
			}

			z.Div(&zxRound, base)
		}
	}

	return &z, nil
}

func rmul(x, y *uint256.Int) (*uint256.Int, error) {
	z, overflow := new(uint256.Int).MulOverflow(x, y)
	if overflow {
		return nil, ErrOverflow
	}
	return z.Div(z, RAY), nil
}
