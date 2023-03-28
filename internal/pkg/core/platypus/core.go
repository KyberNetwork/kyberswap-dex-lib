package platypus

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// _slippageFunc calculates g(xr,i) or g(xr,j). This function always returns >= 0
// @param [k] K slippage parameter in WAD
// @param [n] N slippage parameter
// @param [c1] C1 slippage parameter in WAD
// @param [xThreshold] xThreshold slippage parameter in WAD
// @param [x] coverage ratio of asset in WAD
// @return The result of price slippage curve
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Core.sol#L33
func _slippageFunc(
	k *big.Int,
	n *big.Int,
	c1 *big.Int,
	xThreshold *big.Int,
	x *big.Int,
) (*big.Int, error) {
	if x.Cmp(xThreshold) < 0 {
		return new(big.Int).Sub(c1, x), nil
	}

	return wdiv(
		k,
		new(big.Int).Div(
			new(big.Int).Mul(
				rpow(
					new(big.Int).Div(
						new(big.Int).Mul(x, RAY),
						WAD,
					),
					n,
				),
				WAD,
			),
			RAY,
		),
	)
}

// _slippage returns slippage
// @param [k] K slippage parameter in WAD
// @param [n] N slippage parameter
// @param [c1] C1 slippage parameter in WAD
// @param [xThreshold] xThreshold slippage parameter in WAD
// @param [cash] cash position of asset in WAD
// @param [cashChange] cashChange of asset in WAD
// @param [addCash] true if we are adding cash, false otherwise
// @return The result of one-sided asset slippage
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Core.sol#L59
func _slippage(
	k *big.Int,
	n *big.Int,
	c1 *big.Int,
	xThreshold *big.Int,
	cash *big.Int,
	liability *big.Int,
	cashChange *big.Int,
	addCash bool,
) (*big.Int, error) {
	covBefore, err := wdiv(cash, liability)
	if err != nil {
		return nil, err
	}

	var covAfter *big.Int

	if addCash {
		covAfter, err = wdiv(new(big.Int).Add(cash, cashChange), liability)
		if err != nil {
			return nil, err
		}
	} else {
		covAfter, err = wdiv(new(big.Int).Sub(cash, cashChange), liability)
		if err != nil {
			return nil, err
		}
	}

	if covBefore.Cmp(covAfter) == 0 {
		return constant.Zero, nil
	}

	slippageBefore, err := _slippageFunc(k, n, c1, xThreshold, covBefore)
	if err != nil {
		return nil, err
	}

	slippageAfter, err := _slippageFunc(k, n, c1, xThreshold, covAfter)
	if err != nil {
		return nil, err
	}

	if covBefore.Cmp(covAfter) > 0 {
		return wdiv(
			new(big.Int).Sub(slippageAfter, slippageBefore),
			new(big.Int).Sub(covBefore, covAfter),
		)
	}

	return wdiv(
		new(big.Int).Sub(slippageBefore, slippageAfter),
		new(big.Int).Sub(covAfter, covBefore),
	)
}

// _swappingSlippage
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Core.sol#L100
func _swappingSlippage(
	si *big.Int,
	sj *big.Int,
) *big.Int {
	return new(big.Int).Sub(
		new(big.Int).Add(
			WAD,
			si,
		),
		sj,
	)
}

// _haircut
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Core.sol#L111
func _haircut(
	amount *big.Int,
	rate *big.Int,
) *big.Int {
	return wmul(amount, rate)
}

// _dividend
// https://github.com/platypus-finance/core/blob/ce7a98d5a12aa54d3f9b31777b6dde8f1f771318/contracts/pool/Core.sol#L121
func _dividend(amount *big.Int, ratio *big.Int) *big.Int {
	return wmul(amount, new(big.Int).Sub(WAD, ratio))
}
