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

// getAmounts writes into amountIn, amountOut and fee what a swap of amountsInLeft takes into the
// bin at activeID, fee included, what it gets out of it, and the fee.
func (b *BinU256) getAmounts(
	parameters *parameters,
	binStep uint16,
	swapForY bool,
	activeID uint32,
	amountsInLeft, amountIn, amountOut, fee *uint256.Int,
) error {
	var price, maxAmountIn, maxFee, totalFee uint256.Int
	if err := getPriceFromID(&price, activeID, binStep); err != nil {
		return err
	}

	binReserveOut := b.getReserveOut(!swapForY)

	var err error
	if swapForY {
		err = shiftDivRoundUp(&maxAmountIn, binReserveOut, scaleOffset, &price)
	} else {
		err = mulShiftRoundUp(&maxAmountIn, binReserveOut, &price, scaleOffset)
	}
	if err != nil {
		return err
	}

	parameters.getTotalFee(binStep, &totalFee)
	if err := getFeeAmount(&maxFee, &maxAmountIn, &totalFee); err != nil {
		return err
	}

	maxAmountIn.Add(&maxAmountIn, &maxFee)

	if amountsInLeft.Cmp(&maxAmountIn) >= 0 {
		fee.Set(&maxFee)
		amountIn.Set(&maxAmountIn)
		amountOut.Set(binReserveOut)
		return nil
	}

	if err := getFeeAmountFrom(fee, amountsInLeft, &totalFee); err != nil {
		return err
	}
	var amountInLessFee uint256.Int
	amountInLessFee.Sub(amountsInLeft, fee)

	if swapForY {
		err = mulShiftRoundDown(amountOut, &amountInLessFee, &price, scaleOffset)
	} else {
		err = shiftDivRoundDown(amountOut, &amountInLessFee, scaleOffset, &price)
	}
	if err != nil {
		return err
	}

	if amountOut.Cmp(binReserveOut) > 0 {
		amountOut.Set(binReserveOut)
	}
	amountIn.Set(amountsInLeft)

	return nil
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

// binReserveChanges is what a swap put into one bin and took out of it: token Y in and X out when
// SwapForX, token X in and Y out otherwise.
type binReserveChanges struct {
	BinID     uint32
	SwapForX  bool
	AmountIn  uint256.Int
	AmountOut uint256.Int
}

// newBinReserveChanges copies amountIn and amountOut, which the swap goes on to reuse.
func newBinReserveChanges(
	binID uint32,
	swapForX bool,
	amountIn *uint256.Int,
	amountOut *uint256.Int,
) binReserveChanges {
	return binReserveChanges{BinID: binID, SwapForX: swapForX, AmountIn: *amountIn, AmountOut: *amountOut}
}

// changeX and changeY are what the swap did to the bin's reserves of X and of Y, modulo 2^256.
func (c *binReserveChanges) changeX(z *uint256.Int) *uint256.Int {
	if c.SwapForX {
		return z.Neg(&c.AmountOut)
	}
	return z.Set(&c.AmountIn)
}

func (c *binReserveChanges) changeY(z *uint256.Int) *uint256.Int {
	if c.SwapForX {
		return z.Set(&c.AmountIn)
	}
	return z.Neg(&c.AmountOut)
}
