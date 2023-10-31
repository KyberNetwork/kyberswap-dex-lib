package liquiditybookv21

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

type bin struct {
	ID          uint32   `json:"id"`
	ReserveX    *big.Int `json:"reserveX"`
	ReserveY    *big.Int `json:"reserveY"`
	TotalSupply *big.Int `json:"totalSupply"`
}

func (b *bin) isEmptyForSwap(swapForX bool) bool {
	zero := integer.Zero()
	if swapForX {
		return b.ReserveX.Cmp(zero) == 0
	}
	return b.ReserveY.Cmp(zero) == 0
}

func (b *bin) isEmpty() bool {
	zero := integer.Zero()
	return b.ReserveX.Cmp(zero) == 0 && b.ReserveY.Cmp(zero) == 0
}

func (b *bin) getAmounts(
	parameters *parameters,
	binStep uint16,
	swapForY bool,
	activeID uint32,
	amountsInLeft *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {
	price, err := getPriceFromID(activeID, binStep)
	if err != nil {
		return nil, nil, nil, err
	}

	binReserveOut := b.getReserveOut(!swapForY)

	var maxAmountIn *big.Int
	if swapForY {
		if maxAmountIn, err = shiftDivRoundUp(binReserveOut, scaleOffset, price); err != nil {
			return nil, nil, nil, err
		}
	} else {
		if maxAmountIn, err = mulShiftRoundUp(binReserveOut, price, scaleOffset); err != nil {
			return nil, nil, nil, err
		}
	}

	totalFee := parameters.getTotalFee(binStep)
	maxFee, err := getFeeAmount(maxAmountIn, totalFee)
	if err != nil {
		return nil, nil, nil, err
	}

	maxAmountIn = new(big.Int).Add(maxAmountIn, maxFee)

	amountIn128 := amountsInLeft
	var fee128, amountOut128 *big.Int

	if amountIn128.Cmp(maxAmountIn) >= 0 {
		fee128 = maxFee
		amountIn128 = maxAmountIn
		amountOut128 = binReserveOut
	} else {
		var err error
		fee128, err = getFeeAmountFrom(amountIn128, totalFee)
		if err != nil {
			return nil, nil, nil, err
		}
		amountIn := new(big.Int).Sub(amountIn128, fee128)

		if swapForY {
			amountOut128, err = mulShiftRoundDown(amountIn, price, scaleOffset)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			amountOut128, err = shiftDivRoundDown(amountIn, scaleOffset, price)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		if amountOut128.Cmp(binReserveOut) > 0 {
			amountOut128 = binReserveOut
		}
	}

	return amountIn128, amountOut128, fee128, nil
}

func (b *bin) getReserveOut(swapForX bool) *big.Int {
	if swapForX {
		return b.ReserveX
	}
	return b.ReserveY
}

type binReserveChanges struct {
	BinID      uint32   `json:"binId"`
	AmountXIn  *big.Int `json:"amountInX"`
	AmountXOut *big.Int `json:"amountOutX"`
	AmountYIn  *big.Int `json:"amountInY"`
	AmountYOut *big.Int `json:"amountOutY"`
}

func newBinReserveChanges(
	binID uint32,
	swapForX bool,
	amountIn *big.Int,
	amountOut *big.Int,
) binReserveChanges {
	if swapForX {
		return binReserveChanges{
			BinID:      binID,
			AmountXIn:  integer.Zero(),
			AmountXOut: amountOut,
			AmountYIn:  amountIn,
			AmountYOut: integer.Zero(),
		}
	}
	return binReserveChanges{
		BinID:      binID,
		AmountXIn:  amountIn,
		AmountXOut: integer.Zero(),
		AmountYIn:  integer.Zero(),
		AmountYOut: amountOut,
	}
}
