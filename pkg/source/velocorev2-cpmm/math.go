package velocorev2cpmm

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocorev2-cpmm/sd59x18"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func ceilDiv(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(zero) == 0 {
		return zero
	}
	a = new(big.Int).Sub(a, bignumber.One)
	return new(big.Int).Add(new(big.Int).Div(a, b), bignumber.One)
}

func ceilDivUnsafe(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Div(new(big.Int).Sub(new(big.Int).Add(a, b), big.NewInt(1)), b)
}

func rpow(x *big.Int, n *big.Int, base *big.Int) *big.Int {
	if x.Cmp(zero) == 0 {
		if n.Cmp(zero) == 0 {
			return base
		}
		return zero
	}

	z := x
	if new(big.Int).Mod(x, two).Cmp(zero) == 0 {
		z = base
	}

	half := new(big.Int).Div(base, two)
	for n = new(big.Int).Div(n, two); n.Cmp(zero) > 0; n = new(big.Int).Div(n, two) {
		xx := new(big.Int).Mul(x, x)

		// skip the following check:
		// if iszero(eq(div(xx, x), x)) { revert(0, 0) }

		xxRound := new(big.Int).Add(xx, half)

		// skip the following check:
		// if lt(xxRound, xx) { revert(0, 0) }

		x := new(big.Int).Div(xxRound, base)
		if new(big.Int).Mod(n, two).Cmp(zero) == 0 {
			zx := new(big.Int).Mul(z, x)

			// skip the following check:
			// if and(iszero(iszero(x)), iszero(eq(div(zx, x), z))) { revert(0, 0) }

			zxRound := new(big.Int).Add(zx, half)

			// skip the following check:
			// if lt(zxRound, zx) { revert(0, 0) }

			z = new(big.Int).Div(zxRound, base)
		}
	}

	return nil
}

func powReciprocal(x1e18 *big.Int, n *big.Int) (*big.Int, *big.Int, error) {
	if n.Cmp(bigint0) == 0 || x1e18.Cmp(bigint1e18) == 0 {
		return bigint1e18, bigint1e18, nil
	}

	if n.Cmp(bigint1) == 0 {
		return x1e18, x1e18, nil
	}

	if n.Cmp(new(big.Int).Neg(bigint1)) == 0 {
		bigint1e18Square := new(big.Int).Mul(bigint1e18, bigint1e18)
		return new(big.Int).Div(bigint1e18Square, x1e18), ceilDivUnsafe(bigint1e18Square, x1e18), nil
	}

	if n.Cmp(bigint2) == 0 {
		x1e18Mul1e18 := new(big.Int).Mul(x1e18, bigint1e18)
		s := new(big.Int).Sqrt(x1e18Mul1e18)
		if new(big.Int).Mul(s, s).Cmp(x1e18Mul1e18) < 0 {
			return s, new(big.Int).Add(s, bigint1), nil
		}
		return s, s, nil
	}

	if n.Cmp(new(big.Int).Neg(bigint2)) == 0 {
		x1e18Mul1e18 := new(big.Int).Mul(x1e18, bigint1e18)
		s := new(big.Int).Sqrt(x1e18Mul1e18)
		ss := s
		if new(big.Int).Mul(s, s).Cmp(x1e18Mul1e18) < 0 {
			ss = new(big.Int).Add(s, bigint1)
		}
		square1e18 := new(big.Int).Mul(bigint1e18, bigint1e18)
		return new(big.Int).Div(square1e18, ss), ceilDiv(square1e18, s), nil
	}

	nSD59x18, err := sd59x18.ConvertToSD59x18(n)
	if err != nil {
		return nil, nil, err
	}
	bigint1e18SD59x18, err := sd59x18.ConvertToSD59x18(bigint1e18)
	if err != nil {
		return nil, nil, err
	}
	bigint1e18DivN, err := sd59x18.Div(bigint1e18SD59x18, nSD59x18)
	if err != nil {
		return nil, nil, err
	}
	x1e18SD59x18, err := sd59x18.ConvertToSD59x18(x1e18)
	if err != nil {
		return nil, nil, err
	}
	rawSD59x18, err := sd59x18.Pow(x1e18SD59x18, bigint1e18DivN)
	if err != nil {
		return nil, nil, err
	}

	var raw *big.Int = rawSD59x18
	maxError := new(big.Int).Add(
		ceilDiv(
			new(big.Int).Mul(raw, bigint1e4),
			bigint1e18,
		),
		bigint1,
	)

	ret0 := bigint0
	if raw.Cmp(maxError) >= 0 {
		ret0 = new(big.Int).Sub(raw, maxError)
	}
	ret1 := new(big.Int).Add(raw, maxError)
	return ret0, ret1, nil
}
