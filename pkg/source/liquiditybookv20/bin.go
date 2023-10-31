package liquiditybookv20

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type bin struct {
	ID          uint32   `json:"id"`
	ReserveX    *big.Int `json:"reserveX"`
	ReserveY    *big.Int `json:"reserveY"`
	TotalSupply *big.Int `json:"totalSupply"`
}

func (b *bin) isEmptyForSwap(swapForX bool) bool {
	if swapForX {
		return b.ReserveX.Cmp(bignumber.ZeroBI) == 0
	}
	return b.ReserveY.Cmp(bignumber.ZeroBI) == 0
}

func (b *bin) isEmpty() bool {
	return b.ReserveX.Cmp(bignumber.ZeroBI) == 0 && b.ReserveY.Cmp(bignumber.ZeroBI) == 0
}

func (b *bin) getReserveOut(swapForX bool) *big.Int {
	if swapForX {
		return b.ReserveX
	}
	return b.ReserveY
}

func (b *bin) getAmounts(
	fp *feeParameters,
	activeID uint32,
	swapForY bool,
	amountIn *big.Int,
) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	var (
		amountInToBin  *big.Int
		amountOutOfBin *big.Int
		totalFee       *big.Int
		protocolFee    *big.Int
	)

	price, err := getPriceFromID(activeID, fp.BinStep)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	binReserveOut := b.getReserveOut(!swapForY)
	var maxAmountInToBin *big.Int
	if swapForY {
		if maxAmountInToBin, err = shiftDivRoundUp(binReserveOut, scaleOffset, price); err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		if maxAmountInToBin, err = mulShiftRoundUp(binReserveOut, price, scaleOffset); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	fp.updateVolatilityAccumulated(activeID)

	totalFee, protocolFee = fp.getFeeAmountDistribution(fp.getFeeAmount(maxAmountInToBin))

	if new(big.Int).Add(maxAmountInToBin, totalFee).Cmp(amountIn) <= 0 {
		amountInToBin = maxAmountInToBin
		amountOutOfBin = binReserveOut
	} else {
		totalFee, protocolFee = fp.getFeeAmountDistribution(fp.getFeeAmount(amountIn))
		amountInToBin = new(big.Int).Sub(amountIn, totalFee)

		if swapForY {
			amountOutOfBin, err = mulShiftRoundDown(amountInToBin, price, scaleOffset)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		} else {
			amountOutOfBin, err = shiftDivRoundDown(amountInToBin, scaleOffset, price)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}

		if amountOutOfBin.Cmp(binReserveOut) > 0 {
			amountOutOfBin = binReserveOut
		}
	}

	return amountInToBin, amountOutOfBin, totalFee, protocolFee, nil
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
			AmountXIn:  bignumber.ZeroBI,
			AmountXOut: amountOut,
			AmountYIn:  amountIn,
			AmountYOut: bignumber.ZeroBI,
		}
	}
	return binReserveChanges{
		BinID:      binID,
		AmountXIn:  amountIn,
		AmountXOut: bignumber.ZeroBI,
		AmountYIn:  bignumber.ZeroBI,
		AmountYOut: amountOut,
	}
}
