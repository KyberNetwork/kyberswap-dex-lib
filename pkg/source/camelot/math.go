package camelot

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// _f = x0 * (y * y / 1e18 * y / 1e18) / 1e18 + (x0 * x0 / 1e18 * x0 / 1e18) * y / 1e18
func _f(x0 *big.Int, y *big.Int) *big.Int {
	return new(big.Int).Add(
		new(big.Int).Div(
			new(big.Int).Mul(
				x0,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(new(big.Int).Mul(y, y), bignumber.BONE),
						y,
					),
					bignumber.BONE,
				),
			),
			bignumber.BONE,
		),
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Div(new(big.Int).Mul(x0, x0), bignumber.BONE),
						x0,
					),
					bignumber.BONE,
				),
				y,
			),
			bignumber.BONE,
		),
	)
}

// _d = 3 * x0 * (y * y / 1e18) / 1e18 + (x0 * x0 / 1e18 * x0 / 1e18);
func _d(x0 *big.Int, y *big.Int) *big.Int {
	return new(big.Int).Add(
		new(big.Int).Div(
			new(big.Int).Mul(
				bignumber.Three,
				new(big.Int).Mul(
					x0,
					new(big.Int).Div(new(big.Int).Mul(y, y), bignumber.BONE),
				),
			),
			bignumber.BONE,
		),
		new(big.Int).Mul(
			new(big.Int).Div(new(big.Int).Mul(x0, x0), bignumber.BONE),
			new(big.Int).Div(x0, bignumber.BONE),
		),
	)
}
