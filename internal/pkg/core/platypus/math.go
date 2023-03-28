package platypus

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// wmul
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/libraries/DSMath.sol#L25
func wmul(
	x *big.Int,
	y *big.Int,
) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(x, y),
			new(big.Int).Div(WAD, constant.Two),
		),
		WAD,
	)
}

// wdiv
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/libraries/DSMath.sol#L30
func wdiv(
	x *big.Int,
	y *big.Int,
) (*big.Int, error) {
	// this check doesn't have in the original smart contract, but we need it here to avoid division by zero
	if y == nil || y.Cmp(constant.Zero) == 0 {
		return nil, ErrDivisionByZero
	}

	return new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(x, WAD),
			new(big.Int).Div(y, constant.Two),
		),
		y,
	), nil
}

// rpow exponentiation by squaring
// https://github.com/platypus-finance/core/ blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/libraries/DSMath.sol#L53
func rpow(
	x *big.Int,
	n *big.Int,
) *big.Int {
	var z *big.Int

	if new(big.Int).Mod(n, constant.Two).Cmp(constant.Zero) != 0 {
		z = x
	} else {
		z = RAY
	}

	for n = new(big.Int).Div(n, constant.Two); n.Cmp(constant.Zero) != 0; n = new(big.Int).Div(n, constant.Two) {
		x = rmul(x, x)

		if new(big.Int).Mod(n, constant.Two).Cmp(constant.Zero) != 0 {
			z = rmul(z, x)
		}
	}

	return z
}

// rmul
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/libraries/DSMath.sol#L66
func rmul(
	x *big.Int,
	y *big.Int,
) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(x, y),
			new(big.Int).Div(RAY, constant.Two),
		),
		RAY,
	)
}
