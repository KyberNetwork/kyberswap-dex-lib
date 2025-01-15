package maverickv1

import (
	"math/bits"

	"github.com/KyberNetwork/kutils"
	"github.com/holiman/uint256"

	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L72
func swap(
	state *MaverickPoolState,
	amount *uint256.Int,
	tokenAIn bool,
	exactOutput bool,
	isForPricing bool,
) (amtIn, amtOut *uint256.Int, ticksSwapped int, err error) {
	delta := &Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int).Set(amount),
		SqrtPriceLimit:     new(uint256.Int),
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int),
		TokenAIn:           tokenAIn,
		ExactOutput:        exactOutput,
	}

	for delta.Excess.Sign() > 0 {
		newDelta, err := swapTick(delta, state)
		if err != nil {
			return nil, nil, 0, err
		}
		combine(delta, newDelta)
		ticksSwapped++

		if isForPricing && ticksSwapped > MaxSwapCalcIterForPricing || ticksSwapped > MaxSwapCalcIter {
			return uZero, uZero, ticksSwapped, nil
		}
	}

	return delta.DeltaInErc, delta.DeltaOutErc, ticksSwapped, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L121
func swapTick(delta *Delta, state *MaverickPoolState) (*Delta, error) {
	activeTick := state.ActiveTick
	if delta.DecrementTick {
		activeTick--
	}

	var active = getKindsAtTick(state.BinMap, activeTick)

	if active.Word == 0 {
		activeTick = nextActive(state.BinMap, activeTick, delta.TokenAIn, state.minBinMapIndex, state.maxBinMapIndex)
	}

	if activeTick > MaxTick || activeTick < -MaxTick {
		return nil, ErrLargerThanMaxTick
	}

	oldActiveTick := activeTick
	currReserveA, currReserveB, currSqrtPrice, currLiq, currBins, err := currentTickLiquidity(activeTick, state)
	if err != nil {
		return nil, err
	}
	delta.SqrtPrice = currSqrtPrice

	delta.SqrtLowerTickPrice, err = tickPrice(state.TickSpacing, activeTick)
	if err != nil {
		return nil, err
	}
	delta.SqrtUpperTickPrice, err = tickPrice(state.TickSpacing, activeTick+1)
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
			currLiq,
			currReserveA,
			currReserveB,
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
			currLiq,
			currReserveA,
			currReserveB,
			delta.Excess,
			limitInBin,
			delta.TokenAIn,
			state,
		)
		if err != nil {
			return nil, err
		}
	}

	for i := range currBins {
		var thisBinAmount = currBins[i].ReserveA
		if currReserveB.Sign() > 0 {
			thisBinAmount = currBins[i].ReserveB
		}

		var totalAmount = currReserveA
		if currReserveB.Sign() > 0 {
			totalAmount = currReserveB
		}

		if err := adjustAB(&currBins[i], newDelta, thisBinAmount, totalAmount, oldActiveTick, state); err != nil {
			return nil, err
		}
	}

	if !newDelta.Excess.IsZero() {
		if newDelta.TokenAIn {
			state.ActiveTick++
		}
		newDelta.DecrementTick = !delta.TokenAIn
	}

	return newDelta, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L336
func computeSwapExactOut(
	sqrtP, liquidity, reserveA, reserveB, amountOut *uint256.Int,
	tokenAIn bool,
	state *MaverickPoolState,
) (*Delta, error) {
	amountOutAvailable := new(uint256.Int)
	if tokenAIn {
		amountOutAvailable.Set(reserveB)
	} else {
		amountOutAvailable.Set(reserveA)
	}

	swapped := amountOutAvailable.Cmp(amountOut) <= 0
	delta := &Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int),
		SqrtPriceLimit:     new(uint256.Int),
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int),
		TokenAIn:           tokenAIn,
		ExactOutput:        true,
		DecrementTick:      false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
	}

	delta.DeltaOutErc.Set(minU(amountOut, amountOutAvailable))

	endSqrtP, err := inv(sqrtP)
	if err != nil {
		return nil, err
	}
	if !tokenAIn {
		sqrtP, endSqrtP = endSqrtP, sqrtP
	}
	deltaOutErcDivLiquidity, err := div(delta.DeltaOutErc, liquidity)
	if err != nil {
		return nil, err
	}
	endSqrtP = endSqrtP.Sub(endSqrtP, deltaOutErcDivLiquidity)

	binAmountIn, err := mulDiv(
		delta.DeltaOutErc,
		sqrtP,
		endSqrtP,
		true,
	)
	if err != nil {
		return nil, err
	}

	feeBasis, err := mulDiv(
		binAmountIn,
		state.Fee,
		new(uint256.Int).Sub(bignumber.BONE, state.Fee),
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
		delta.Excess = new(uint256.Int)
	}

	return delta, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L261
func computeSwapExactIn(
	sqrtEdgePrice, sqrtPrice, liquidity, reserveA, reserveB, amountIn *uint256.Int,
	limitInBin, tokenAIn bool,
	state *MaverickPoolState,
) (*Delta, error) {
	binAmountIn, err := deltaAmount(liquidity, sqrtPrice, sqrtEdgePrice, tokenAIn)
	if err != nil {
		return nil, err
	}

	delta := &Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int),
		SqrtPriceLimit:     new(uint256.Int),
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int),
		TokenAIn:           tokenAIn,
		ExactOutput:        false,
		DecrementTick:      false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
	}

	var feeBasis *uint256.Int

	tmp, err := mul(amountIn, new(uint256.Int).Sub(bignumber.BONE, state.Fee))
	if err != nil {
		return nil, err
	}
	if tmp.Cmp(binAmountIn) >= 0 {
		feeBasis, err = mulDiv(binAmountIn, state.Fee, new(uint256.Int).Sub(bignumber.BONE, state.Fee), true)
		if err != nil {
			return nil, err
		}
		delta.DeltaInErc = delta.DeltaInErc.Add(binAmountIn, feeBasis)
		if limitInBin {
			delta.SwappedToMaxPrice = true
		} else {
			delta.DeltaOutErc = reserveA
			if tokenAIn {
				delta.DeltaOutErc = reserveB
			}
			delta.Excess = clip(amountIn, delta.DeltaInErc)
		}
	} else {
		binAmountIn, err = mul(amountIn, new(uint256.Int).Sub(bignumber.BONE, state.Fee))
		if err != nil {
			return nil, err
		}
		delta.DeltaInErc = delta.DeltaInErc.Set(amountIn)
		feeBasis = new(uint256.Int).Sub(delta.DeltaInErc, binAmountIn)
	}
	delta.DeltaInBinInternal, err = amountToBin(delta.DeltaInErc, feeBasis, state)
	if err != nil {
		return nil, err
	}
	if !delta.Excess.IsZero() || liquidity.IsZero() {
		return delta, nil
	}

	tmpReserve := reserveA
	if tokenAIn {
		tmpReserve = reserveB
	}

	sqrtP := new(uint256.Int).Set(sqrtPrice)
	endSqrtP, err := inv(sqrtPrice)
	if err != nil {
		return nil, err
	}
	if tokenAIn {
		sqrtP, endSqrtP = endSqrtP, sqrtP
	}
	tmpDiv, err := div(binAmountIn, liquidity)
	if err != nil {
		return nil, err
	}
	endSqrtP = endSqrtP.Add(tmpDiv, endSqrtP)
	tmpMulDiv, err := mulDiv(binAmountIn, sqrtP, endSqrtP, false)
	if err != nil {
		return nil, err
	}

	delta.DeltaOutErc = minU(tmpReserve, tmpMulDiv)

	return delta, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L252
func amountToBin(deltaInErc, feeBases *uint256.Int, state *MaverickPoolState) (*uint256.Int, error) {
	if !state.ProtocolFeeRatio.IsZero() {
		tmp := bignumber.TenPowInt(15)
		tmpMul, err := mul(feeBases, tmp.Mul(state.ProtocolFeeRatio, tmp))
		if err != nil {
			return nil, err
		}
		return clip(deltaInErc, tmpMul.AddUint64(tmpMul, 1)), nil
	}

	return deltaInErc, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L240
func deltaAmount(liquidity, sqrtPrice, sqrtEdgePrice *uint256.Int, isA bool) (*uint256.Int, error) {
	if isA {
		return mul(liquidity, new(uint256.Int).Sub(sqrtEdgePrice, sqrtPrice))
	}

	liquidityDivLower, err := div(liquidity, sqrtEdgePrice)
	if err != nil {
		return nil, err
	}
	liquidityDivUpper, err := div(liquidity, sqrtPrice)
	if err != nil {
		return nil, err
	}
	return new(uint256.Int).Sub(liquidityDivLower, liquidityDivUpper), nil
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
	delta.SwappedToMaxPrice = newDelta.SwappedToMaxPrice
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L390C3-L390C23
func currentTickLiquidity(activeTick int32,
	state *MaverickPoolState) (reserveA, reserveB, sqrtPrice, liquidity *uint256.Int, bins []Bin, err error) {
	active := getKindsAtTick(state.BinMap, activeTick)
	reserveA, reserveB = new(uint256.Int), new(uint256.Int)

	for i := uint8(0); i < Kinds; i++ {
		if active.Word&(1<<i) != 0 {
			if binID, ok := state.BinPositions[activeTick][i]; ok {
				bin := state.Bins[binID]
				reserveA = reserveA.Add(reserveA, bin.ReserveA)
				reserveB = reserveB.Add(reserveB, bin.ReserveB)
				bins = append(bins, bin)
			}
		}
	}

	var lowerSqrtP, upperSqrtP *uint256.Int
	if lowerSqrtP, err = tickPrice(state.TickSpacing, activeTick); err != nil {
		return nil, nil, nil, nil, nil, err
	} else if upperSqrtP, err = tickPrice(state.TickSpacing, activeTick+1); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	sqrtPrice, liquidity, err = getTickSqrtPAndLiq(reserveA, reserveB, lowerSqrtP, upperSqrtP)
	return reserveA, reserveB, sqrtPrice, liquidity, bins, err
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L511C3-L511C12
func tickPrice(tickSpacing int32, activeTick int32) (*uint256.Int, error) {
	tick := kutils.Abs(activeTick) * tickSpacing
	if tick > MaxTick {
		return nil, ErrLargerThanMaxTick
	}

	ratio := new(uint256.Int)
	if tick&1 != 0 {
		ratio.Set(CompareConst1)
	} else {
		ratio.Set(CompareConst2)
	}
	if tick&0x2 != 0 {
		ratio.Mul(ratio, MulConst1).
			Rsh(ratio, 128)
	}
	if tick&0x4 != 0 {
		ratio.Mul(ratio, MulConst2).
			Rsh(ratio, 128)
	}
	if tick&0x8 != 0 {
		ratio.Mul(ratio, MulConst3).
			Rsh(ratio, 128)
	}
	if tick&0x10 != 0 {
		ratio.Mul(ratio, MulConst4).
			Rsh(ratio, 128)
	}
	if tick&0x20 != 0 {
		ratio.Mul(ratio, MulConst5).
			Rsh(ratio, 128)
	}
	if tick&0x40 != 0 {
		ratio.Mul(ratio, MulConst6).
			Rsh(ratio, 128)
	}
	if tick&0x80 != 0 {
		ratio.Mul(ratio, MulConst7).
			Rsh(ratio, 128)
	}
	if tick&0x100 != 0 {
		ratio.Mul(ratio, MulConst8).
			Rsh(ratio, 128)
	}
	if tick&0x200 != 0 {
		ratio.Mul(ratio, MulConst9).
			Rsh(ratio, 128)
	}
	if tick&0x400 != 0 {
		ratio.Mul(ratio, MulConst10).
			Rsh(ratio, 128)
	}
	if tick&0x800 != 0 {
		ratio.Mul(ratio, MulConst11).
			Rsh(ratio, 128)
	}
	if tick&0x1000 != 0 {
		ratio.Mul(ratio, MulConst12).
			Rsh(ratio, 128)
	}
	if tick&0x2000 != 0 {
		ratio.Mul(ratio, MulConst13).
			Rsh(ratio, 128)
	}
	if tick&0x4000 != 0 {
		ratio.Mul(ratio, MulConst14).
			Rsh(ratio, 128)
	}
	if tick&0x8000 != 0 {
		ratio.Mul(ratio, MulConst15).
			Rsh(ratio, 128)
	}
	if tick&0x10000 != 0 {
		ratio.Mul(ratio, MulConst16).
			Rsh(ratio, 128)
	}
	if tick&0x20000 != 0 {
		ratio.Mul(ratio, MulConst17).
			Rsh(ratio, 128)
	}
	if tick&0x40000 != 0 {
		ratio.Mul(ratio, MulConst18).
			Rsh(ratio, 128)
	}
	if tick&0x80000 != 0 {
		ratio.Mul(ratio, MulConst19).
			Rsh(ratio, 128)
	}
	if tick&0x100000 != 0 {
		ratio.Mul(ratio, MulConst20).
			Rsh(ratio, 128)
	}

	if activeTick > 0 {
		ratio.Div(MaxUint256, ratio)
	}

	ratio.Mul(ratio, bignumber.BONE).
		Rsh(ratio, 128)

	return ratio, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L56
func pastMaxPrice(delta *Delta) {
	delta.SwappedToMaxPrice = !delta.SqrtPriceLimit.IsZero() &&
		delta.TokenAIn == (delta.SqrtPriceLimit.Cmp(delta.SqrtPrice) <= 0)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L66
func noSwapReset(delta *Delta) {
	delta.Excess.Clear()
	delta.SkipCombine = true
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L63
func sqrtEdgePrice(delta *Delta) *uint256.Int {
	if delta.TokenAIn {
		return delta.SqrtUpperTickPrice
	}
	return delta.SqrtLowerTickPrice
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L210
func adjustAB(bin *Bin, delta *Delta, thisBinAmount, totalAmount *uint256.Int, activeTick int32,
	state *MaverickPoolState) error {
	deltaIn, err := mulDiv(delta.DeltaInBinInternal, thisBinAmount, totalAmount, false)
	if err != nil {
		return err
	}
	var deltaOut *uint256.Int
	if !delta.Excess.IsZero() {
		deltaOut = new(uint256.Int)
	} else {
		if deltaOut, err = mulDiv(delta.DeltaOutErc, thisBinAmount, totalAmount, true); err != nil {
			return err
		} else if deltaOut.IsZero() {
			return ErrInvalidDeltaOut
		}
	}

	if delta.TokenAIn {
		bin.ReserveA = bin.ReserveA.Add(bin.ReserveA, deltaIn)

		if delta.Excess.Sign() <= 0 {
			bin.ReserveB = clip(bin.ReserveB, deltaOut)
		} else {
			bin.ReserveB = new(uint256.Int)
		}
	} else {
		bin.ReserveB = bin.ReserveB.Add(bin.ReserveB, deltaIn)

		if delta.Excess.Sign() <= 0 {
			bin.ReserveA = clip(bin.ReserveA, deltaOut)
		} else {
			bin.ReserveA = new(uint256.Int)
		}
	}

	// Custom code to update state.Bin
	var active = getKindsAtTick(state.BinMap, activeTick)
	if active.Word&(1<<bin.Kind) != 0 {
		if binID, ok := state.BinPositions[activeTick][bin.Kind]; ok {
			state.Bins[binID] = *bin
		}
	}

	return nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L484C3-L484C23
func getTickSqrtPAndLiq(reserveA, reserveB, lowerSqrtP, upperSqrtP *uint256.Int) (sqrtP, liq *uint256.Int, err error) {
	liq, err = getTickL(reserveA, reserveB, lowerSqrtP, upperSqrtP)
	if err != nil {
		return nil, nil, err
	} else if liq.Sign() < 0 {
		return nil, nil, ErrInvalidLiquidity
	} else if reserveA.IsZero() {
		return lowerSqrtP, liq, nil
	} else if reserveB.IsZero() {
		return upperSqrtP, liq, nil
	}

	qo, err := mul(liq, lowerSqrtP)
	if err != nil {
		return nil, nil, err
	}
	bo, err := div(liq, upperSqrtP)
	if err != nil {
		return nil, nil, err
	}

	tmpDiv, err := div(qo.Add(reserveA, qo), bo.Add(reserveB, bo))
	if err != nil {
		return nil, nil, err
	}
	sqrtPrice := sqrt(tmpDiv)

	return sqrtPrice, liq, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-pool-math.ts#L434
func getTickL(
	reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *uint256.Int,
) (*uint256.Int, error) {
	precisionBump := uint(0)

	var tmp uint256.Int
	if tmp.Rsh(reserveA, 60).IsZero() && tmp.Rsh(reserveB, 60).IsZero() {
		precisionBump = 40
		reserveA = new(uint256.Int).Lsh(reserveA, precisionBump)
		reserveB = new(uint256.Int).Lsh(reserveB, precisionBump)
	}

	if reserveA.IsZero() || reserveB.IsZero() {
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
func nextActive(binMap map[int16]*uint256.Int, tick int32, isRight bool, minBinMapIndex, maxBinMapIndex int16) int32 {
	refTick := tick
	if isRight {
		refTick++
	} else {
		refTick--
	}
	mapIndex, shift := getMapPointer(refTick * Kinds)

	var tack int16
	var nextTick int32
	if isRight {
		tack = 1
		nextTick = 1000000000
	} else {
		shift = WordSize - shift
		tack = -1
		nextTick = -1000000000
	}

	if len(binMap) == 0 { // we can never find anything so don't bother going to the expensive loop
		return nextTick
	}

	var nextWord uint256.Int
	var found bool
	for i := 0; i < 4000; i++ {
		if bin := binMap[mapIndex]; bin != nil {
			if isRight {
				nextWord.Rsh(bin, shift)
			} else {
				nextWord.Lsh(bin, shift)
			}
			if !nextWord.IsZero() {
				found = true
				break
			}
		}

		shift = 0
		mapIndex += tack
		// mapIndex will always either increase or decrease, not both
		// so we can check against min/max index and terminate early
		if tack > 0 && mapIndex > maxBinMapIndex || tack < 0 && mapIndex < minBinMapIndex {
			break
		}
	}

	var subIndex int32
	if found {
		if isRight {
			subIndex = int32(lsb(&nextWord)) + int32(shift)
		} else {
			subIndex = int32(msb(&nextWord)) - int32(shift)
		}
		pos := int32(mapIndex)*WordSize + subIndex
		neg := pos < 0
		if neg {
			pos++
		}
		nextTick = pos / Kinds // use truncated div here instead of Euclidean div (-1427/4 = -356 instead of -357)
		if neg {
			nextTick--
		}
	}

	return nextTick
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L126
func getKindsAtTick(binMap map[int16]*uint256.Int, tick int32) Active {
	mapIndex, offset := getMapPointer(tick * Kinds)
	var word uint8
	if subMap := binMap[mapIndex]; subMap != nil {
		var tmp uint256.Int
		word = uint8(tmp.Rsh(subMap, offset).Uint64() & KindMask)
	}
	return Active{
		Word: word,
		Tick: tick,
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L97
func getMapPointer(tick int32) (mapIndex int16, offset uint) {
	return int16(tick >> Offsets), uint(tick & OffsetMask)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L54
func lsb(x *uint256.Int) uint {
	for i := 0; i < len(x); i++ {
		if w := x[i]; w != 0 {
			return uint(i*64 + bits.TrailingZeros64(w))
		}
	}
	return 255
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-bin-map.ts#L17
func msb(x *uint256.Int) uint {
	for i := len(x) - 1; i >= 0; i-- {
		if w := x[i]; w != 0 {
			return uint(i*64 + 63 - bits.LeadingZeros64(w))
		}
	}
	return 0
}

// ------------- maverick basic math --------------------

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L131
func mulDiv(a, b, c *uint256.Int, ceil bool) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)
	if product.IsZero() {
		return product, nil
	}
	var tmp uint256.Int
	if ceil && !tmp.Mod(product, c).IsZero() {
		return product.AddUint64(product.Div(product, c), 1), nil
	}
	return product.Div(product, c), nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L46
func clip(x, y *uint256.Int) *uint256.Int {
	if x.Cmp(y) < 0 {
		return new(uint256.Int)
	}
	return new(uint256.Int).Sub(x, y)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L42
func inv(a *uint256.Int) (*uint256.Int, error) {
	return div(Bone, a)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L68
func div(a, b *uint256.Int) (*uint256.Int, error) {
	return divDownFixed(a, b)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L120
func divDownFixed(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrDividedByZero
	} else if a.IsZero() {
		return new(uint256.Int), nil
	} else {
		aInflated := new(uint256.Int).Mul(a, Bone)
		return aInflated.Div(aInflated, b), nil
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L72
func sqrt(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return new(uint256.Int)
	}

	x = new(uint256.Int).Mul(x, Bone)

	// Set the initial guess to the least power of two that is greater than or equal to sqrt(x).
	var xAux uint256.Int
	xAux.Set(x)
	result := uint256.NewInt(1)
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
func minU(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L34
func mul(a, b *uint256.Int) (*uint256.Int, error) {
	var tmp, tmo uint256.Int
	if tmp.Mod(
		tmp.Mul(tmp.Abs(a), tmo.Abs(b)),
		Bone,
	).Cmp(MulConst49_17) > 0 {
		return mulUpFixed(a, b)
	} else {
		return sMulDownFixed(a, b)
	}
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L50
func mulUpFixed(a, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)
	isNegative := product.Sign() == -1

	if product.IsZero() {
		return product, nil
	}

	result := product.Abs(product).SubUint64(product, 1)
	result.Div(result, Bone)
	result.AddUint64(result, 1)

	if isNegative {
		result.Neg(result)
	}

	return result, nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L181
func sMulDownFixed(a, b *uint256.Int) (*uint256.Int, error) {
	var product = new(uint256.Int).Mul(a, b)
	return product.Div(product, Bone), nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L146
func sDivDownFixed(a, b *uint256.Int) (*uint256.Int, error) {
	return divDownFixed(a, b)
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-math/maverick-basic-math.ts#L158
func sMulUpFixed(a, b *uint256.Int) (*uint256.Int, error) {
	product := new(uint256.Int).Mul(a, b)
	var tmp uint256.Int
	if !a.IsZero() && tmp.Div(product, a).Cmp(b) != 0 {
		return nil, ErrMulOverflow
	} else if product.IsZero() {
		return product, nil
	}
	return product.AddUint64(
		product.Div(product.SubUint64(product, 1), Bone),
		1,
	), nil
}

// https://github.com/paraswap/paraswap-dex-lib/blob/34f92e9e34080ee1389be9ea0f6e82740e748a64/src/dex/maverick-v1/maverick-v1-pool.ts#L369
func scaleFromAmount(amount *uint256.Int, decimals uint8) (*uint256.Int, error) {
	if decimals == 18 {
		return amount, nil
	}
	var scalingFactor uint256.Int
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
func ScaleToAmount(amount *uint256.Int, decimals uint8) (*uint256.Int, error) {
	if decimals == 18 {
		return amount, nil
	}
	var scalingFactor uint256.Int
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
