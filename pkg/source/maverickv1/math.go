package maverickv1

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
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
		new(big.Int).Sub(bignumber.TenPowInt(18), state.Fee),
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

	tmp, err := mul(amountIn, new(big.Int).Sub(bignumber.TenPowInt(18), state.Fee))
	if err != nil {
		return nil, err
	}
	if tmp.Cmp(binAmountIn) >= 0 {
		feeBasis, err = mulDiv(binAmountIn, state.Fee, new(big.Int).Sub(bignumber.TenPowInt(18), state.Fee), true)
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
		binAmountIn, err = mul(amountIn, new(big.Int).Sub(bignumber.TenPowInt(18), state.Fee))
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
	var tick *big.Int
	if activeTick.Cmp(zeroBI) < 0 {
		tick = new(big.Int).Neg(activeTick)
	} else {
		tick = new(big.Int).Set(activeTick)
	}

	tick = new(big.Int).Mul(tick, tickSpacing)

	if tick.Cmp(big.NewInt(int64(MaxTick))) > 0 {
		return nil, ErrLargerThanMaxTick
	}

	var ratio *big.Int
	if new(big.Int).And(tick, bignumber.NewBig("0x1")).Cmp(zeroBI) != 0 {
		ratio = bignumber.NewBig("0xfffcb933bd6fad9d3af5f0b9f25db4d6")
	} else {
		ratio = bignumber.NewBig("0x100000000000000000000000000000000")
	}

	if new(big.Int).And(tick, bignumber.NewBig("0x2")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xfff97272373d41fd789c8cb37ffcaa1c"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x4")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xfff2e50f5f656ac9229c67059486f389"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x8")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xffe5caca7e10e81259b3cddc7a064941"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x10")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xffcb9843d60f67b19e8887e0bd251eb7"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x20")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xff973b41fa98cd2e57b660be99eb2c4a"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x40")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xff2ea16466c9838804e327cb417cafcb"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x80")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xfe5dee046a99d51e2cc356c2f617dbe0"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x100")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xfcbe86c7900aecf64236ab31f1f9dcb5"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x200")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xf987a7253ac4d9194200696907cf2e37"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x400")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xf3392b0822b88206f8abe8a3b44dd9be"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x800")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xe7159475a2c578ef4f1d17b2b235d480"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x1000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xd097f3bdfd254ee83bdd3f248e7e785e"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x2000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0xa9f746462d8f7dd10e744d913d033333"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x4000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x70d869a156ddd32a39e257bc3f50aa9b"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x8000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x31be135f97da6e09a19dc367e3b6da40"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x10000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x9aa508b5b7e5a9780b0cc4e25d61a56"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x20000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x5d6af8dedbcb3a6ccb7ce618d14225"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x40000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x2216e584f630389b2052b8db590e"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x80000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x48a1703920644d4030024fe"))
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(tick, bignumber.NewBig("0x100000")).Cmp(zeroBI) != 0 {
		ratio.Mul(ratio, bignumber.NewBig("0x149b34ee7b4532"))
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

	ratio.Mul(ratio, bignumber.TenPowInt(18))
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
		bin.ReserveB = big.NewInt(0)
		if delta.Excess.Cmp(zeroBI) <= 0 {
			bin.ReserveB = clip(bin.ReserveB, deltaOut)
		}
	} else {
		bin.ReserveB = new(big.Int).Add(bin.ReserveB, deltaIn)
		bin.ReserveA = big.NewInt(0)
		if delta.Excess.Cmp(zeroBI) <= 0 {
			bin.ReserveA = clip(bin.ReserveA, deltaOut)
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
	var refTick, shift, tack, nextWord, subIndex, nextTick *big.Int

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

	for i := 0; i < 4000; i++ {
		nextWord = binMap[mapIndex.String()]
		if nextWord == nil {
			nextWord = big.NewInt(0)
		}
		if isRight {
			nextWord = new(big.Int).Rsh(nextWord, uint(shift.Uint64()))
		} else {
			nextWord = new(big.Int).Lsh(nextWord, uint(shift.Uint64()))
		}
		nextWord = new(big.Int).And(nextWord, BitMask)
		if nextWord.Cmp(zeroBI) != 0 {
			break
		}
		shift = big.NewInt(0)
		mapIndex = new(big.Int).Add(mapIndex, tack)
	}

	if nextWord != nil && nextWord.Cmp(zeroBI) != 0 {
		if isRight {
			subIndex = new(big.Int).Add(lsb(nextWord), shift)
		} else {
			subIndex = new(big.Int).Sub(msb(nextWord), shift)
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
	if a.Cmp(zeroBI) == 0 || new(big.Int).Div(product, a).Cmp(b) == 0 {
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

	return nil, ErrMulOverflow
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
	if xAux.Cmp(bignumber.NewBig("0x100000000000000000000000000000000")) >= 0 {
		xAux.Rsh(xAux, 128)
		result.Lsh(result, 64)
	}
	if xAux.Cmp(bignumber.NewBig("0x10000000000000000")) >= 0 {
		xAux.Rsh(xAux, 64)
		result.Lsh(result, 32)
	}
	if xAux.Cmp(bignumber.NewBig("0x100000000")) >= 0 {
		xAux.Rsh(xAux, 32)
		result.Lsh(result, 16)
	}
	if xAux.Cmp(bignumber.NewBig("0x10000")) >= 0 {
		xAux.Rsh(xAux, 16)
		result.Lsh(result, 8)
	}
	if xAux.Cmp(bignumber.NewBig("0x100")) >= 0 {
		xAux.Rsh(xAux, 8)
		result.Lsh(result, 4)
	}
	if xAux.Cmp(bignumber.NewBig("0x10")) >= 0 {
		xAux.Rsh(xAux, 4)
		result.Lsh(result, 2)
	}
	if xAux.Cmp(bignumber.NewBig("0x8")) >= 0 {
		result.Lsh(result, 1)
	}

	for i := 0; i < 7; i++ { // Seven iterations should be enough
		xDiv := new(big.Int).Div(x, result)
		result = new(big.Int).Add(result, xDiv)
		result = new(big.Int).Rsh(result, 1)
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
	).Cmp(bignumber.NewBig("499999999999999999")) > 0 {
		return mulUpFixed(a, b)
	} else {
		return sMulDownFixed(a, b)
	}
}

func mulUpFixed(a, b *big.Int) (*big.Int, error) {
	product := new(big.Int).Mul(a, b)

	if a.Cmp(zeroBI) == 0 || new(big.Int).Div(product, a).Cmp(b) == 0 {
		isNegative := false
		if (a.Cmp(zeroBI) < 0 && b.Cmp(zeroBI) > 0) || (a.Cmp(zeroBI) > 0 && b.Cmp(zeroBI) < 0) {
			isNegative = true
		}

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

	return nil, ErrMulOverflow
}

func sMulDownFixed(a, b *big.Int) (*big.Int, error) {
	var product = new(big.Int).Mul(a, b)
	if a.Cmp(zeroBI) == 0 || new(big.Int).Div(product, a).Cmp(b) == 0 {
		return new(big.Int).Div(product, One), nil
	}

	return nil, ErrMulOverflow
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
			bignumber.TenPowInt(18),
			bignumber.TenPowInt(decimals-18),
		)
		return sDivDownFixed(amount, scalingFactor)
	} else {
		scalingFactor = new(big.Int).Mul(
			bignumber.TenPowInt(18),
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
			bignumber.TenPowInt(18),
			bignumber.TenPowInt(decimals-18),
		)
		return sMulUpFixed(amount, scalingFactor)
	} else {
		scalingFactor = new(big.Int).Mul(
			bignumber.TenPowInt(18),
			bignumber.TenPowInt(18-decimals),
		)
		return sDivDownFixed(amount, scalingFactor)
	}
}
