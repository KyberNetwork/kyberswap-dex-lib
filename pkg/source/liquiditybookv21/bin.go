package liquiditybookv21

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Bin struct {
	ID       uint32   `json:"id"`
	ReserveX *big.Int `json:"reserveX"`
	ReserveY *big.Int `json:"reserveY"`
}

type BinU256 struct {
	ID       uint32       `json:"id"`
	ReserveX *uint256.Int `json:"reserveX"`
	ReserveY *uint256.Int `json:"reserveY"`
}

func (b *BinU256) isEmptyForSwap(swapForX bool) bool {
	if swapForX {
		return b.ReserveX.IsZero()
	}
	return b.ReserveY.IsZero()
}

func (b *BinU256) isEmpty() bool {
	return b.ReserveX.IsZero() && b.ReserveY.IsZero()
}

func (b *BinU256) getAmounts(
	parameters *parameters,
	binStep uint16,
	swapForY bool,
	activeID uint32,
	amountsInLeft *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	price, err := getPriceFromID(activeID, binStep)
	if err != nil {
		return nil, nil, nil, err
	}

	binReserveOut := b.getReserveOut(!swapForY)

	var maxAmountIn *uint256.Int
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

	maxAmountIn.Add(maxAmountIn, maxFee)

	amountIn128 := new(uint256.Int).Set(amountsInLeft)
	var fee128, amountOut128 *uint256.Int

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
		amountIn := new(uint256.Int).Sub(amountIn128, fee128)

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

func (b *BinU256) getReserveOut(swapForX bool) *uint256.Int {
	if swapForX {
		return b.ReserveX
	}
	return b.ReserveY
}

// https://github.com/traderjoe-xyz/joe-v2/blob/1297c3822f0605e643155c35948959c0a0d05e17/src/libraries/math/PackedUint128Math.sol#L131
/**
 * @dev Decodes a bytes32 into a uint128 as the first or second uint128
 * @param z The encoded bytes32 as follows:
 * if first:
 * [0 - 128[: x1
 * [128 - 256[: empty
 * else:
 * [0 - 128[: empty
 * [128 - 256[: x2
 * @param first Whether to decode as the first or second uint128
 * @return x The decoded uint128
 */
func (b *BinU256) decode(first bool) *uint256.Int {
	if first {
		return b.ReserveX
	}
	return b.ReserveY

}

type binReserveChanges struct {
	BinID      uint32       `json:"binId"`
	AmountXIn  *uint256.Int `json:"amountInX"`
	AmountXOut *uint256.Int `json:"amountOutX"`
	AmountYIn  *uint256.Int `json:"amountInY"`
	AmountYOut *uint256.Int `json:"amountOutY"`
}

func newBinReserveChanges(
	binID uint32,
	swapForX bool,
	amountIn *uint256.Int,
	amountOut *uint256.Int,
) binReserveChanges {
	if swapForX {
		return binReserveChanges{
			BinID:      binID,
			AmountXIn:  new(uint256.Int),
			AmountXOut: amountOut,
			AmountYIn:  amountIn,
			AmountYOut: new(uint256.Int),
		}
	}
	return binReserveChanges{
		BinID:      binID,
		AmountXIn:  amountIn,
		AmountXOut: new(uint256.Int),
		AmountYIn:  new(uint256.Int),
		AmountYOut: amountOut,
	}
}
