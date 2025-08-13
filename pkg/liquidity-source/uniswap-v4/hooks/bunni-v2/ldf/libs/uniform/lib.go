package uniform

import (
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// DecodeParams decodes the LDF parameters from bytes32
func DecodeParams(
	tickSpacing,
	twapTick int,
	ldfParams [32]byte,
) (
	tickLower,
	tickUpper int,
	shiftMode shiftmode.ShiftMode,
) {
	shiftMode = shiftmode.ShiftMode(ldfParams[0])

	if shiftMode != shiftmode.Static {
		// | shiftMode - 1 byte | offset - 3 bytes | length - 3 bytes |
		offsetRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])
		offset := int(signExtend24to32(offsetRaw))

		lengthRaw := uint32(ldfParams[4])<<16 | uint32(ldfParams[5])<<8 | uint32(ldfParams[6])
		length := int(signExtend24to32(lengthRaw))

		tickLower = math.RoundTickSingle(twapTick+offset, tickSpacing)
		tickUpper = tickLower + length*tickSpacing

		minUsableTick := math.MinUsableTick(tickSpacing)
		maxUsableTick := math.MaxUsableTick(tickSpacing)

		if tickLower < minUsableTick {
			tickLower = minUsableTick
			tickUpper = min(tickLower+length*tickSpacing, maxUsableTick)
		} else if tickUpper > maxUsableTick {
			tickUpper = maxUsableTick
			tickLower = max(tickUpper-length*tickSpacing, minUsableTick)
		}
	} else {
		// | shiftMode - 1 byte | tickLower - 3 bytes | tickUpper - 3 bytes |
		tickLowerRaw := uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])
		tickLower = int(signExtend24to32(tickLowerRaw))

		tickUpperRaw := uint32(ldfParams[4])<<16 | uint32(ldfParams[5])<<8 | uint32(ldfParams[6])
		tickUpper = int(signExtend24to32(tickUpperRaw))
	}

	return
}

// LiquidityDensityX96 computes the liquidity density at a given tick
func LiquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper int) *uint256.Int {
	if roundedTick < tickLower || roundedTick >= tickUpper {
		return uint256.NewInt(0)
	}
	length := (tickUpper - tickLower) / tickSpacing
	res := uint256.NewInt(uint64(length))

	return res.Div(math.Q96, res)
}

// CumulativeAmount0 computes the cumulative amount0
func CumulativeAmount0(
	tickSpacing,
	roundedTick int,
	totalLiquidity *uint256.Int,
	tickLower,
	tickUpper int,
	isCarpet bool,
) (amount0 *uint256.Int, err error) {
	if roundedTick >= tickUpper || tickLower >= tickUpper {
		return uint256.NewInt(0), nil
	}
	if roundedTick < tickLower {
		roundedTick = tickLower
	}

	length := (tickUpper - tickLower) / tickSpacing

	sqrtPriceRoundedTick, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, err
	}
	sqrtPriceTickUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	if isCarpet {
		amount0, err = math.GetAmount0Delta(
			sqrtPriceRoundedTick,
			sqrtPriceTickUpper,
			math.DivUp(totalLiquidity, uint256.NewInt(uint64(length))),
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		return amount0, nil
	} else {
		amount0, err = math.GetAmount0Delta(
			sqrtPriceRoundedTick,
			sqrtPriceTickUpper,
			math.DivUp(math.Q96, uint256.NewInt(uint64(length))),
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		amount0, err = math.FullMulX96Up(totalLiquidity, amount0)
		if err != nil {
			return nil, err
		}

		return amount0, nil
	}
}

// CumulativeAmount1 computes the cumulative amount1
func CumulativeAmount1(tickSpacing, roundedTick int, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (*uint256.Int, error) {
	if roundedTick < tickLower || tickLower >= tickUpper {
		return uint256.NewInt(0), nil
	}
	if roundedTick > tickUpper-tickSpacing {
		roundedTick = tickUpper - tickSpacing
	}

	length := (tickUpper - tickLower) / tickSpacing
	if length <= 0 {
		return uint256.NewInt(0), nil
	}

	sqrtPriceTickLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, err
	}
	sqrtPriceRoundedTickPlusSpacing, err := math.GetSqrtPriceAtTick(roundedTick + tickSpacing)
	if err != nil {
		return nil, err
	}

	if isCarpet {
		amount1, err := math.GetAmount1Delta(
			sqrtPriceTickLower,
			sqrtPriceRoundedTickPlusSpacing,
			math.DivUp(totalLiquidity, uint256.NewInt(uint64(length))),
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		return amount1, nil
	} else {
		amount1, err := math.GetAmount1Delta(
			sqrtPriceTickLower,
			sqrtPriceRoundedTickPlusSpacing,
			math.DivUp(math.Q96, uint256.NewInt(uint64(length))),
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result, err := math.FullMulX96Up(totalLiquidity, amount1)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// InverseCumulativeAmount0 computes the inverse of cumulative amount0
func InverseCumulativeAmount0(tickSpacing int, cumulativeAmount0_, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (bool, int) {
	if cumulativeAmount0_.IsZero() {
		return true, tickUpper
	}

	length := (tickUpper - tickLower) / tickSpacing

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return false, 0
	}
	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return false, 0
	}

	var sqrtPrice *uint256.Int
	if isCarpet {
		sqrtPrice, err = math.GetNextSqrtPriceFromAmount0RoundingUp(
			sqrtPriceUpper,
			math.DivUp(totalLiquidity, uint256.NewInt(uint64(length))),
			cumulativeAmount0_,
			true,
		)
		if err != nil {
			return false, 0
		}
	} else {
		scaledAmount, err := math.FullMulDiv(cumulativeAmount0_, math.Q96, totalLiquidity)
		if err != nil {
			return false, 0
		}

		sqrtPrice, err = math.GetNextSqrtPriceFromAmount0RoundingUp(
			sqrtPriceUpper,
			math.DivUp(math.Q96, uint256.NewInt(uint64(length))),
			scaledAmount,
			true,
		)
		if err != nil {
			return false, 0
		}
	}

	if sqrtPrice.Lt(sqrtPriceLower) {
		return false, 0
	}

	tick, err := math.GetTickAtSqrtPrice(sqrtPrice)
	if err != nil {
		return false, 0
	}

	roundedTick := math.RoundTickSingle(tick, tickSpacing)

	if roundedTick < tickLower || roundedTick > tickUpper {
		return false, 0
	}

	if roundedTick == tickUpper {
		return true, tickUpper - tickSpacing
	}

	return true, roundedTick
}

// InverseCumulativeAmount1 computes the inverse of cumulative amount1
func InverseCumulativeAmount1(tickSpacing int, cumulativeAmount1_, totalLiquidity *uint256.Int, tickLower, tickUpper int, isCarpet bool) (bool, int) {
	if cumulativeAmount1_.IsZero() {
		return true, tickLower - tickSpacing
	}

	length := (tickUpper - tickLower) / tickSpacing

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return false, 0
	}
	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return false, 0
	}

	var sqrtPrice *uint256.Int
	if isCarpet {
		sqrtPrice, err = math.GetNextSqrtPriceFromAmount1RoundingDown(
			sqrtPriceLower,
			math.DivUp(totalLiquidity, uint256.NewInt(uint64(length))),
			cumulativeAmount1_,
			true,
		)
		if err != nil {
			return false, 0
		}
	} else {
		scaledAmount, err := math.FullMulDiv(cumulativeAmount1_, math.Q96, totalLiquidity)
		if err != nil {
			return false, 0
		}

		sqrtPrice, err = math.GetNextSqrtPriceFromAmount1RoundingDown(
			sqrtPriceLower,
			math.DivUp(math.Q96, uint256.NewInt(uint64(length))),
			scaledAmount,
			true,
		)
		if err != nil {
			return false, 0
		}
	}

	if sqrtPrice.Gt(sqrtPriceUpper) {
		return false, 0
	}

	tick, err := math.GetTickAtSqrtPrice(sqrtPrice)
	if err != nil {
		return false, 0
	}

	if tick == tickUpper {
		tick -= 1
	}

	roundedTick := math.RoundTickSingle(tick, tickSpacing)

	if roundedTick < tickLower-tickSpacing || roundedTick >= tickUpper {
		return false, 0
	}

	if roundedTick == tickLower-tickSpacing {
		return true, tickLower
	}

	return true, roundedTick
}

func signExtend24to32(value24 uint32) int32 {
	if value24&0x800000 != 0 {
		value24 |= 0xFF000000
	}
	return int32(value24)
}
