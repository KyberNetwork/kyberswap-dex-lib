package dexLite

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra
	TokenDecimals [2]uint8

	DexKey         DexKey                // Pool's key (token0, token1, salt)
	PoolState      PoolState             // The 4 storage variables
	DexVars        *UnpackedDexVariables // Unpacked dex variables
	BlockTimestamp uint64                // Block timestamp when state was fetched
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	// Use the correctly calculated fee and reserves from entity.Pool for efficiency
	// These are extracted from PoolState.dexVariables in PoolListUpdater/PoolTracker
	fee := big.NewInt(int64(entityPool.SwapFee * FeePercentPrecision))

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
			SwapFee:     fee,
		}},
		StaticExtra:    staticExtra,
		TokenDecimals:  [2]uint8{entityPool.Tokens[0].Decimals, entityPool.Tokens[1].Decimals},
		DexKey:         extra.DexKey,
		PoolState:      extra.PoolState,
		DexVars:        unpackDexVariables(extra.PoolState.DexVariables),
		BlockTimestamp: extra.BlockTimestamp,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	idxIn, idxOut := s.GetTokenIndex(param.TokenAmountIn.Token), s.GetTokenIndex(param.TokenOut)
	if idxIn == -1 || idxOut == -1 {
		return nil, ErrInvalidToken
	}

	// Simulate the swap and get the complete new state and fee
	amountOut, fee, newPoolState, err := s.calculateSwapInWithState(idxIn, idxOut, amountIn, s.PoolState)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{NewPoolState: newPoolState},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if param.TokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}
	amountOut, overflow := uint256.FromBig(param.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	idxIn, idxOut := s.GetTokenIndex(param.TokenIn), s.GetTokenIndex(param.TokenAmountOut.Token)
	if idxIn == -1 || idxOut == -1 {
		return nil, ErrInvalidToken
	}

	// Simulate the swap and get the complete new state and fee
	amountIn, fee, newPoolState, err := s.calculateSwapOutWithState(idxIn, idxOut, amountOut, s.PoolState)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: fee.ToBig()},
		Gas:           defaultGas,
		SwapInfo:      SwapInfo{NewPoolState: newPoolState},
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// Update the PoolState (source of truth for FluidDexLite calculations)
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.PoolState = *swapInfo.NewPoolState

		// Also update entity.Pool reserves for efficiency and consistency
		// Extract new supplies from updated dexVariables
		token0TotalSupplyAdjusted, token1TotalSupplyAdjusted := unpackTotalSupplies(swapInfo.NewPoolState.DexVariables)

		s.Info.Reserves[0] = s.adjustFromInternalDecimals(token0TotalSupplyAdjusted, 0).ToBig()
		s.Info.Reserves[1] = s.adjustFromInternalDecimals(token1TotalSupplyAdjusted, 1).ToBig()
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
		DexKey:          s.DexKey,
		ApprovalAddress: s.StaticExtra.DexLiteAddress,
	}
}

// ------------------------------------------------------------------------------------------------
// FluidDexLite Math Implementation
// Implementing exactly the same logic as in the FluidDexLite contract
// ------------------------------------------------------------------------------------------------

// calculateSwapInWithState implements the exact same logic as _swapIn in the FluidDexLite contract
// Returns: amountOut, fee, newPoolState (all 4 variables), error
func (s *PoolSimulator) calculateSwapInWithState(idxIn, idxOut int, amountIn *uint256.Int,
	currentPoolState PoolState) (*uint256.Int, *uint256.Int, *PoolState, error) {
	if currentPoolState.DexVariables.Sign() == 0 {
		return nil, nil, nil, ErrPoolNotInitialized
	}

	// Calculate pricing and imaginary reserves with complete shifting logic
	newPoolState := currentPoolState.Clone() // all 4 variables can potentially change
	centerPrice, imaginaryReserves, err := s.getPricesAndReservesWithState(s.DexVars, newPoolState)
	if err != nil {
		return nil, nil, nil, err
	}

	var tmp, tmp2, tmp3 uint256.Int
	// Adjust input amount to internal decimals (9 precision as in contract)
	amountInAdjusted := s.adjustToInternalDecimals(amountIn, idxIn)
	if amountInAdjusted.Cmp(FourDecimals) < 0 || amountInAdjusted.Cmp(X60) > 0 {
		return nil, nil, nil, ErrInvalidAmountIn
	} else if amountInAdjusted.Cmp(tmp.Div(imaginaryReserves[idxIn], big256.U2)) > 0 {
		return nil, nil, nil, ErrExcessiveSwapAmount
	}

	feeInAdjusted, _ := tmp.MulDivOverflow(amountInAdjusted, s.DexVars.Fee, SixDecimals)
	fee := s.adjustFromInternalDecimals(feeInAdjusted, idxIn)

	// amountOut = (amountIn * iReserveOut) / (iReserveIn + amountIn)
	amountInAfterFee := tmp2.Sub(amountInAdjusted, feeInAdjusted)
	denominator := tmp3.Add(imaginaryReserves[idxIn], amountInAfterFee)
	amountOut, _ := tmp2.MulDivOverflow(amountInAfterFee, imaginaryReserves[idxOut], denominator)

	supplies := [2]*uint256.Int{s.DexVars.Token0TotalSupplyAdjusted, s.DexVars.Token1TotalSupplyAdjusted}
	if supplies[idxOut].Cmp(amountOut) < 0 {
		return nil, nil, nil, ErrInsufficientReserve
	}
	var newSupplies [2]*uint256.Int
	revenueCut, _ := tmp.MulDivOverflow(feeInAdjusted, s.DexVars.RevenueCut, TwoDecimals)
	newSupplies[idxIn] = amountInAdjusted.Add(supplies[idxIn], amountInAdjusted.Sub(amountInAdjusted, revenueCut))
	newSupplies[idxOut] = new(uint256.Int).Sub(supplies[idxOut], amountOut)

	// Check ratio: token1Supply >= (token0Supply * centerPrice) / (PRICE_PRECISION * MINIMUM_LIQUIDITY_SWAP)
	// Check ratio: token0Supply >= (token1Supply * PRICE_PRECISION) / (centerPrice * MINIMUM_LIQUIDITY_SWAP)
	minPriceIn, minLiqPrice := centerPrice, PricePrecision
	if idxIn == 1 {
		minPriceIn, minLiqPrice = PricePrecision, centerPrice
	}
	minTokenOut, _ := tmp.MulDivOverflow(newSupplies[idxIn], minPriceIn, tmp3.Mul(minLiqPrice, MinimumLiquiditySwap))
	if newSupplies[idxOut].Cmp(minTokenOut) < 0 {
		return nil, nil, nil, ErrTokenReservesRatioTooHigh
	} else if newSupplies[idxIn].Cmp(X60) > 0 || newSupplies[idxOut].Cmp(X60) > 0 { // Check for overflow
		return nil, nil, nil, ErrAdjustedSupplyOverflow
	}

	// Convert output back to token decimals
	amountOut = s.adjustFromInternalDecimals(amountOut, idxOut)

	// Calculate the current price after swap for rebalancing status check
	// price = (token1ImaginaryReserves - amountOut) * PRICE_PRECISION / (token0ImaginaryReserves + amountIn)
	// price = (token1ImaginaryReserves + amountIn) * PRICE_PRECISION / (token0ImaginaryReserves - amountOut)
	imaginaryReserves[idxIn].Add(imaginaryReserves[idxIn], s.adjustToInternalDecimals(amountIn, idxIn))
	imaginaryReserves[idxOut].Sub(imaginaryReserves[idxOut], s.adjustToInternalDecimals(amountOut, idxOut))
	currentPrice, _ := imaginaryReserves[1].MulDivOverflow(imaginaryReserves[1], PricePrecision, imaginaryReserves[0])

	// Update rebalancing status and check for state changes
	rebalancingStatus := rshAnd(newPoolState.DexVariables, BitPosRebalancingStatus, X2)
	if rebalancingStatus.Sign() > 0 {
		blockTimestamp := s.BlockTimestamp
		newRebalancingStatus := s.getRebalancingStatus(newPoolState.DexVariables, newPoolState, rebalancingStatus,
			currentPrice, centerPrice, blockTimestamp)

		// Update centerPriceShift timestamp if rebalancing is active or center price shift is active
		centerPriceShiftActive := rshAnd(newPoolState.DexVariables, BitPosCenterPriceShiftActive, X1)
		if newRebalancingStatus.Cmp(big256.U1) > 0 || centerPriceShiftActive.Cmp(big256.U1) == 0 {
			// Update last interaction timestamp: _centerPriceShift[dexId_] = _centerPriceShift[dexId_] & ~(X33 << BITS_DEX_LITE_CENTER_PRICE_SHIFT_LAST_INTERACTION_TIMESTAMP) | (block.timestamp << BITS_DEX_LITE_CENTER_PRICE_SHIFT_LAST_INTERACTION_TIMESTAMP)
			clearMask := tmp.Lsh(X33, BitPosCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift.And(newPoolState.CenterPriceShift, clearMask.Not(clearMask))
			newTimestamp := tmp.Lsh(tmp.SetUint64(blockTimestamp), BitPosCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift.Or(newPoolState.CenterPriceShift, newTimestamp)
		}
	}

	// Update dex variables with new supplies
	newPoolState.DexVariables = s.updateSuppliesInDexVariables(newPoolState.DexVariables, newSupplies)

	return amountOut, fee, newPoolState, nil
}

// calculateSwapOutWithState implements the exact same logic as _swapOut in the FluidDexLite contract
// Returns: amountIn, fee, newPoolState (all 4 variables), error
func (s *PoolSimulator) calculateSwapOutWithState(idxIn, idxOut int, amountOut *uint256.Int,
	currentPoolState PoolState) (*uint256.Int, *uint256.Int, *PoolState, error) {
	if currentPoolState.DexVariables.Sign() == 0 {
		return nil, nil, nil, ErrPoolNotInitialized
	}

	newPoolState := currentPoolState.Clone() // all 4 variables can potentially change
	// Calculate pricing and imaginary reserves with complete shifting logic
	centerPrice, imaginaryReserves, err := s.getPricesAndReservesWithState(s.DexVars, newPoolState)
	if err != nil {
		return nil, nil, nil, err
	}

	var tmp, tmp2, tmp3 uint256.Int
	// Adjust output amount to internal decimals (9 precision as in contract)
	amountOutAdjusted := s.adjustToInternalDecimals(amountOut, idxOut)
	if amountOutAdjusted.Cmp(FourDecimals) < 0 || amountOutAdjusted.Cmp(X60) > 0 {
		return nil, nil, nil, ErrInvalidAmountOut
	} else if amountOutAdjusted.Cmp(tmp.Div(imaginaryReserves[idxOut], big256.U2)) > 0 {
		return nil, nil, nil, ErrExcessiveSwapAmount
	}

	// amountIn = (amountOut * iReserveIn) / (iReserveOut - amountOut)
	denominator := tmp3.Sub(imaginaryReserves[idxOut], amountOutAdjusted)
	if denominator.Sign() <= 0 {
		return nil, nil, nil, ErrInsufficientReserve
	}
	amountIn, _ := tmp2.MulDivOverflow(amountOutAdjusted, imaginaryReserves[idxIn], denominator)

	// Calculate fee and total amount in
	feeDenominator := tmp.Sub(SixDecimals, s.DexVars.Fee)
	if feeDenominator.Sign() <= 0 {
		return nil, nil, nil, ErrInvalidFeeRate
	}

	totalAmountIn, _ := tmp3.MulDivOverflow(amountIn, SixDecimals, feeDenominator)
	feeInAdjusted := tmp.Sub(totalAmountIn, amountIn)
	fee := s.adjustFromInternalDecimals(feeInAdjusted, idxIn)
	amountIn = totalAmountIn

	supplies := [2]*uint256.Int{s.DexVars.Token0TotalSupplyAdjusted, s.DexVars.Token1TotalSupplyAdjusted}
	if supplies[idxOut].Cmp(amountOutAdjusted) < 0 {
		return nil, nil, nil, ErrInsufficientReserve
	}
	var newSupplies [2]*uint256.Int
	revenueCut, _ := tmp.MulDivOverflow(feeInAdjusted, s.DexVars.RevenueCut, TwoDecimals)
	newSupplies[idxIn] = tmp2.Add(supplies[idxIn], revenueCut.Sub(amountIn, revenueCut))
	newSupplies[idxOut] = amountOutAdjusted.Sub(supplies[idxOut], amountOutAdjusted)

	// Check ratio: token1Supply >= (token0Supply * centerPrice) / (PRICE_PRECISION * MINIMUM_LIQUIDITY_SWAP)
	// Check ratio: token0Supply >= (token1Supply * PRICE_PRECISION) / (centerPrice * MINIMUM_LIQUIDITY_SWAP)
	minPriceIn, minLiqPrice := centerPrice, PricePrecision
	if idxIn == 1 {
		minPriceIn, minLiqPrice = PricePrecision, centerPrice
	}
	minTokenOut, _ := tmp.MulDivOverflow(newSupplies[idxIn], minPriceIn, tmp3.Mul(minLiqPrice, MinimumLiquiditySwap))
	if newSupplies[idxOut].Cmp(minTokenOut) < 0 {
		return nil, nil, nil, ErrTokenReservesRatioTooHigh
	} else if newSupplies[idxIn].Cmp(X60) > 0 || newSupplies[idxOut].Cmp(X60) > 0 { // Check for overflow
		return nil, nil, nil, ErrAdjustedSupplyOverflow
	}

	// Convert input back to token decimals
	amountIn = s.adjustFromInternalDecimals(amountIn, 0)

	// Calculate the current price after swap for rebalancing status check (same logic as in calculateSwapInWithState)
	// price = (token1ImaginaryReserves - amountOut) * PRICE_PRECISION / (token0ImaginaryReserves + amountIn)
	// price = (token1ImaginaryReserves + amountIn) * PRICE_PRECISION / (token0ImaginaryReserves - amountOut)
	imaginaryReserves[idxIn].Add(imaginaryReserves[idxIn], s.adjustToInternalDecimals(amountIn, idxIn))
	imaginaryReserves[idxOut].Sub(imaginaryReserves[idxOut], s.adjustToInternalDecimals(amountOut, idxOut))
	currentPrice, _ := imaginaryReserves[1].MulDivOverflow(imaginaryReserves[1], PricePrecision, imaginaryReserves[0])

	// Update rebalancing status and check for state changes
	rebalancingStatus := rshAnd(newPoolState.DexVariables, BitPosRebalancingStatus, X2)
	if rebalancingStatus.Sign() > 0 {
		blockTimestamp := s.BlockTimestamp
		newRebalancingStatus := s.getRebalancingStatus(newPoolState.DexVariables, newPoolState, rebalancingStatus,
			currentPrice, centerPrice, blockTimestamp)

		// Update centerPriceShift timestamp if rebalancing is active or center price shift is active
		centerPriceShiftActive := rshAnd(newPoolState.DexVariables, BitPosCenterPriceShiftActive, X1)
		if newRebalancingStatus.Cmp(big256.U1) > 0 || centerPriceShiftActive.Cmp(big256.U1) == 0 {
			// Update last interaction timestamp
			clearMask := tmp.Lsh(X33, BitPosCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift.And(newPoolState.CenterPriceShift, clearMask.Not(clearMask))
			newTimestamp := tmp.Lsh(tmp.SetUint64(blockTimestamp), BitPosCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift.Or(newPoolState.CenterPriceShift, newTimestamp)
		}
	}

	// Update dex variables with new supplies
	newPoolState.DexVariables = s.updateSuppliesInDexVariables(newPoolState.DexVariables, newSupplies)

	return amountIn, fee, newPoolState, nil
}

// ------------------------------------------------------------------------------------------------
// FluidDexLite Shifting Helper Functions
// Implementing exactly the same logic as in the FluidDexLite contract helpers
// ------------------------------------------------------------------------------------------------

// calcShiftingDone implements _calcShiftingDone from the contract
func (s *PoolSimulator) calcShiftingDone(current, old, timePassed, shiftDuration *uint256.Int) *uint256.Int {
	var diff uint256.Int
	diff.Sub(current, old)
	currentGtOld := diff.Sign() > 0
	// current > old: old + ((current - old) * timePassed) / shiftDuration
	shifted, _ := diff.MulDivOverflow(diff.Abs(&diff), timePassed, shiftDuration)
	if currentGtOld {
		return shifted.Add(old, shifted)
	} else {
		return shifted.Sub(old, shifted)
	}
}

// calcRangeShifting implements _calcRangeShifting from the contract
func (s *PoolSimulator) calcRangeShifting(upperRange, lowerRange *uint256.Int, poolState *PoolState,
	blockTimestamp uint64) (*uint256.Int, *uint256.Int) {
	rangeShift := poolState.RangeShift

	// Extract shift data
	shiftDuration := rshAnd(rangeShift, BitPosRangeShiftTimeToShift, X20)
	startTimestamp := rshAnd(rangeShift, BitPosRangeShiftTimestamp, X33)

	currentTime := uint256.NewInt(blockTimestamp)
	var tmp uint256.Int
	endTime := tmp.Add(startTimestamp, shiftDuration)

	if currentTime.Cmp(endTime) >= 0 {
		// Shifting fully done - clear the range shift and deactivate
		poolState.RangeShift = big256.U0 // delete _rangeShift[dexId_]

		// Clear range shift active bit in dexVariables
		mask := tmp.Lsh(big256.U1, BitPosRangePercentShiftActive)
		poolState.DexVariables.And(poolState.DexVariables, mask.Not(mask))

		return upperRange, lowerRange
	}

	timePassed := currentTime.Sub(currentTime, startTimestamp)

	// Extract old values
	oldLowerRange := rshAnd(rangeShift, BitPosRangeShiftOldLowerRangePercent, X14)
	oldUpperRange := rangeShift.And(rangeShift, X14) // first 14 bits

	// Calculate shifted values
	newUpperRange := s.calcShiftingDone(upperRange, oldUpperRange, timePassed, shiftDuration)
	newLowerRange := s.calcShiftingDone(lowerRange, oldLowerRange, timePassed, shiftDuration)

	return newUpperRange, newLowerRange
}

// calcThresholdShifting implements _calcThresholdShifting from the contract
func (s *PoolSimulator) calcThresholdShifting(upperThreshold, lowerThreshold *uint256.Int, poolState *PoolState,
	blockTimestamp uint64) (*uint256.Int, *uint256.Int) {
	thresholdShift := poolState.ThresholdShift

	// Extract shift data
	shiftDuration := rshAnd(thresholdShift, BitPosThresholdShiftTimeToShift, X20)
	startTimestamp := rshAnd(thresholdShift, BitPosThresholdShiftTimestamp, X33)

	currentTime := uint256.NewInt(blockTimestamp)
	var tmp uint256.Int
	endTime := tmp.Add(startTimestamp, shiftDuration)

	if currentTime.Cmp(endTime) >= 0 {
		// Shifting fully done - clear the threshold shift and deactivate
		poolState.ThresholdShift = big256.U0 // delete _thresholdShift[dexId_]

		// Clear threshold shift active bit in dexVariables
		mask := tmp.Lsh(big256.U1, BitPosThresholdPercentShiftActive)
		poolState.DexVariables.And(poolState.DexVariables, mask.Not(mask))

		return upperThreshold, lowerThreshold
	}

	timePassed := currentTime.Sub(currentTime, startTimestamp)

	// Extract old values - 7 bits each
	oldLowerThreshold := rshAnd(thresholdShift, BitPosThresholdShiftOldLowerThresholdPercent, X7)
	oldUpperThreshold := thresholdShift.And(thresholdShift, X7) // first 7 bits

	// Calculate shifted values
	newUpperThreshold := s.calcShiftingDone(upperThreshold, oldUpperThreshold, timePassed, shiftDuration)
	newLowerThreshold := s.calcShiftingDone(lowerThreshold, oldLowerThreshold, timePassed, shiftDuration)

	return newUpperThreshold, newLowerThreshold
}

// getRebalancingStatus implements _getRebalancingStatus from the contract
func (s *PoolSimulator) getRebalancingStatus(dexVariables *uint256.Int, poolState *PoolState,
	rebalancingStatus, price, centerPrice *uint256.Int, blockTimestamp uint64) *uint256.Int {
	// Extract range percents from dexVariables
	upperRangePercent := rshAnd(dexVariables, BitPosUpperPercent, X14)
	lowerRangePercent := rshAnd(dexVariables, BitPosLowerPercent, X14)

	// Check if range shift is active and calculate if needed
	rangeShiftActive := rshAnd(dexVariables, BitPosRangePercentShiftActive, X1)
	if rangeShiftActive.Cmp(big256.U1) == 0 {
		upperRangePercent, lowerRangePercent = s.calcRangeShifting(upperRangePercent, lowerRangePercent, poolState,
			blockTimestamp)
	}

	// Calculate range prices
	var upperRangePrice, lowerRangePrice, tmp uint256.Int
	// upperRangePrice = (centerPrice * FOUR_DECIMALS) / (FOUR_DECIMALS - upperRangePercent)
	denominator := tmp.Sub(FourDecimals, upperRangePercent)
	upperRangePrice.MulDivOverflow(centerPrice, FourDecimals, denominator)
	// lowerRangePrice = (centerPrice * (FOUR_DECIMALS - lowerRangePercent)) / FOUR_DECIMALS
	numerator := tmp.Sub(FourDecimals, lowerRangePercent)
	lowerRangePrice.MulDivOverflow(centerPrice, numerator, FourDecimals)

	// Extract threshold percents
	upperThresholdPercent := rshAnd(dexVariables, BitPosUpperShiftThresholdPercent, X7)
	lowerThresholdPercent := rshAnd(dexVariables, BitPosLowerShiftThresholdPercent, X7)

	// Check if threshold shift is active and calculate if needed
	thresholdShiftActive := rshAnd(dexVariables, BitPosThresholdPercentShiftActive, X1)
	if thresholdShiftActive.Cmp(big256.U1) == 0 {
		upperThresholdPercent, lowerThresholdPercent = s.calcThresholdShifting(upperThresholdPercent,
			lowerThresholdPercent, poolState, blockTimestamp)
	}

	// Calculate threshold prices
	// upperThreshold = centerPrice + ((upperRangePrice - centerPrice) * (TWO_DECIMALS - upperThresholdPercent)) / TWO_DECIMALS
	rangeDiff := upperRangePrice.Sub(&upperRangePrice, centerPrice)
	thresholdFactor := tmp.Sub(TwoDecimals, upperThresholdPercent)
	adjustment, _ := rangeDiff.MulDivOverflow(rangeDiff, thresholdFactor, TwoDecimals)
	upperThreshold := adjustment.Add(centerPrice, adjustment)
	// lowerThreshold = centerPrice - ((centerPrice - lowerRangePrice) * (TWO_DECIMALS - lowerThresholdPercent)) / TWO_DECIMALS
	rangeDiff = lowerRangePrice.Sub(centerPrice, &lowerRangePrice)
	thresholdFactor = tmp.Sub(TwoDecimals, lowerThresholdPercent)
	adjustment, _ = rangeDiff.MulDivOverflow(rangeDiff, thresholdFactor, TwoDecimals)
	lowerThreshold := adjustment.Sub(centerPrice, adjustment)

	// Check thresholds and update rebalancing status
	if price.Cmp(upperThreshold) > 0 {
		if rebalancingStatus.Cmp(big256.U2) != 0 {
			// Update dexVariables with rebalancing status = 2
			clearMask := tmp.Lsh(X2, BitPosRebalancingStatus)
			poolState.DexVariables.And(poolState.DexVariables, clearMask.Not(clearMask))
			newStatus := tmp.Lsh(big256.U2, BitPosRebalancingStatus)
			poolState.DexVariables.Or(poolState.DexVariables, newStatus)
			return big256.U2
		}
	} else if price.Cmp(lowerThreshold) < 0 {
		if rebalancingStatus.Cmp(big256.U3) != 0 {
			// Update dexVariables with rebalancing status = 3
			clearMask := tmp.Lsh(X2, BitPosRebalancingStatus)
			poolState.DexVariables.And(poolState.DexVariables, clearMask.Not(clearMask))
			newStatus := tmp.Lsh(big256.U3, BitPosRebalancingStatus)
			poolState.DexVariables.Or(poolState.DexVariables, newStatus)
			return big256.U3
		}
	} else {
		// Price is within normal range
		if rebalancingStatus.Cmp(big256.U1) != 0 {
			// Update dexVariables with rebalancing status = 1
			clearMask := tmp.Lsh(X2, BitPosRebalancingStatus)
			poolState.DexVariables.And(poolState.DexVariables, clearMask.Not(clearMask))
			newStatus := tmp.Lsh(big256.U1, BitPosRebalancingStatus)
			poolState.DexVariables.Or(poolState.DexVariables, newStatus)
			return big256.U1
		}
	}

	return rebalancingStatus
}

// Helper function to adjust amounts to internal decimals (TOKENS_DECIMALS_PRECISION = 9)
func (s *PoolSimulator) adjustToInternalDecimals(amount *uint256.Int, tokenIdx int) *uint256.Int {
	decimals := s.TokenDecimals[tokenIdx]
	if decimals > TokensDecimalsPrecision {
		return new(uint256.Int).Div(amount, big256.TenPow(decimals-TokensDecimalsPrecision))
	} else {
		return new(uint256.Int).Mul(amount, big256.TenPow(TokensDecimalsPrecision-decimals))
	}
}

// Helper function to adjust amounts from internal decimals back to token decimals
func (s *PoolSimulator) adjustFromInternalDecimals(amount *uint256.Int, tokenIdx int) *uint256.Int {
	return adjustFromInternalDecimals(amount, s.TokenDecimals[tokenIdx])
}

// expandCenterPrice expands the compressed center price
func (s *PoolSimulator) expandCenterPrice(centerPrice *uint256.Int) *uint256.Int {
	var coefficient, exponent uint256.Int
	return coefficient.Lsh(coefficient.Rsh(centerPrice, DefaultExponentSize),
		uint(exponent.And(centerPrice, DefaultExponentMask).Uint64()))
}

// getPricesAndReservesWithState implements complete _getPricesAndReserves with all shifting logic
func (s *PoolSimulator) getPricesAndReservesWithState(dexVars *UnpackedDexVariables,
	poolState *PoolState) (*uint256.Int, [2]*uint256.Int, error) {
	// Use the actual block timestamp from when the pool state was fetched
	blockTimestamp := s.BlockTimestamp

	// Check for external center price functionality that we don't support
	centerPriceShiftActive := rshAnd(poolState.DexVariables, BitPosCenterPriceShiftActive, X1)
	centerPriceContractAddress := rshAnd(poolState.DexVariables, BitPosCenterPriceContractAddress, X19)

	if centerPriceShiftActive.Cmp(big256.U1) == 0 || centerPriceContractAddress.Sign() > 0 {
		return nil, [2]*uint256.Int{}, ErrExternalCenterPriceNotSupported
	}

	// Extract center price with exponential encoding (static price only)
	centerPriceRaw := rshAnd(poolState.DexVariables, BitPosCenterPrice, X40)
	centerPrice := s.expandCenterPrice(centerPriceRaw)

	// Extract range percents
	upperRangePercent := rshAnd(poolState.DexVariables, BitPosUpperPercent, X14)
	lowerRangePercent := rshAnd(poolState.DexVariables, BitPosLowerPercent, X14)

	// Check if range shift is active
	rangeShiftActive := rshAnd(poolState.DexVariables, BitPosRangePercentShiftActive, X1)
	if rangeShiftActive.Cmp(big256.U1) == 0 {
		// An active range shift is going on
		upperRangePercent, lowerRangePercent = s.calcRangeShifting(upperRangePercent, lowerRangePercent, poolState,
			blockTimestamp)
	}

	// Calculate range prices
	var upperRangePrice, lowerRangePrice, tmp, tmp2 uint256.Int
	// upperRangePrice = (centerPrice * FOUR_DECIMALS) / (FOUR_DECIMALS - upperRangePercent)
	denominator := tmp.Sub(FourDecimals, upperRangePercent)
	upperRangePrice.MulDivOverflow(centerPrice, FourDecimals, denominator)
	// lowerRangePrice = (centerPrice * (FOUR_DECIMALS - lowerRangePercent)) / FOUR_DECIMALS
	numerator := tmp.Sub(FourDecimals, lowerRangePercent)
	lowerRangePrice.MulDivOverflow(centerPrice, numerator, FourDecimals)

	// Handle rebalancing if status > 1
	rebalancingStatus := rshAnd(poolState.DexVariables, BitPosRebalancingStatus, X2)
	if rebalancingStatus.Cmp(big256.U1) > 0 {
		centerPriceShift := poolState.CenterPriceShift
		if centerPriceShift.Sign() > 0 {
			shiftingTime := rshAnd(centerPriceShift, BitPosCenterPriceShiftShiftingTime, X24)
			var timeElapsed uint256.Int
			lastInteractionTimestamp := tmp.And(centerPriceShift,
				X33) // BitPosCenterPriceShiftLastInteractionTimestamp = 0
			timeElapsed.Sub(timeElapsed.SetUint64(blockTimestamp), lastInteractionTimestamp)

			if rebalancingStatus.Cmp(big256.U2) == 0 {
				// Price shifting towards upper range
				if timeElapsed.Cmp(shiftingTime) < 0 {
					// Partial shift: centerPrice + ((upperRangePrice - centerPrice) * timeElapsed) / shiftingTime
					diff := tmp.Sub(&upperRangePrice, centerPrice)
					shift, _ := tmp.MulDivOverflow(diff, &timeElapsed, shiftingTime)
					centerPrice.Add(centerPrice, shift)
				} else {
					// 100% price shifted
					centerPrice.Set(&upperRangePrice)
				}
			} else if rebalancingStatus.Cmp(big256.U3) == 0 {
				// Price shifting towards lower range
				if timeElapsed.Cmp(shiftingTime) < 0 {
					// Partial shift: centerPrice - ((centerPrice - lowerRangePrice) * timeElapsed) / shiftingTime
					diff := tmp.Sub(centerPrice, &lowerRangePrice)
					shift, _ := tmp.MulDivOverflow(diff, &timeElapsed, shiftingTime)
					centerPrice.Sub(centerPrice, shift)
				} else {
					// 100% price shifted
					centerPrice.Set(&lowerRangePrice)
				}
			}

			// Check min/max bounds if rebalancing actually happened
			maxCenterPrice := rshAnd(centerPriceShift, BitPosCenterPriceShiftMaxCenterPrice, X28)
			maxCenterPriceExpanded := s.expandCenterPrice(maxCenterPrice)
			if centerPrice.Cmp(maxCenterPriceExpanded) > 0 {
				centerPrice = maxCenterPriceExpanded
			} else {
				minCenterPrice := rshAnd(centerPriceShift, BitPosCenterPriceShiftMinCenterPrice, X28)
				minCenterPriceExpanded := s.expandCenterPrice(minCenterPrice)
				if centerPrice.Cmp(minCenterPriceExpanded) < 0 {
					centerPrice = minCenterPriceExpanded
				}
			}

			// Update range prices as center price moved
			denominator = tmp.Sub(FourDecimals, upperRangePercent)
			upperRangePrice.MulDivOverflow(centerPrice, FourDecimals, denominator)
			numerator = tmp.Sub(FourDecimals, lowerRangePercent)
			lowerRangePrice.MulDivOverflow(centerPrice, numerator, FourDecimals)
		}
	}

	// Calculate geometric mean price
	var geometricMeanPrice *uint256.Int
	if upperRangePrice.Cmp(threshold1e38) < 0 {
		// upperRangePrice * lowerRangePrice < 1e76 (within safe limits)
		product := tmp.Mul(&upperRangePrice, &lowerRangePrice)
		geometricMeanPrice = product.Sqrt(product)
	} else {
		// Scale down to prevent overflow
		scaledUpper := tmp.Div(&upperRangePrice, big256.BONE)
		scaledLower := tmp2.Div(&lowerRangePrice, big256.BONE)
		product := tmp.Mul(scaledUpper, scaledLower)
		geometricMeanPrice = tmp.Mul(product.Sqrt(product), big256.BONE)
	}

	// Get token supplies
	token0Supply, token1Supply := dexVars.Token0TotalSupplyAdjusted, dexVars.Token1TotalSupplyAdjusted

	// Calculate imaginary reserves
	var token0ImaginaryReserves, token1ImaginaryReserves *uint256.Int
	if geometricMeanPrice.Cmp(PricePrecision) < 0 { // < 1e27
		token0ImaginaryReserves, token1ImaginaryReserves = s.calculateReservesOutsideRange(
			geometricMeanPrice, &upperRangePrice, token0Supply, token1Supply)
	} else {
		// Inverse calculation for large prices
		inverseGeometricMean := tmp.Div(PricePrecisionSq, geometricMeanPrice) // 1e54 / geometricMeanPrice
		inverseLowerRange := tmp2.Div(PricePrecisionSq, &lowerRangePrice)     // 1e54 / lowerRangePrice

		token1ImaginaryReserves, token0ImaginaryReserves = s.calculateReservesOutsideRange(
			inverseGeometricMean, inverseLowerRange, token1Supply, token0Supply)
	}

	// Add real supplies to imaginary reserves
	token0ImaginaryReserves.Add(token0ImaginaryReserves, token0Supply)
	token1ImaginaryReserves.Add(token1ImaginaryReserves, token1Supply)

	return centerPrice, [2]*uint256.Int{token0ImaginaryReserves, token1ImaginaryReserves}, nil
}

// calculateReservesOutsideRange calculates reserves outside the range (simplified)
func (s *PoolSimulator) calculateReservesOutsideRange(gp, pa, rx, ry *uint256.Int) (*uint256.Int, *uint256.Int) {
	// Simplified calculation based on the contract formula
	var p1, p2, tmp, discriminant uint256.Int
	if p1.Sub(pa, gp).Sign() <= 0 {
		return big256.U0, big256.U0
	}

	p2.Mul(gp, rx)
	p2.Add(&p2, tmp.Mul(ry, PricePrecision))
	p2.Div(&p2, tmp.Mul(big256.U2, &p1))

	discriminant.MulDivOverflow(discriminant.Mul(rx, ry), PricePrecision, &p1)
	discriminant.Add(&discriminant, tmp.Mul(&p2, &p2))

	xa := p1.Add(&p2, discriminant.Sqrt(&discriminant))
	yb, _ := p2.MulDivOverflow(xa, gp, PricePrecision)

	return xa, yb
}

// updateSuppliesInDexVariables updates the token supplies in the packed dex variables
func (s *PoolSimulator) updateSuppliesInDexVariables(dexVariables *uint256.Int, supplies [2]*uint256.Int) *uint256.Int {
	// Clear existing supply bits
	var clearMask0, clearMask1 uint256.Int
	clearMask0.Lsh(X60, BitPosToken0TotalSupplyAdjusted)
	clearMask1.Lsh(X60, BitPosToken1TotalSupplyAdjusted)
	clearMask := clearMask0.Or(&clearMask0, &clearMask1)
	newDexVars := dexVariables.And(dexVariables, clearMask.Not(clearMask))

	newSupply0 := supplies[0].Lsh(supplies[0].And(supplies[0], X60), BitPosToken0TotalSupplyAdjusted)
	newSupply1 := supplies[1].Lsh(supplies[1].And(supplies[1], X60), BitPosToken1TotalSupplyAdjusted)

	return newDexVars.Or(newDexVars, newSupply0).Or(newDexVars, newSupply1)
}
