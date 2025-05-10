package bin

import (
	"github.com/holiman/uint256"
)

func GetBinById(bins []Bin, id uint32) (Bin, error) {
	idx, err := FindBinArrIndex(bins, id)
	if err != nil {
		return Bin{}, err
	}

	return bins[idx], nil
}

func GetNextNonEmptyBin(swapForY bool, bins []Bin, id uint32) (uint32, error) {
	if swapForY {
		return FindFirstRight(bins, id)
	}

	return FindFirstLeft(bins, id)
}

func FindFirstRight(bins []Bin, id uint32) (uint32, error) {
	idx, err := FindBinArrIndex(bins, id)
	if err != nil {
		return 0, err
	}
	if idx == 0 {
		return 0, ErrBinIDNotFound
	}
	return bins[idx-1].ID, nil
}

func FindFirstLeft(bins []Bin, id uint32) (uint32, error) {
	idx, err := FindBinArrIndex(bins, id)
	if err != nil {
		return 0, err
	}
	if idx == uint32(len(bins)-1) {
		return 0, ErrBinIDNotFound
	}
	return bins[idx+1].ID, nil
}

func FindBinArrIndex(bins []Bin, binID uint32) (uint32, error) {
	if len(bins) == 0 {
		return 0, ErrBinIDNotFound
	}

	var (
		l = 0
		r = len(bins)
	)

	for r-l > 1 {
		m := (r + l) >> 1
		if bins[m].ID <= binID {
			l = m
		} else {
			r = m
		}
	}

	if bins[l].ID != binID {
		return 0, ErrBinIDNotFound
	}

	return uint32(l), nil
}

type Bin struct {
	ID       uint32       `json:"id"`
	ReserveX *uint256.Int `json:"reserveX"`
	ReserveY *uint256.Int `json:"reserveY"`
}

func (b *Bin) Clone() Bin {
	return Bin{
		ID:       b.ID,
		ReserveX: new(uint256.Int).Set(b.ReserveX),
		ReserveY: new(uint256.Int).Set(b.ReserveY),
	}
}

func (b *Bin) IsEmpty(swapForX bool) bool {
	if swapForX {
		return b.ReserveX.IsZero()
	}

	return b.ReserveY.IsZero()
}

func (b *Bin) GetReserveOut(swapForX bool) *uint256.Int {
	if swapForX {
		return b.ReserveX
	}

	return b.ReserveY
}

func (b *Bin) GetAmountsOut(
	fee *uint256.Int,
	binStep uint16,
	swapForY bool,
	amountsInLeft *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	price, err := getPriceFromID(b.ID, binStep)
	if err != nil {
		return nil, nil, nil, err
	}

	binReserveOut := b.GetReserveOut(!swapForY)

	var maxAmountIn *uint256.Int
	if swapForY {
		if maxAmountIn, err = shiftDivRoundUp(binReserveOut, _SCALE_OFFSET, price); err != nil {
			return nil, nil, nil, err
		}
	} else {
		if maxAmountIn, err = mulShiftRoundUp(binReserveOut, price, _SCALE_OFFSET); err != nil {
			return nil, nil, nil, err
		}
	}

	maxFee := getFeeAmount(maxAmountIn, fee)

	maxAmountIn.Add(maxAmountIn, maxFee)

	amountIn128 := new(uint256.Int).Set(amountsInLeft)
	var feeAmount, amountOut *uint256.Int

	if amountIn128.Cmp(maxAmountIn) >= 0 {
		feeAmount = maxFee
		amountIn128 = maxAmountIn
		amountOut = binReserveOut
	} else {
		feeAmount = getFeeAmountFrom(amountIn128, fee)

		amountIn := new(uint256.Int).Sub(amountIn128, feeAmount)

		if swapForY {
			amountOut, err = mulShiftRoundDown(amountIn, price, _SCALE_OFFSET)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			amountOut, err = shiftDivRoundDown(amountIn, _SCALE_OFFSET, price)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		if amountOut.Gt(binReserveOut) {
			amountOut = binReserveOut
		}
	}

	return amountIn128, amountOut, feeAmount, nil
}

func (b *Bin) GetAmountsIn(
	fee *uint256.Int,
	binStep uint16,
	swapForY bool,
	amountsOutLeft *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	price, err := getPriceFromID(b.ID, binStep)
	if err != nil {
		return nil, nil, nil, err
	}

	binReserveOut := b.GetReserveOut(!swapForY)

	amountOutOfBin := new(uint256.Int).Set(binReserveOut)
	if binReserveOut.Gt(amountsOutLeft) {
		amountOutOfBin = new(uint256.Int).Set(amountsOutLeft)
	}

	var amountInWithoutFee *uint256.Int
	if swapForY {
		if amountInWithoutFee, err = shiftDivRoundUp(amountOutOfBin, _SCALE_OFFSET, price); err != nil {
			return nil, nil, nil, err
		}
	} else {
		if amountInWithoutFee, err = mulShiftRoundUp(amountOutOfBin, price, _SCALE_OFFSET); err != nil {
			return nil, nil, nil, err
		}
	}

	feeAmount := getFeeAmount(amountInWithoutFee, fee)

	amountsInWithFees := amountInWithoutFee.Add(amountInWithoutFee, feeAmount)

	return amountsInWithFees, amountOutOfBin, feeAmount, nil
}

func (b *Bin) GetLiquidity(price *uint256.Int) (*uint256.Int, error) {
	liquidity := uint256.NewInt(0)

	var (
		tmp      uint256.Int
		overflow bool
	)

	if b.ReserveX.Sign() > 0 {
		liquidity, overflow = liquidity.MulOverflow(price, b.ReserveX)
		if overflow {
			return nil, ErrLiquidityOverflow
		}
	}

	if b.ReserveY.Sign() > 0 {
		y := tmp.Lsh(b.ReserveY, _SCALE_OFFSET)

		liquidity, overflow = liquidity.AddOverflow(liquidity, y)
		if overflow {
			return nil, ErrLiquidityOverflow
		}
	}

	return liquidity, nil
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
