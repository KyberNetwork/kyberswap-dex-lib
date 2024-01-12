package maverickv1

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func GetAmountOut(
	state *MaverickPoolState,
	amount *big.Int,
	tokenAIn bool,
	exactOutput bool,
	isForPricing bool,
) (*big.Int, *big.Int, error) {
	delta := &Delta{
		DeltaInBinInternal: big.NewInt(0),
		DeltaInErc:         big.NewInt(0),
		DeltaOutErc:        big.NewInt(0),
		Excess:             new(big.Int).Set(amount),
		TokenAIn:           tokenAIn,
		EndSqrtPrice:       big.NewInt(0),
		ExactOutput:        exactOutput,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		DecrementTick:      false,
		SqrtPriceLimit:     big.NewInt(0),
		SqrtLowerTickPrice: big.NewInt(0),
		SqrtUpperTickPrice: big.NewInt(0),
		SqrtPrice:          big.NewInt(0),
	}

	var counter = 0
	for delta.Excess.Cmp(zeroBI) > 0 {
		newDelta, err := swapTick(delta, state)
		if err != nil {
			return nil, nil, err
		}
		combine(delta, newDelta)

		// We can not do too much iteration. This variable chosen
		// as reasonable threshold
		counter += 1
		if isForPricing && counter > MaxSwapIterationCalculation {
			return zeroBI, zeroBI, nil
		}
	}
	var amountIn = delta.DeltaInErc
	var amountOut = delta.DeltaOutErc

	return amountIn, amountOut, nil
}

func swapTick(delta *Delta, state *MaverickPoolState) (*Delta, error) {
	var activeTick = new(big.Int).Set(state.ActiveTick)
	if delta.DecrementTick {
		activeTick = new(big.Int).Sub(state.ActiveTick, bignumber.One)
	}

	var active = getKindsAtTick(state.BinMap, activeTick)

	if active.Word.Cmp(zeroBI) == 0 {
		activeTick = nextActive(state.BinMap, activeTick, delta.TokenAIn)
	}

	var currentReserveA, currentReserveB, currentLiquidity *big.Int
	var currentBins []Bin
	var err error

	oldActiveTick := new(big.Int).Set(activeTick)
	currentReserveA, currentReserveB, delta.SqrtPrice, currentLiquidity, currentBins, err = currentTickLiquidity(activeTick, state)
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
		if currentReserveB.Cmp(zeroBI) > 0 {
			thisBinAmount = currentBins[i].ReserveB
		}

		var totalAmount = currentReserveA
		if currentReserveB.Cmp(zeroBI) > 0 {
			totalAmount = currentReserveB
		}

		if err := adjustAB(&currentBins[i], newDelta, thisBinAmount, totalAmount, oldActiveTick, state); err != nil {
			return nil, err
		}
	}

	if newDelta.Excess.Cmp(zeroBI) != 0 {
		if newDelta.TokenAIn {
			state.ActiveTick = new(big.Int).Add(state.ActiveTick, bignumber.One)
		}
		newDelta.DecrementTick = !delta.TokenAIn
	}

	return newDelta, nil
}

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
		DeltaInBinInternal: big.NewInt(0),
		DeltaInErc:         big.NewInt(0),
		DeltaOutErc:        big.NewInt(0),
		Excess:             big.NewInt(0),
		TokenAIn:           tokenAIn,
		EndSqrtPrice:       big.NewInt(0),
		ExactOutput:        true,
		DecrementTick:      false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		SqrtPriceLimit:     big.NewInt(0),
		SqrtLowerTickPrice: big.NewInt(0),
		SqrtUpperTickPrice: big.NewInt(0),
		SqrtPrice:          big.NewInt(0),
	}

	delta.DeltaOutErc.Set(min(amountOut, amountOutAvailable))

	var binAmountIn *big.Int
	var err error

	tmpB := sqrtPrice
	if !tokenAIn {
		tmpB, err = inv(sqrtPrice)
		if err != nil {
			return nil, err
		}
	}

	tmpC, err := inv(sqrtPrice)
	if err != nil {
		return nil, err
	}
	if !tokenAIn {
		tmpC = sqrtPrice
	}
	deltaOutErcDivLiquidity, err := div(delta.DeltaOutErc, liquidity)
	if err != nil {
		return nil, err
	}
	tmpC = new(big.Int).Sub(tmpC, deltaOutErcDivLiquidity)

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
	delta.DeltaInErc = new(big.Int).Add(binAmountIn, feeBasis)
	delta.DeltaInBinInternal, err = amountToBin(delta.DeltaInErc, feeBasis, state)
	if err != nil {
		return nil, err
	}
	if swapped {
		delta.Excess = clip(amountOut, delta.DeltaOutErc)
	} else {
		delta.Excess = big.NewInt(0)
	}

	return delta, nil
}

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
		DeltaInBinInternal: big.NewInt(0),
		DeltaInErc:         big.NewInt(0),
		DeltaOutErc:        big.NewInt(0),
		Excess:             big.NewInt(0),
		TokenAIn:           tokenAIn,
		EndSqrtPrice:       big.NewInt(0),
		ExactOutput:        false,
		DecrementTick:      false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		SqrtPriceLimit:     big.NewInt(0),
		SqrtLowerTickPrice: big.NewInt(0),
		SqrtUpperTickPrice: big.NewInt(0),
		SqrtPrice:          big.NewInt(0),
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
		delta.DeltaInErc = new(big.Int).Add(binAmountIn, feeBasis)
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
		delta.DeltaInErc = new(big.Int).Set(amountIn)
		feeBasis = new(big.Int).Sub(delta.DeltaInErc, binAmountIn)
	}
	delta.DeltaInBinInternal, err = amountToBin(delta.DeltaInErc, feeBasis, state)
	if err != nil {
		return nil, err
	}
	if delta.Excess.Cmp(zeroBI) != 0 || liquidity.Cmp(zeroBI) == 0 {
		return delta, nil
	}

	tmpReserve := reserveA
	if tokenAIn {
		tmpReserve = reserveB
	}

	tmpB := new(big.Int).Set(sqrtPrice)
	if tokenAIn {
		tmpB, err = inv(sqrtPrice)
		if err != nil {
			return nil, err
		}
	}
	tmpC := new(big.Int).Set(sqrtPrice)
	if !tokenAIn {
		tmpC, err = inv(sqrtPrice)
		if err != nil {
			return nil, err
		}
	}
	tmpDiv, err := div(binAmountIn, liquidity)
	if err != nil {
		return nil, err
	}
	tmpC = new(big.Int).Add(tmpDiv, tmpC)
	tmpEndSqrtPrice := new(big.Int).Set(tmpC)
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

func amountToBin(deltaInErc, feeBases *big.Int, state *MaverickPoolState) (*big.Int, error) {
	protocolFeeRatio := new(big.Int).Set(state.ProtocolFeeRatio)

	if protocolFeeRatio.Cmp(zeroBI) != 0 {
		tmpMul, err := mul(feeBases, new(big.Int).Mul(protocolFeeRatio, bignumber.TenPowInt(15)))
		if err != nil {
			return nil, err
		}
		return clip(deltaInErc, new(big.Int).Add(tmpMul, bignumber.One)), nil
	}

	return deltaInErc, nil
}

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

func combine(delta, newDelta *Delta) {
	if !delta.SkipCombine {
		delta.DeltaInBinInternal = new(big.Int).Add(delta.DeltaInBinInternal, newDelta.DeltaInBinInternal)
		delta.DeltaInErc = new(big.Int).Add(delta.DeltaInErc, newDelta.DeltaInErc)
		delta.DeltaOutErc = new(big.Int).Add(delta.DeltaOutErc, newDelta.DeltaOutErc)
	}
	delta.Excess = newDelta.Excess
	delta.DecrementTick = newDelta.DecrementTick
	delta.EndSqrtPrice = newDelta.EndSqrtPrice
	delta.SwappedToMaxPrice = newDelta.SwappedToMaxPrice
}

func currentTickLiquidity(activeTick *big.Int, state *MaverickPoolState) (*big.Int, *big.Int, *big.Int, *big.Int, []Bin, error) {
	var active = getKindsAtTick(state.BinMap, activeTick)

	var reserveA = big.NewInt(0)
	var reserveB = big.NewInt(0)
	var bins = make([]Bin, 0)

	for i := 0; i < 4; i++ {
		bigI := big.NewInt(int64(i))
		if new(big.Int).And(active.Word, new(big.Int).Lsh(big.NewInt(1), uint(i))).Cmp(zeroBI) > 0 {
			if state.BinPositions[activeTick.String()] == nil {
				state.BinPositions[activeTick.String()] = make(map[string]*big.Int)
			}
			var binID = state.BinPositions[activeTick.String()][bigI.String()]
			if binID == nil {
				binID = big.NewInt(0)
			}
			if binID.Cmp(zeroBI) > 0 {
				var bin = state.Bins[binID.String()]
				reserveA = new(big.Int).Add(reserveA, bin.ReserveA)
				reserveB = new(big.Int).Add(reserveB, bin.ReserveB)
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

func tickPrice(tickSpacing *big.Int, activeTick *big.Int) (*big.Int, error) {
	var tick big.Int
	if activeTick.Cmp(zeroBI) < 0 {
		tick.Neg(activeTick)
	} else {
		tick.Set(activeTick)
	}

	tick.Mul(&tick, tickSpacing)

	if tick.Cmp(MaxTickBI) > 0 {
		return nil, ErrLargerThanMaxTick
	}

	var ratio *big.Int
	if tick.Bit(0) != 0 {
		ratio = new(big.Int).Set(CompareConst1)
	} else {
		ratio = new(big.Int).Set(CompareConst2)
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

	if activeTick.Cmp(zeroBI) > 0 {
		ratio.Div(
			big.NewInt(0).Sub(
				new(big.Int).Lsh(big.NewInt(1), 256),
				bignumber.One),
			ratio,
		)
	}

	ratio.Mul(ratio, bignumber.BONE)
	ratio.Rsh(ratio, 128)

	return ratio, nil
}

func pastMaxPrice(delta *Delta) {
	var tmpBool = delta.SqrtPriceLimit.Cmp(delta.SqrtPrice) >= 0
	if delta.TokenAIn {
		tmpBool = delta.SqrtPriceLimit.Cmp(delta.SqrtPrice) <= 0
	}
	delta.SwappedToMaxPrice = delta.SqrtPriceLimit.Cmp(zeroBI) != 0 && tmpBool
}

func noSwapReset(delta *Delta) {
	delta.Excess = big.NewInt(0)
	delta.SkipCombine = true
	delta.EndSqrtPrice = delta.SqrtPrice
}

func sqrtEdgePrice(delta *Delta) *big.Int {
	if delta.TokenAIn {
		return delta.SqrtUpperTickPrice
	}
	return delta.SqrtLowerTickPrice
}

func adjustAB(bin *Bin, delta *Delta, thisBinAmount, totalAmount, activeTick *big.Int, state *MaverickPoolState) error {
	var deltaOut = big.NewInt(0)
	deltaIn, err := mulDiv(delta.DeltaInBinInternal, thisBinAmount, totalAmount, false)
	if err != nil {
		return err
	}
	if delta.Excess.Cmp(zeroBI) == 0 {
		deltaOut, err = mulDiv(delta.DeltaOutErc, thisBinAmount, totalAmount, true)
		if err != nil {
			return err
		}
	}

	if delta.TokenAIn {
		bin.ReserveA = new(big.Int).Add(bin.ReserveA, deltaIn)

		if delta.Excess.Cmp(zeroBI) <= 0 {
			bin.ReserveB = clip(bin.ReserveB, deltaOut)
		} else {
			bin.ReserveB = big.NewInt(0)
		}
	} else {
		bin.ReserveB = new(big.Int).Add(bin.ReserveB, deltaIn)

		if delta.Excess.Cmp(zeroBI) <= 0 {
			bin.ReserveA = clip(bin.ReserveA, deltaOut)
		} else {
			bin.ReserveA = big.NewInt(0)
		}
	}

	// Custom code to update state.Bin
	var active = getKindsAtTick(state.BinMap, activeTick)
	bigI := big.NewInt(bin.Kind.Int64())
	if new(big.Int).And(active.Word, new(big.Int).Lsh(big.NewInt(1), uint(bin.Kind.Int64()))).Cmp(zeroBI) > 0 {
		if state.BinPositions[activeTick.String()] == nil {
			state.BinPositions[activeTick.String()] = make(map[string]*big.Int)
		}
		var binID = state.BinPositions[activeTick.String()][bigI.String()]
		if binID == nil {
			binID = big.NewInt(0)
		}
		if binID.Cmp(zeroBI) > 0 {
			state.Bins[binID.String()] = *bin
		}
	}

	return nil
}

func getTickSqrtPriceAndL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *big.Int) (*big.Int, *big.Int, error) {
	liquidity, err := getTickL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice)
	if err != nil {
		return nil, nil, err
	}
	if liquidity.Cmp(big.NewInt(0)) < 0 {
		return nil, nil, ErrInvalidLiquidity
	}

	if reserveA.Cmp(big.NewInt(0)) == 0 {
		return sqrtLowerTickPrice, liquidity, nil
	}

	if reserveB.Cmp(big.NewInt(0)) == 0 {
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

	tmpDiv, err := div(new(big.Int).Add(reserveA, qo), new(big.Int).Add(reserveB, bo))
	if err != nil {
		return nil, nil, err
	}
	sqrtPrice := sqrt(tmpDiv)

	return sqrtPrice, liquidity, nil
}

func getTickL(
	reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *big.Int,
) (*big.Int, error) {
	precisionBump := big.NewInt(0)
	big40 := big.NewInt(40)

	if new(big.Int).Rsh(reserveA, 60).Cmp(zeroBI) == 0 && new(big.Int).Rsh(reserveB, 60).Cmp(zeroBI) == 0 {
		precisionBump.Set(big40)
		reserveA = new(big.Int).Lsh(reserveA, uint(precisionBump.Int64()))
		reserveB = new(big.Int).Lsh(reserveB, uint(precisionBump.Int64()))
	}

	if reserveA.Cmp(zeroBI) == 0 || reserveB.Cmp(zeroBI) == 0 {
		tmpDiv, err := div(reserveA, sqrtUpperTickPrice)
		if err != nil {
			return nil, err
		}
		tmpMul, err := mul(reserveB, sqrtLowerTickPrice)
		if err != nil {
			return nil, err
		}
		b := new(big.Int).Add(tmpDiv, tmpMul)

		result, err := mulDiv(b, sqrtUpperTickPrice, new(big.Int).Sub(sqrtUpperTickPrice, sqrtLowerTickPrice), false)
		if err != nil {
			return nil, err
		}
		result = new(big.Int).Rsh(result, uint(precisionBump.Int64()))

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
		b := new(big.Int).Add(tmpDiv, tmpMul)
		b = new(big.Int).Rsh(b, 1)

		diff := new(big.Int).Sub(sqrtUpperTickPrice, sqrtLowerTickPrice)

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
		result, err := mulDiv(new(big.Int).Add(b, sqrt(new(big.Int).Add(tmpMulBB, tmpMulDiv))), sqrtUpperTickPrice, diff, false)
		if err != nil {
			return nil, err
		}
		result = new(big.Int).Rsh(result, uint(precisionBump.Int64()))

		return result, nil
	}
}

// ------------- maverick bin map -----------------------

func nextActive(binMap map[string]*big.Int, tick *big.Int, isRight bool) *big.Int {
	var refTick, shift, tack, subIndex, nextTick *big.Int

	refTick = new(big.Int).Set(tick)
	if isRight {
		refTick = new(big.Int).Add(tick, bignumber.One)
	}
	var offset, mapIndex = getMapPointer(new(big.Int).Mul(refTick, Kinds))
	if isRight {
		shift = offset
		tack = big.NewInt(1)
		nextTick = big.NewInt(1000000000)
	} else {
		shift = new(big.Int).Sub(WordSize, offset)
		tack = big.NewInt(-1)
		nextTick = big.NewInt(-1000000000)
	}

	binMapCount := len(binMap)
	if binMapCount == 0 {
		// we can never find anything so don't bother going to the expensive loop
		return nextTick
	}

	// we'll use a single bigInt for nextWord through the loop, instead of allocating every times
	var nextWord big.Int
	for i := 0; i < 4000; i++ {
		bin := binMap[mapIndex.String()]
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
		if nextWord.Cmp(zeroBI) != 0 {
			break
		}
		if i == 0 {
			shift = big.NewInt(0)
		}
		// mapIndex already get allocated within `getMapPointer`, so we can safely overwrite it here
		mapIndex = mapIndex.Add(mapIndex, tack)
	}

	if nextWord.Cmp(zeroBI) != 0 {
		if isRight {
			subIndex = new(big.Int).Add(lsb(&nextWord), shift)
		} else {
			subIndex = new(big.Int).Sub(msb(&nextWord), shift)
		}
		posFirst := new(big.Int).Add(new(big.Int).Mul(mapIndex, WordSize), subIndex)
		pos := new(big.Int).Set(posFirst)
		if posFirst.Cmp(zeroBI) < 0 {
			pos = new(big.Int).Add(pos, bignumber.One)
		}
		nextTick = new(big.Int).Div(pos, Kinds)
		if posFirst.Cmp(zeroBI) < 0 {
			nextTick = new(big.Int).Sub(nextTick, bignumber.One)
		}
	}

	return nextTick
}

func getKindsAtTick(binMap map[string]*big.Int, tick *big.Int) Active {
	offset, mapIndex := getMapPointer(new(big.Int).Mul(tick, Kinds))
	subMap := binMap[mapIndex.String()]
	if subMap == nil {
		subMap = big.NewInt(0)
	}
	presentBits := new(big.Int).And(new(big.Int).Rsh(subMap, uint(offset.Int64())), Mask)

	return Active{
		Word: presentBits,
		Tick: new(big.Int).Set(tick),
	}
}

func getMapPointer(tick *big.Int) (*big.Int, *big.Int) {
	offset := new(big.Int).And(tick, OffsetMask)
	mapIndex := new(big.Int).Rsh(tick, 8)

	return offset, mapIndex
}

func lsb(x *big.Int) *big.Int {
	r := big.NewInt(255)
	// bigint in typescript is pass by value. So I do not want this function change the input X
	tmpX := new(big.Int).Set(x)

	if tmpX.And(tmpX, bignumber.NewBig("0xffffffffffffffffffffffffffffffff")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(128))
	} else {
		tmpX.Rsh(tmpX, 128)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0xffffffffffffffff")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(64))
	} else {
		tmpX.Rsh(tmpX, 64)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0xffffffff")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(32))
	} else {
		tmpX.Rsh(tmpX, 32)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0xffff")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(16))
	} else {
		tmpX.Rsh(tmpX, 16)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0xff")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(8))
	} else {
		tmpX.Rsh(tmpX, 8)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0xf")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(4))
	} else {
		tmpX.Rsh(tmpX, 4)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0x3")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(2))
	} else {
		tmpX.Rsh(tmpX, 2)
	}
	if tmpX.And(tmpX, bignumber.NewBig("0x1")).Cmp(zeroBI) > 0 {
		r.Sub(r, big.NewInt(1))
	}

	return r
}

func msb(x *big.Int) *big.Int {
	r := big.NewInt(0)

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
func mulDiv(a, b, c *big.Int, ceil bool) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)
	if product.Cmp(zeroBI) == 0 {
		return big.NewInt(0), nil
	} else {
		if ceil && new(big.Int).Mod(product, c).Cmp(zeroBI) != 0 {
			return new(big.Int).Add(new(big.Int).Div(product, c), bignumber.One), nil
		} else {
			return new(big.Int).Div(product, c), nil
		}
	}
}

func clip(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Sub(x, y)
}

func inv(a *big.Int) (*big.Int, error) {
	return div(new(big.Int).Set(One), a)
}

func div(a, b *big.Int) (*big.Int, error) {
	return divDownFixed(a, b)
}

func divDownFixed(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(zeroBI) != 0 {
		if a.Cmp(zeroBI) == 0 {
			return big.NewInt(0), nil
		} else {
			aInflated := new(big.Int).Mul(a, One)
			return new(big.Int).Div(aInflated, b), nil
		}
	}

	return nil, ErrDividedByZero
}

func sqrt(x *big.Int) *big.Int {
	if x.Cmp(zeroBI) == 0 {
		return big.NewInt(0)
	}

	x = new(big.Int).Mul(x, Unit)

	// Set the initial guess to the least power of two that is greater than or equal to sqrt(x).
	xAux := new(big.Int).Set(x)
	result := big.NewInt(1)
	if xAux.Cmp(XAuxConst64) >= 0 {
		xAux.Rsh(xAux, 128)
		result.Lsh(result, 64)
	}
	if xAux.Cmp(XAuxConst32) >= 0 {
		xAux.Rsh(xAux, 64)
		result.Lsh(result, 32)
	}
	if xAux.Cmp(XAuxConst16) >= 0 {
		xAux.Rsh(xAux, 32)
		result.Lsh(result, 16)
	}
	if xAux.Cmp(XAuxConst8) >= 0 {
		xAux.Rsh(xAux, 16)
		result.Lsh(result, 8)
	}
	if xAux.Cmp(XAuxConst4) >= 0 {
		xAux.Rsh(xAux, 8)
		result.Lsh(result, 4)
	}
	if xAux.Cmp(XAuxConst2) >= 0 {
		xAux.Rsh(xAux, 4)
		result.Lsh(result, 2)
	}
	if xAux.Cmp(XAuxConst1) >= 0 {
		result.Lsh(result, 1)
	}

	var xDiv big.Int
	for i := 0; i < 7; i++ { // Seven iterations should be enough
		xDiv.Div(x, result)
		result.Add(result, &xDiv)
		result.Rsh(result, 1)
	}

	roundedDownResult := new(big.Int).Div(x, result)
	if result.Cmp(roundedDownResult) >= 0 {
		return roundedDownResult
	}
	return result
}

func min(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}

func mul(a, b *big.Int) (*big.Int, error) {
	if new(big.Int).Mod(
		new(big.Int).Mul(abs(a), abs(b)),
		Unit,
	).Cmp(MulConst49_17) > 0 {
		return mulUpFixed(a, b)
	} else {
		return sMulDownFixed(a, b)
	}
}

func mulUpFixed(a, b *big.Int) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)
	isNegative := product.Sign() == -1

	if product.Cmp(zeroBI) == 0 {
		return big.NewInt(0), nil
	} else {
		result := new(big.Int).Sub(abs(product), bignumber.One)
		result.Div(result, One)
		result.Add(result, bignumber.One)

		if isNegative {
			result.Neg(result)
		}

		return result, nil
	}
}

func sMulDownFixed(a, b *big.Int) (*big.Int, error) {
	var product = new(big.Int).Mul(a, b)
	return new(big.Int).Div(product, One), nil
}

func abs(x *big.Int) *big.Int {
	if x.Sign() < 0 {
		return new(big.Int).Neg(x)
	}
	return x
}

func sDivDownFixed(a, b *big.Int) (*big.Int, error) {
	return divDownFixed(a, b)
}

func sMulUpFixed(a, b *big.Int) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)
	if a.Cmp(zeroBI) == 0 || new(big.Int).Div(product, a).Cmp(b) == 0 {
		if product.Cmp(zeroBI) == 0 {
			return big.NewInt(0), nil
		} else {
			return new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Sub(product, bignumber.One), One),
				bignumber.One,
			), nil
		}
	}

	return nil, ErrMulOverflow
}

func scaleFromAmount(amount *big.Int, decimals uint8) (*big.Int, error) {
	if decimals == 18 {
		return amount, nil
	}
	var scalingFactor *big.Int
	if decimals > 18 {
		scalingFactor = new(big.Int).Mul(
			bignumber.BONE,
			bignumber.TenPowInt(decimals-18),
		)
		return sDivDownFixed(amount, scalingFactor)
	} else {
		scalingFactor = new(big.Int).Mul(
			bignumber.BONE,
			bignumber.TenPowInt(18-decimals),
		)
		return sMulUpFixed(amount, scalingFactor)
	}
}

func ScaleToAmount(amount *big.Int, decimals uint8) (*big.Int, error) {
	if decimals == 18 {
		return amount, nil
	}
	var scalingFactor *big.Int
	if decimals > 18 {
		scalingFactor = new(big.Int).Mul(
			bignumber.BONE,
			bignumber.TenPowInt(decimals-18),
		)
		return sMulUpFixed(amount, scalingFactor)
	} else {
		scalingFactor = new(big.Int).Mul(
			bignumber.BONE,
			bignumber.TenPowInt(18-decimals),
		)
		return sDivDownFixed(amount, scalingFactor)
	}
}
