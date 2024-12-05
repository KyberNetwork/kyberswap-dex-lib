package maverickv1

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L72
func swap(
	state *MaverickPoolState,
	amount *big.Int,
	tokenAIn bool,
	exactOutput bool,
	isForPricing bool,
) (*big.Int, *big.Int, int, error) {
	delta := &Delta{
		DeltaInBinInternal: new(big.Int),
		DeltaInErc:         new(big.Int),
		DeltaOutErc:        new(big.Int),
		Excess:             new(big.Int).Set(amount),
		TokenAIn:           tokenAIn,
		EndSqrtPrice:       new(big.Int),
		ExactOutput:        exactOutput,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		DecrementTick:      false,
		SqrtPriceLimit:     new(big.Int),
		SqrtLowerTickPrice: new(big.Int),
		SqrtUpperTickPrice: new(big.Int),
		SqrtPrice:          new(big.Int),
	}

	var counter = 0
	for delta.Excess.Sign() > 0 {
		newDelta, err := swapTick(delta, state)
		if err != nil {
			return nil, nil, 0, err
		}
		combine(delta, newDelta)

		// We can not do too much iteration. This variable chosen
		// as reasonable threshold
		counter += 1
		if isForPricing && counter > MaxSwapIterationCalculation {
			return zeroBI, zeroBI, counter, nil
		}
	}
	var amountIn = delta.DeltaInErc
	var amountOut = delta.DeltaOutErc

	return amountIn, amountOut, counter, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L121
func swapTick(delta *Delta, state *MaverickPoolState) (*Delta, error) {
	var activeTick = new(big.Int)
	if delta.DecrementTick {
		activeTick.Sub(state.ActiveTick, bignumber.One)
	} else {
		activeTick.Set(state.ActiveTick)
	}

	var active = getKindsAtTick(state.BinMap, state.BinMapHex, activeTick)

	if active.Word.Sign() == 0 {
		activeTick = nextActive(state.BinMap, state.BinMapHex, activeTick, delta.TokenAIn, state.minBinMapIndex,
			state.maxBinMapIndex)
	}

	var currentReserveA, currentReserveB, currentLiquidity *big.Int
	var currentBins []Bin
	var err error

	oldActiveTick := new(big.Int).Set(activeTick)
	currentReserveA, currentReserveB, delta.SqrtPrice, currentLiquidity, currentBins, err = currentTickLiquidity(activeTick,
		state)
	if err != nil {
		return nil, err
	}

	delta.SqrtLowerTickPrice, err = tickPrice(state.TickSpacing, activeTick)
	if err != nil {
		return nil, err
	}
	delta.SqrtUpperTickPrice, err = tickPrice(state.TickSpacing, new(big.Int).Add(activeTick, bignumber.One))
	if err != nil {
		return nil, err
	}

	pastMaxPrice(delta)
	if delta.SwappedToMaxPrice {
		noSwapReset(delta)
		return delta, nil
	}
	state.ActiveTick = activeTick

	var newDelta *Delta

	var limitInBin = (delta.TokenAIn &&
		delta.SqrtPriceLimit.Cmp(delta.SqrtPrice) >= 0 &&
		delta.SqrtPriceLimit.Cmp(delta.SqrtUpperTickPrice) <= 0) ||
		(!delta.TokenAIn &&
			delta.SqrtPriceLimit.Cmp(delta.SqrtPrice) <= 0 &&
			delta.SqrtPriceLimit.Cmp(delta.SqrtLowerTickPrice) >= 0)

	if delta.ExactOutput {
		newDelta, err = computeSwapExactOut(
			delta.SqrtPrice,
			currentLiquidity,
			currentReserveA,
			currentReserveB,
			delta.Excess,
			delta.TokenAIn,
			state,
		)
		if err != nil {
			return nil, err
		}
	} else {
		var edgePrice = delta.SqrtPriceLimit
		if !limitInBin {
			edgePrice = sqrtEdgePrice(delta)
		}
		newDelta, err = computeSwapExactIn(
			edgePrice,
			delta.SqrtPrice,
			currentLiquidity,
			currentReserveA,
			currentReserveB,
			delta.Excess,
			limitInBin,
			delta.TokenAIn,
			state,
		)
		if err != nil {
			return nil, err
		}
	}

	for i := range currentBins {
		var thisBinAmount = currentBins[i].ReserveA
		if currentReserveB.Sign() > 0 {
			thisBinAmount = currentBins[i].ReserveB
		}

		var totalAmount = currentReserveA
		if currentReserveB.Sign() > 0 {
			totalAmount = currentReserveB
		}

		if err := adjustAB(&currentBins[i], newDelta, thisBinAmount, totalAmount, oldActiveTick, state); err != nil {
			return nil, err
		}
	}

	if newDelta.Excess.Sign() != 0 {
		if newDelta.TokenAIn {
			state.ActiveTick = state.ActiveTick.Add(state.ActiveTick, bignumber.One)
		}
		newDelta.DecrementTick = !delta.TokenAIn
	}

	return newDelta, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L336
func computeSwapExactOut(
	sqrtPrice, liquidity, reserveA, reserveB, amountOut *big.Int,
	tokenAIn bool,
	state *MaverickPoolState,
) (*Delta, error) {
	amountOutAvailable := new(big.Int)
	if tokenAIn {
		amountOutAvailable.Set(reserveB)
	} else {
		amountOutAvailable.Set(reserveA)
	}

	swapped := amountOutAvailable.Cmp(amountOut) <= 0
	delta := &Delta{
		DeltaInBinInternal: new(big.Int),
		DeltaInErc:         new(big.Int),
		DeltaOutErc:        new(big.Int),
		Excess:             new(big.Int),
		TokenAIn:           tokenAIn,
		EndSqrtPrice:       new(big.Int),
		ExactOutput:        true,
		DecrementTick:      false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		SqrtPriceLimit:     new(big.Int),
		SqrtLowerTickPrice: new(big.Int),
		SqrtUpperTickPrice: new(big.Int),
		SqrtPrice:          new(big.Int),
	}

	delta.DeltaOutErc.Set(min(amountOut, amountOutAvailable))

	var binAmountIn *big.Int
	var err error

	tmpB := sqrtPrice
	tmpC, err := inv(sqrtPrice)
	if err != nil {
		return nil, err
	}
	if !tokenAIn {
		tmpB, tmpC = tmpC, tmpB
	}
	deltaOutErcDivLiquidity, err := div(delta.DeltaOutErc, liquidity)
	if err != nil {
		return nil, err
	}
	tmpC = tmpC.Sub(tmpC, deltaOutErcDivLiquidity)

	binAmountIn, err = mulDiv(
		delta.DeltaOutErc,
		tmpB,
		tmpC,
		true,
	)
	if err != nil {
		return nil, err
	}

	delta.EndSqrtPrice = tmpC
	if tokenAIn {
		delta.EndSqrtPrice, err = inv(delta.EndSqrtPrice)
		if err != nil {
			return nil, err
		}
	}

	feeBasis, err := mulDiv(
		binAmountIn,
		state.Fee,
		new(big.Int).Sub(bignumber.BONE, state.Fee),
		true,
	)
	if err != nil {
		return nil, err
	}
	delta.DeltaInErc = delta.DeltaInErc.Add(binAmountIn, feeBasis)
	delta.DeltaInBinInternal, err = amountToBin(delta.DeltaInErc, feeBasis, state)
	if err != nil {
		return nil, err
	}
	if swapped {
		delta.Excess = clip(amountOut, delta.DeltaOutErc)
	} else {
		delta.Excess = new(big.Int)
	}

	return delta, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L261
func computeSwapExactIn(
	sqrtEdgePrice, sqrtPrice, liquidity, reserveA, reserveB, amountIn *big.Int,
	limitInBin, tokenAIn bool,
	state *MaverickPoolState,
) (*Delta, error) {
	var binAmountIn *big.Int
	var err error

	if tokenAIn {
		binAmountIn, err = deltaAmount(liquidity, sqrtPrice, sqrtEdgePrice, true)
		if err != nil {
			return nil, err
		}
	} else {
		binAmountIn, err = deltaAmount(liquidity, sqrtEdgePrice, sqrtPrice, false)
		if err != nil {
			return nil, err
		}
	}

	delta := &Delta{
		DeltaInBinInternal: new(big.Int),
		DeltaInErc:         new(big.Int),
		DeltaOutErc:        new(big.Int),
		Excess:             new(big.Int),
		TokenAIn:           tokenAIn,
		EndSqrtPrice:       new(big.Int),
		ExactOutput:        false,
		DecrementTick:      false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		SqrtPriceLimit:     new(big.Int),
		SqrtLowerTickPrice: new(big.Int),
		SqrtUpperTickPrice: new(big.Int),
		SqrtPrice:          new(big.Int),
	}

	var feeBasis *big.Int

	tmp, err := mul(amountIn, new(big.Int).Sub(bignumber.BONE, state.Fee))
	if err != nil {
		return nil, err
	}
	if tmp.Cmp(binAmountIn) >= 0 {
		feeBasis, err = mulDiv(binAmountIn, state.Fee, new(big.Int).Sub(bignumber.BONE, state.Fee), true)
		if err != nil {
			return nil, err
		}
		delta.DeltaInErc = delta.DeltaInErc.Add(binAmountIn, feeBasis)
		if limitInBin {
			delta.SwappedToMaxPrice = true
		} else {
			delta.EndSqrtPrice = sqrtEdgePrice
			delta.DeltaOutErc = reserveA
			if tokenAIn {
				delta.DeltaOutErc = reserveB
			}
			delta.Excess = clip(amountIn, delta.DeltaInErc)
		}
	} else {
		binAmountIn, err = mul(amountIn, new(big.Int).Sub(bignumber.BONE, state.Fee))
		if err != nil {
			return nil, err
		}
		delta.DeltaInErc = delta.DeltaInErc.Set(amountIn)
		feeBasis = new(big.Int).Sub(delta.DeltaInErc, binAmountIn)
	}
	delta.DeltaInBinInternal, err = amountToBin(delta.DeltaInErc, feeBasis, state)
	if err != nil {
		return nil, err
	}
	if delta.Excess.Sign() != 0 || liquidity.Sign() == 0 {
		return delta, nil
	}

	tmpReserve := reserveA
	if tokenAIn {
		tmpReserve = reserveB
	}

	tmpB := new(big.Int).Set(sqrtPrice)
	tmpC, err := inv(sqrtPrice)
	if err != nil {
		return nil, err
	}
	if tokenAIn {
		tmpB, tmpC = tmpC, tmpB
	}
	tmpDiv, err := div(binAmountIn, liquidity)
	if err != nil {
		return nil, err
	}
	tmpEndSqrtPrice := tmpC.Add(tmpDiv, tmpC)
	tmpMulDiv, err := mulDiv(binAmountIn, tmpB, tmpC, false)
	if err != nil {
		return nil, err
	}

	delta.DeltaOutErc = min(tmpReserve, tmpMulDiv)
	delta.EndSqrtPrice = tmpEndSqrtPrice

	if !tokenAIn {
		delta.EndSqrtPrice, err = inv(delta.EndSqrtPrice)
		if err != nil {
			return nil, err
		}
	}

	return delta, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L252
func amountToBin(deltaInErc, feeBases *big.Int, state *MaverickPoolState) (*big.Int, error) {
	if state.ProtocolFeeRatio.Sign() != 0 {
		tmp := bignumber.TenPowInt(15)
		tmpMul, err := mul(feeBases, tmp.Mul(state.ProtocolFeeRatio, tmp))
		if err != nil {
			return nil, err
		}
		return clip(deltaInErc, tmpMul.Add(tmpMul, bignumber.One)), nil
	}

	return deltaInErc, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L240
func deltaAmount(liquidity, lowerSqrtPrice, upperSqrtPrice *big.Int, isA bool) (*big.Int, error) {
	if isA {
		res, err := mul(liquidity, new(big.Int).Sub(upperSqrtPrice, lowerSqrtPrice))
		if err != nil {
			return nil, err
		}

		return res, nil
	} else {
		liquidityDivLower, err := div(liquidity, lowerSqrtPrice)
		if err != nil {
			return nil, err
		}
		liquidityDivUpper, err := div(liquidity, upperSqrtPrice)
		if err != nil {
			return nil, err
		}
		res := new(big.Int).Sub(liquidityDivLower, liquidityDivUpper)

		return res, nil
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L45
func combine(delta, newDelta *Delta) {
	if !delta.SkipCombine {
		delta.DeltaInBinInternal = delta.DeltaInBinInternal.Add(delta.DeltaInBinInternal, newDelta.DeltaInBinInternal)
		delta.DeltaInErc = delta.DeltaInErc.Add(delta.DeltaInErc, newDelta.DeltaInErc)
		delta.DeltaOutErc = delta.DeltaOutErc.Add(delta.DeltaOutErc, newDelta.DeltaOutErc)
	}
	delta.Excess = newDelta.Excess
	delta.DecrementTick = newDelta.DecrementTick
	delta.EndSqrtPrice = newDelta.EndSqrtPrice
	delta.SwappedToMaxPrice = newDelta.SwappedToMaxPrice
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L390C3-L390C23
func currentTickLiquidity(activeTick *big.Int, state *MaverickPoolState) (*big.Int, *big.Int, *big.Int, *big.Int, []Bin,
	error) {
	var active = getKindsAtTick(state.BinMap, state.BinMapHex, activeTick)

	var reserveA = new(big.Int)
	var reserveB = new(big.Int)
	var bins = make([]Bin, 0)

	for i := 0; i < 4; i++ {
		bigI := big.NewInt(int64(i))
		if active.Word.Bit(i) > 0 {
			var binID = state.BinPositions[activeTick.String()][bigI.String()]
			if binID != nil && binID.Sign() > 0 {
				bin := state.Bins[binID.String()]
				reserveA = reserveA.Add(reserveA, bin.ReserveA)
				reserveB = reserveB.Add(reserveB, bin.ReserveB)
				bins = append(bins, bin)
			}
		}
	}

	var sqrtLowerTickPrice, sqrtUpperTickPrice *big.Int
	var err error
	sqrtLowerTickPrice, err = tickPrice(state.TickSpacing, activeTick)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	sqrtUpperTickPrice, err = tickPrice(state.TickSpacing, new(big.Int).Add(activeTick, bignumber.One))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	sqrtPrice, liquidity, err := getTickSqrtPriceAndL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return reserveA, reserveB, sqrtPrice, liquidity, bins, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L511C3-L511C12
func tickPrice(tickSpacing *big.Int, activeTick *big.Int) (*big.Int, error) {
	var tick big.Int
	tick.Abs(activeTick).Mul(&tick, tickSpacing)
	if tick.Cmp(MaxTickBI) > 0 {
		return nil, ErrLargerThanMaxTick
	}

	ratio := new(big.Int)
	if tick.Bit(0) != 0 {
		ratio.Set(CompareConst1)
	} else {
		ratio.Set(CompareConst2)
	}

	if tick.Bit(1) != 0 {
		ratio.Mul(ratio, MulConst1)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(2) != 0 {
		ratio.Mul(ratio, MulConst2)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(3) != 0 {
		ratio.Mul(ratio, MulConst3)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(4) != 0 {
		ratio.Mul(ratio, MulConst4)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(5) != 0 {
		ratio.Mul(ratio, MulConst5)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(6) != 0 {
		ratio.Mul(ratio, MulConst6)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(7) != 0 {
		ratio.Mul(ratio, MulConst7)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(8) != 0 {
		ratio.Mul(ratio, MulConst8)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(9) != 0 {
		ratio.Mul(ratio, MulConst9)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(10) != 0 {
		ratio.Mul(ratio, MulConst10)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(11) != 0 {
		ratio.Mul(ratio, MulConst11)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(12) != 0 {
		ratio.Mul(ratio, MulConst12)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(13) != 0 {
		ratio.Mul(ratio, MulConst13)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(14) != 0 {
		ratio.Mul(ratio, MulConst14)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(15) != 0 {
		ratio.Mul(ratio, MulConst15)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(16) != 0 {
		ratio.Mul(ratio, MulConst16)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(17) != 0 {
		ratio.Mul(ratio, MulConst17)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(18) != 0 {
		ratio.Mul(ratio, MulConst18)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(19) != 0 {
		ratio.Mul(ratio, MulConst19)
		ratio.Rsh(ratio, 128)
	}
	if tick.Bit(20) != 0 {
		ratio.Mul(ratio, MulConst20)
		ratio.Rsh(ratio, 128)
	}

	if activeTick.Sign() > 0 {
		var tmp big.Int
		ratio.Div(
			tmp.Sub(
				tmp.Lsh(tmp.SetInt64(1), 256),
				bignumber.One),
			ratio,
		)
	}

	ratio.Mul(ratio, bignumber.BONE)
	ratio.Rsh(ratio, 128)

	return ratio, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L56
func pastMaxPrice(delta *Delta) {
	cmp := delta.SqrtPriceLimit.Cmp(delta.SqrtPrice)
	tmpBool := cmp >= 0
	if delta.TokenAIn {
		tmpBool = cmp <= 0
	}
	delta.SwappedToMaxPrice = delta.SqrtPriceLimit.Sign() != 0 && tmpBool
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L66
func noSwapReset(delta *Delta) {
	delta.Excess = new(big.Int)
	delta.SkipCombine = true
	delta.EndSqrtPrice = delta.SqrtPrice
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L63
func sqrtEdgePrice(delta *Delta) *big.Int {
	if delta.TokenAIn {
		return delta.SqrtUpperTickPrice
	}
	return delta.SqrtLowerTickPrice
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L210
func adjustAB(bin *Bin, delta *Delta, thisBinAmount, totalAmount, activeTick *big.Int, state *MaverickPoolState) error {
	deltaIn, err := mulDiv(delta.DeltaInBinInternal, thisBinAmount, totalAmount, false)
	if err != nil {
		return err
	}
	var deltaOut *big.Int
	if delta.Excess.Sign() != 0 {
		deltaOut = new(big.Int)
	} else {
		if deltaOut, err = mulDiv(delta.DeltaOutErc, thisBinAmount, totalAmount, true); err != nil {
			return err
		} else if deltaOut.Sign() == 0 {
			return ErrInvalidDeltaOut
		}
	}

	if delta.TokenAIn {
		bin.ReserveA = bin.ReserveA.Add(bin.ReserveA, deltaIn)

		if delta.Excess.Sign() <= 0 {
			bin.ReserveB = clip(bin.ReserveB, deltaOut)
		} else {
			bin.ReserveB = new(big.Int)
		}
	} else {
		bin.ReserveB = bin.ReserveB.Add(bin.ReserveB, deltaIn)

		if delta.Excess.Sign() <= 0 {
			bin.ReserveA = clip(bin.ReserveA, deltaOut)
		} else {
			bin.ReserveA = new(big.Int)
		}
	}

	// Custom code to update state.Bin
	var active = getKindsAtTick(state.BinMap, state.BinMapHex, activeTick)
	if active.Word.Bit(int(bin.Kind.Int64())) > 0 {
		binID := state.BinPositions[activeTick.String()][bin.Kind.String()]
		if binID != nil && binID.Sign() > 0 {
			state.Bins[binID.String()] = *bin
		}
	}

	return nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L484C3-L484C23
func getTickSqrtPriceAndL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *big.Int) (*big.Int, *big.Int,
	error) {
	liquidity, err := getTickL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice)
	if err != nil {
		return nil, nil, err
	}
	if liquidity.Sign() < 0 {
		return nil, nil, ErrInvalidLiquidity
	}

	if reserveA.Sign() == 0 {
		return sqrtLowerTickPrice, liquidity, nil
	}

	if reserveB.Sign() == 0 {
		return sqrtUpperTickPrice, liquidity, nil
	}

	qo, err := mul(liquidity, sqrtLowerTickPrice)
	if err != nil {
		return nil, nil, err
	}
	bo, err := div(liquidity, sqrtUpperTickPrice)
	if err != nil {
		return nil, nil, err
	}

	tmpDiv, err := div(qo.Add(reserveA, qo), bo.Add(reserveB, bo))
	if err != nil {
		return nil, nil, err
	}
	sqrtPrice := sqrt(tmpDiv)

	return sqrtPrice, liquidity, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L434
func getTickL(
	reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *big.Int,
) (*big.Int, error) {
	precisionBump := uint(0)

	var tmp big.Int
	if tmp.Rsh(reserveA, 60).Sign() == 0 && tmp.Rsh(reserveB, 60).Sign() == 0 {
		precisionBump = 40
		reserveA = new(big.Int).Lsh(reserveA, precisionBump)
		reserveB = new(big.Int).Lsh(reserveB, precisionBump)
	}

	if reserveA.Sign() == 0 || reserveB.Sign() == 0 {
		tmpDiv, err := div(reserveA, sqrtUpperTickPrice)
		if err != nil {
			return nil, err
		}
		tmpMul, err := mul(reserveB, sqrtLowerTickPrice)
		if err != nil {
			return nil, err
		}
		b := tmpDiv.Add(tmpDiv, tmpMul)

		result, err := mulDiv(b, sqrtUpperTickPrice, tmpMul.Sub(sqrtUpperTickPrice, sqrtLowerTickPrice), false)
		if err != nil {
			return nil, err
		}
		result = result.Rsh(result, precisionBump)

		return result, nil
	} else {
		tmpDiv, err := div(reserveA, sqrtUpperTickPrice)
		if err != nil {
			return nil, err
		}
		tmpMul, err := mul(reserveB, sqrtLowerTickPrice)
		if err != nil {
			return nil, err
		}
		b := tmpDiv.Add(tmpDiv, tmpMul)
		b = b.Rsh(b, 1)

		diff := tmpMul.Sub(sqrtUpperTickPrice, sqrtLowerTickPrice)

		tmpMulReserveBReserveA, err := mul(reserveB, reserveA)
		if err != nil {
			return nil, err
		}
		tmpMulDiv, err := mulDiv(tmpMulReserveBReserveA, diff, sqrtUpperTickPrice, false)
		if err != nil {
			return nil, err
		}
		tmpMulBB, err := mul(b, b)
		if err != nil {
			return nil, err
		}
		result, err := mulDiv(b.Add(b, sqrt(tmpMulBB.Add(tmpMulBB, tmpMulDiv))), sqrtUpperTickPrice, diff, false)
		if err != nil {
			return nil, err
		}
		result = result.Rsh(result, precisionBump)

		return result, nil
	}
}

// ------------- maverick bin map -----------------------

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L139
func nextActive(binMap map[string]*big.Int, binMapHex map[string]*big.Int, tick *big.Int, isRight bool,
	minBinMapIndex, maxBinMapIndex *big.Int) *big.Int {
	var refTick, shift, tack, subIndex, nextTick *big.Int

	refTick = new(big.Int)
	if isRight {
		refTick = refTick.Add(tick, bignumber.One)
	} else {
		refTick = refTick.Sub(tick, bignumber.One)
	}
	var tmp big.Int
	var offset, mapIndex = getMapPointer(tmp.Mul(refTick, Kinds))
	if isRight {
		shift = offset
		tack = big.NewInt(1)
		nextTick = big.NewInt(1000000000)
	} else {
		shift = tmp.Sub(WordSize, offset)
		tack = big.NewInt(-1)
		nextTick = big.NewInt(-1000000000)
	}

	binMapCount := len(binMap)
	binMapHexCount := len(binMapHex)
	if binMapHexCount == 0 && binMapCount == 0 {
		// we can never find anything so don't bother going to the expensive loop
		return nextTick
	}

	// we'll use a single bigInt for nextWord through the loop, instead of allocating every times
	var nextWord big.Int
	var bin *big.Int
	for i := 0; i < 4000; i++ {
		if binMapHexCount > 0 {
			bin = binMapHex[mapIndex.Text(16)]
		} else {
			bin = binMap[mapIndex.String()]
		}
		if bin == nil {
			nextWord.Set(zeroBI)
		} else {
			nextWord.Set(bin)
		}
		if i == 0 {
			// after the 1st iteration `shift` will be set to 0, so we can skip this to avoid over allocating
			if isRight {
				nextWord.Rsh(&nextWord, uint(shift.Uint64()))
			} else {
				nextWord.Lsh(&nextWord, uint(shift.Uint64()))
			}
		}
		nextWord.And(&nextWord, BitMask)
		if nextWord.Sign() != 0 {
			break
		}
		if i == 0 {
			shift = zeroBI
		}
		// mapIndex already get allocated within `getMapPointer`, so we can safely overwrite it here
		mapIndex = mapIndex.Add(mapIndex, tack)
		// mapIndex will always either increase or decrease, not both
		// so we can check against min/max index and terminate early
		if tack.Sign() > 0 && mapIndex.Cmp(maxBinMapIndex) > 0 {
			break
		}
		if tack.Sign() < 0 && mapIndex.Cmp(minBinMapIndex) < 0 {
			break
		}
	}

	if nextWord.Sign() != 0 {
		subIndex = new(big.Int)
		if isRight {
			subIndex = subIndex.Add(lsb(&nextWord), shift)
		} else {
			subIndex = subIndex.Sub(msb(&nextWord), shift)
		}
		posFirst := tmp.Add(tmp.Mul(mapIndex, WordSize), subIndex)
		pos := new(big.Int).Set(posFirst)
		if posFirst.Sign() < 0 {
			pos = pos.Add(pos, bignumber.One)
		}
		nextTick = pos.Quo(pos,
			Kinds) // use truncated div here instead of Euclidean div (-1427/4 = -356 instead of -357)
		if posFirst.Sign() < 0 {
			nextTick = nextTick.Sub(nextTick, bignumber.One)
		}
	}

	return nextTick
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L126
func getKindsAtTick(binMap map[string]*big.Int, binMapHex map[string]*big.Int, tick *big.Int) Active {
	offset, mapIndex := getMapPointer(new(big.Int).Mul(tick, Kinds))
	var subMap *big.Int
	if len(binMapHex) > 0 {
		subMap = binMapHex[mapIndex.Text(16)]
	} else {
		subMap = binMap[mapIndex.String()]
	}
	if subMap == nil {
		subMap = big.NewInt(0)
	}
	presentBits := new(big.Int).And(new(big.Int).Rsh(subMap, uint(offset.Int64())), Mask)

	return Active{
		Word: presentBits,
		Tick: new(big.Int).Set(tick),
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L97
func getMapPointer(tick *big.Int) (*big.Int, *big.Int) {
	offset := new(big.Int).And(tick, OffsetMask)
	mapIndex := new(big.Int).Rsh(tick, 8)

	return offset, mapIndex
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L54
func lsb(x *big.Int) *big.Int {
	r := big.NewInt(255)
	// bigint in typescript is pass by value. So I do not want this function change the input X
	var tmp big.Int
	tmpX := new(big.Int).Set(x)

	if tmp.And(tmpX, bignumber.NewBig("0xffffffffffffffffffffffffffffffff")).Sign() > 0 {
		r.Sub(r, big.NewInt(128))
	} else {
		tmpX.Rsh(tmpX, 128)
	}
	if tmp.And(tmpX, bignumber.NewBig("0xffffffffffffffff")).Sign() > 0 {
		r.Sub(r, big.NewInt(64))
	} else {
		tmpX.Rsh(tmpX, 64)
	}
	if tmp.And(tmpX, bignumber.NewBig("0xffffffff")).Sign() > 0 {
		r.Sub(r, big.NewInt(32))
	} else {
		tmpX.Rsh(tmpX, 32)
	}
	if tmp.And(tmpX, bignumber.NewBig("0xffff")).Sign() > 0 {
		r.Sub(r, big.NewInt(16))
	} else {
		tmpX.Rsh(tmpX, 16)
	}
	if tmp.And(tmpX, bignumber.NewBig("0xff")).Sign() > 0 {
		r.Sub(r, big.NewInt(8))
	} else {
		tmpX.Rsh(tmpX, 8)
	}
	if tmp.And(tmpX, bignumber.NewBig("0xf")).Sign() > 0 {
		r.Sub(r, big.NewInt(4))
	} else {
		tmpX.Rsh(tmpX, 4)
	}
	if tmp.And(tmpX, bignumber.NewBig("0x3")).Sign() > 0 {
		r.Sub(r, big.NewInt(2))
	} else {
		tmpX.Rsh(tmpX, 2)
	}
	if tmp.And(tmpX, bignumber.NewBig("0x1")).Sign() > 0 {
		r.Sub(r, big.NewInt(1))
	}

	return r
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L17
func msb(x *big.Int) *big.Int {
	r := new(big.Int)

	// bigint in typescript is pass by value. So I do not want this function change the input X
	tmpX := new(big.Int).Set(x)

	if tmpX.Cmp(bignumber.NewBig("0x100000000000000000000000000000000")) >= 0 {
		tmpX.Rsh(tmpX, 128)
		r.Add(r, big.NewInt(128))
	}
	if tmpX.Cmp(bignumber.NewBig("0x10000000000000000")) >= 0 {
		tmpX.Rsh(tmpX, 64)
		r.Add(r, big.NewInt(64))
	}
	if tmpX.Cmp(bignumber.NewBig("0x100000000")) >= 0 {
		tmpX.Rsh(tmpX, 32)
		r.Add(r, big.NewInt(32))
	}
	if tmpX.Cmp(bignumber.NewBig("0x10000")) >= 0 {
		tmpX.Rsh(tmpX, 16)
		r.Add(r, big.NewInt(16))
	}
	if tmpX.Cmp(bignumber.NewBig("0x100")) >= 0 {
		tmpX.Rsh(tmpX, 8)
		r.Add(r, big.NewInt(8))
	}
	if tmpX.Cmp(bignumber.NewBig("0x10")) >= 0 {
		tmpX.Rsh(tmpX, 4)
		r.Add(r, big.NewInt(4))
	}
	if tmpX.Cmp(bignumber.NewBig("0x4")) >= 0 {
		tmpX.Rsh(tmpX, 2)
		r.Add(r, big.NewInt(2))
	}
	if tmpX.Cmp(bignumber.NewBig("0x2")) >= 0 {
		r.Add(r, big.NewInt(1))
	}

	return r
}

// ------------- maverick basic math --------------------

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L131
func mulDiv(a, b, c *big.Int, ceil bool) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)
	if product.Sign() == 0 {
		return product, nil
	}
	var tmp big.Int
	if ceil && tmp.Mod(product, c).Sign() != 0 {
		return product.Add(product.Div(product, c), bignumber.One), nil
	}
	return product.Div(product, c), nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L46
func clip(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return new(big.Int)
	}
	return new(big.Int).Sub(x, y)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L42
func inv(a *big.Int) (*big.Int, error) {
	return div(One, a)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L68
func div(a, b *big.Int) (*big.Int, error) {
	return divDownFixed(a, b)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L120
func divDownFixed(a, b *big.Int) (*big.Int, error) {
	if b.Sign() != 0 {
		if a.Sign() == 0 {
			return new(big.Int), nil
		} else {
			aInflated := new(big.Int).Mul(a, One)
			return aInflated.Div(aInflated, b), nil
		}
	}

	return nil, ErrDividedByZero
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L72
func sqrt(x *big.Int) *big.Int {
	if x.Sign() == 0 {
		return new(big.Int)
	}

	x = new(big.Int).Mul(x, Unit)

	// Set the initial guess to the least power of two that is greater than or equal to sqrt(x).
	var xAux big.Int
	xAux.Set(x)
	result := big.NewInt(1)
	if xAux.Cmp(XAuxConst64) >= 0 {
		xAux.Rsh(&xAux, 128)
		result.Lsh(result, 64)
	}
	if xAux.Cmp(XAuxConst32) >= 0 {
		xAux.Rsh(&xAux, 64)
		result.Lsh(result, 32)
	}
	if xAux.Cmp(XAuxConst16) >= 0 {
		xAux.Rsh(&xAux, 32)
		result.Lsh(result, 16)
	}
	if xAux.Cmp(XAuxConst8) >= 0 {
		xAux.Rsh(&xAux, 16)
		result.Lsh(result, 8)
	}
	if xAux.Cmp(XAuxConst4) >= 0 {
		xAux.Rsh(&xAux, 8)
		result.Lsh(result, 4)
	}
	if xAux.Cmp(XAuxConst2) >= 0 {
		xAux.Rsh(&xAux, 4)
		result.Lsh(result, 2)
	}
	if xAux.Cmp(XAuxConst1) >= 0 {
		result.Lsh(result, 1)
	}

	for i := 0; i < 7; i++ { // Seven iterations should be enough
		xAux.Div(x, result)
		result.Add(result, &xAux)
		result.Rsh(result, 1)
	}

	roundedDownResult := xAux.Div(x, result)
	if result.Cmp(roundedDownResult) >= 0 {
		return roundedDownResult
	}
	return result
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L14
func min(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L34
func mul(a, b *big.Int) (*big.Int, error) {
	var tmp big.Int
	if tmp.Mod(
		tmp.Mul(abs(a), abs(b)),
		Unit,
	).Cmp(MulConst49_17) > 0 {
		return mulUpFixed(a, b)
	} else {
		return sMulDownFixed(a, b)
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L50
func mulUpFixed(a, b *big.Int) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)
	isNegative := product.Sign() == -1

	if product.Sign() == 0 {
		return product, nil
	} else {
		result := product.Sub(abs(product), bignumber.One)
		result.Div(result, One)
		result.Add(result, bignumber.One)

		if isNegative {
			result.Neg(result)
		}

		return result, nil
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L181
func sMulDownFixed(a, b *big.Int) (*big.Int, error) {
	var product = new(big.Int).Mul(a, b)
	return product.Div(product, One), nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L10
func abs(x *big.Int) *big.Int {
	if x.Sign() < 0 {
		return new(big.Int).Neg(x)
	}
	return x
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L146
func sDivDownFixed(a, b *big.Int) (*big.Int, error) {
	return divDownFixed(a, b)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L158
func sMulUpFixed(a, b *big.Int) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)
	var tmp big.Int
	if a.Sign() == 0 || tmp.Div(product, a).Cmp(b) == 0 {
		if product.Sign() == 0 {
			return product, nil
		}
		return product.Add(
			product.Div(product.Sub(product, bignumber.One), One),
			bignumber.One,
		), nil
	}

	return nil, ErrMulOverflow
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-v1-pool.ts#L369
func scaleFromAmount(amount *big.Int, decimals uint8) (*big.Int, error) {
	if decimals == 18 {
		return amount, nil
	}
	var scalingFactor big.Int
	if decimals > 18 {
		scalingFactor.Mul(
			bignumber.BONE,
			bignumber.TenPowInt(decimals-18),
		)
		return sDivDownFixed(amount, &scalingFactor)
	} else {
		scalingFactor.Mul(
			bignumber.BONE,
			bignumber.TenPowInt(18-decimals),
		)
		return sMulUpFixed(amount, &scalingFactor)
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-v1-pool.ts#L382
func ScaleToAmount(amount *big.Int, decimals uint8) (*big.Int, error) {
	if decimals == 18 {
		return amount, nil
	}
	var scalingFactor big.Int
	if decimals > 18 {
		scalingFactor.Mul(
			bignumber.BONE,
			bignumber.TenPowInt(decimals-18),
		)
		return sMulUpFixed(amount, &scalingFactor)
	} else {
		scalingFactor.Mul(
			bignumber.BONE,
			bignumber.TenPowInt(18-decimals),
		)
		return sDivDownFixed(amount, &scalingFactor)
	}
}
