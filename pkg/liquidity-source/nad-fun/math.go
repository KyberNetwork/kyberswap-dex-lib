package nadfun

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

func getAmountOut(amountIn, reserveIn, reserveOut, k *uint256.Int) (*uint256.Int, error) {
	var newReserveIn uint256.Int
	newReserveIn.Add(reserveIn, amountIn)

	var tmp uint256.Int
	tmp.Add(k, &newReserveIn)
	tmp.Sub(&tmp, big256.U1)
	newReserveOut := tmp.Div(&tmp, &newReserveIn)

	amountOut := newReserveOut.Sub(reserveOut, newReserveOut)
	if amountOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	return amountOut, nil
}

func getAmountIn(amountOut, reserveIn, reserveOut, k *uint256.Int) (*uint256.Int, error) {
	var newReserveOut uint256.Int
	newReserveOut.Sub(reserveOut, amountOut)
	if newReserveOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	var tmp uint256.Int
	tmp.Add(k, &newReserveOut)
	tmp.Sub(&tmp, big256.U1)
	newReserveIn := tmp.Div(&tmp, &newReserveOut)

	amountIn := newReserveIn.Sub(newReserveIn, reserveIn)

	return amountIn, nil
}

func getFeeAmount(amount *uint256.Int, protocolFee *uint256.Int) *uint256.Int {
	if protocolFee.IsZero() {
		return uint256.NewInt(0)
	}

	var fee uint256.Int
	fee.MulDivOverflow(amount, protocolFee, FeeDenom)

	return &fee
}

func checkInvariant(virtualNative, virtualToken, k *uint256.Int) error {
	if new(uint256.Int).Mul(virtualNative, virtualToken).Lt(k) {
		return ErrInvariantViolation
	}
	return nil
}
