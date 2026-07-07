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
	amountOut, fee, newPoolState, err := s.calculateSwapInWithState(idxIn, idxOut, amountIn, s.DexVars)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{NewDexVars: newPoolState},
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
	amountIn, fee, newPoolState, err := s.calculateSwapOutWithState(idxIn, idxOut, amountOut, s.DexVars)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: fee.ToBig()},
		Gas:           defaultGas,
		SwapInfo:      SwapInfo{NewDexVars: newPoolState},
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
		s.DexVars = swapInfo.NewDexVars
		token0TotalSupplyAdjusted := s.DexVars.Token0TotalSupplyAdjusted
		token1TotalSupplyAdjusted := s.DexVars.Token1TotalSupplyAdjusted
		s.Info.Reserves[0] = s.adjustFromInternalDecimals(token0TotalSupplyAdjusted, 0).ToBig()
		s.Info.Reserves[1] = s.adjustFromInternalDecimals(token1TotalSupplyAdjusted, 1).ToBig()
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		DexKey:          s.DexKey,
		ApprovalAddress: s.DexLiteAddress,
	}
}

// ------------------------------------------------------------------------------------------------
// FluidDexLite Math Implementation
// Implementing exactly the same logic as in the FluidDexLite contract
// ------------------------------------------------------------------------------------------------

// calculateSwapInWithState implements the exact same logic as _swapIn in the FluidDexLite contract
// Returns: amountOut, fee, newPoolState (all 4 variables), error
func (s *PoolSimulator) calculateSwapInWithState(idxIn, idxOut int, amountIn *uint256.Int,
	dexVars *UnpackedDexVariables) (*uint256.Int, *uint256.Int, *UnpackedDexVariables, error) {
	if dexVars == nil {
		return nil, nil, nil, ErrPoolNotInitialized
	}

	// Calculate pricing and imaginary reserves with complete shifting logic
	newDexVars := lo.ToPtr(*dexVars)
	centerPrice, imaginaryReserves, err := s.getPricesAndReservesWithState(newDexVars)
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

	// Update dex variables with new supplies
	newDexVars.Token0TotalSupplyAdjusted = newSupplies[0]
	newDexVars.Token1TotalSupplyAdjusted = newSupplies[1]

	return amountOut, fee, newDexVars, nil
}

// calculateSwapOutWithState implements the exact same logic as _swapOut in the FluidDexLite contract
// Returns: amountIn, fee, newPoolState (all 4 variables), error
func (s *PoolSimulator) calculateSwapOutWithState(idxIn, idxOut int, amountOut *uint256.Int,
	dexVars *UnpackedDexVariables) (*uint256.Int, *uint256.Int, *UnpackedDexVariables, error) {
	if dexVars == nil {
		return nil, nil, nil, ErrPoolNotInitialized
	}

	// Calculate pricing and imaginary reserves with complete shifting logic
	newDexVars := lo.ToPtr(*dexVars)
	centerPrice, imaginaryReserves, err := s.getPricesAndReservesWithState(newDexVars)
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
	amountIn = s.adjustFromInternalDecimals(amountIn, idxIn)

	// Update dex variables with new supplies
	newDexVars.Token0TotalSupplyAdjusted = newSupplies[0]
	newDexVars.Token1TotalSupplyAdjusted = newSupplies[1]

	return amountIn, fee, newDexVars, nil
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

// calcCenterPrice implements _calcCenterPrice from the contract
func (s *PoolSimulator) calcCenterPrice(poolState *PoolState,
	dexVars *UnpackedDexVariables, blockTimestamp uint64) *uint256.Int {
	oldCenterPrice := expandCenterPrice(dexVars.CenterPrice)
	centerPriceShift := poolState.CenterPriceShift
	tmp := rshAnd(centerPriceShift, BitPosCenterPriceShiftTimestamp, X33)
	tmp2 := rshAnd(centerPriceShift, BitPosCenterPriceShiftLastInteractionTimestamp, X33)
	fromTimestamp := big256.Max(tmp, tmp2).Uint64()
	newCenterPrice := tmp2.Set(poolState.NewCenterPrice)

	priceShift := rshAnd(centerPriceShift, BitPosCenterPriceShiftPercent, X20)
	timePassed := tmp.SetUint64(blockTimestamp - fromTimestamp)
	shiftDuration := rshAnd(centerPriceShift, BitPosCenterPriceShiftTimeToShift, X20)
	priceShift.MulDivOverflow(priceShift.Mul(oldCenterPrice, priceShift), timePassed,
		shiftDuration.Mul(shiftDuration, SixDecimals))

	if newCenterPrice.Cmp(oldCenterPrice) > 0 {
		oldCenterPrice.Add(oldCenterPrice, priceShift)
		if newCenterPrice.Cmp(oldCenterPrice) > 0 {
			newCenterPrice.Set(oldCenterPrice)
		} else { // shifting fully done
			dexVars.CenterPriceShiftActive = false
		}
	} else {
		if oldCenterPrice.Cmp(priceShift) > 0 {
			oldCenterPrice.Sub(oldCenterPrice, priceShift)
		} else {
			oldCenterPrice.Clear()
		}
		if newCenterPrice.Cmp(oldCenterPrice) < 0 {
			newCenterPrice.Set(oldCenterPrice)
		} else { // shifting fully done
			dexVars.CenterPriceShiftActive = false
		}
	}

	return newCenterPrice
}

// calcRangeShifting implements _calcRangeShifting from the contract
func (s *PoolSimulator) calcRangeShifting(poolState *PoolState, dexVars *UnpackedDexVariables,
	blockTimestamp uint64) (*uint256.Int, *uint256.Int) {
	upperRange, lowerRange := dexVars.UpperPercent, dexVars.LowerPercent
	rangeShift := poolState.RangeShift

	// Extract shift data
	tmp := rshAnd(rangeShift, BitPosRangeShiftTimestamp, X33)
	startTimestamp := tmp.Uint64()
	shiftDuration := rshAnd(rangeShift, BitPosRangeShiftTimeToShift, X20)
	if blockTimestamp >= startTimestamp+shiftDuration.Uint64() { // shifting fully done
		dexVars.RangePercentShiftActive = false
		return upperRange, lowerRange
	}

	// Extract old values
	oldLowerRange := rshAnd(rangeShift, BitPosRangeShiftOldLowerRangePercent, X14)
	oldUpperRange := rshAnd(rangeShift, BitPosRangeShiftOldUpperRangePercent, X14)

	// Calculate shifted values
	timePassed := tmp.SetUint64(blockTimestamp - startTimestamp)
	newUpperRange := s.calcShiftingDone(upperRange, oldUpperRange, timePassed, shiftDuration)
	newLowerRange := s.calcShiftingDone(lowerRange, oldLowerRange, timePassed, shiftDuration)

	return newUpperRange, newLowerRange
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
func expandCenterPrice(centerPrice *uint256.Int) *uint256.Int {
	var coefficient, exponent uint256.Int
	return coefficient.Lsh(coefficient.Rsh(centerPrice, DefaultExponentSize),
		uint(exponent.And(centerPrice, DefaultExponentMask).Uint64()))
}

// getPricesAndReservesWithState implements complete _getPricesAndReserves with all shifting logic
func (s *PoolSimulator) getPricesAndReservesWithState(dexVars *UnpackedDexVariables) (*uint256.Int, [2]*uint256.Int,
	error) {
	// Use the actual block timestamp from when the pool state was fetched
	blockTimestamp := s.BlockTimestamp

	var centerPrice *uint256.Int
	if dexVars.CenterPriceShiftActive {
		centerPrice = s.calcCenterPrice(&s.PoolState, dexVars, blockTimestamp)
	} else { // Extract center price with exponential encoding (static price only)
		centerPrice = expandCenterPrice(dexVars.CenterPrice)
	}

	// Extract range percents
	upperRangePercent := dexVars.UpperPercent
	lowerRangePercent := dexVars.LowerPercent

	// Check if range shift is active
	if dexVars.RangePercentShiftActive {
		// An active range shift is going on
		upperRangePercent, lowerRangePercent = s.calcRangeShifting(&s.PoolState, dexVars, blockTimestamp)
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
	rebalancingStatus := dexVars.RebalancingStatus
	if rebalancingStatus > 1 {
		centerPriceShift := s.PoolState.CenterPriceShift
		if centerPriceShift.Sign() > 0 {
			shiftingTime := rshAnd(centerPriceShift, BitPosCenterPriceShiftShiftingTime, X24)
			var timeElapsed uint256.Int
			lastInteractionTimestamp := tmp.And(centerPriceShift, X33)
			timeElapsed.Sub(timeElapsed.SetUint64(blockTimestamp), lastInteractionTimestamp)

			switch rebalancingStatus {
			case 2:
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
			case 3:
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
			maxCenterPriceExpanded := expandCenterPrice(maxCenterPrice)
			if centerPrice.Cmp(maxCenterPriceExpanded) > 0 {
				centerPrice = maxCenterPriceExpanded
			} else {
				minCenterPrice := rshAnd(centerPriceShift, BitPosCenterPriceShiftMinCenterPrice, X28)
				minCenterPriceExpanded := expandCenterPrice(minCenterPrice)
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
