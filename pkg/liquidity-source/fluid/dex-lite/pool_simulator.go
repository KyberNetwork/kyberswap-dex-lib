package dexLite

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra

	DexKey         DexKey    // Pool's key (token0, token1, salt)
	DexId          [8]byte   // Unique identifier for this pool
	PoolState      PoolState // The 4 storage variables
	BlockTimestamp uint64    // Block timestamp when state was fetched

	Token0Decimals uint8
	Token1Decimals uint8

	SyncTimestamp int64
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
		DexKey:         extra.DexKey,
		DexId:          extra.DexId,
		PoolState:      extra.PoolState,
		BlockTimestamp: extra.BlockTimestamp,
		Token0Decimals: entityPool.Tokens[0].Decimals,
		Token1Decimals: entityPool.Tokens[1].Decimals,
		StaticExtra:    staticExtra,
		SyncTimestamp:  entityPool.Timestamp,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	swap0To1 := param.TokenAmountIn.Token == s.Info.Tokens[0]

	// Simulate the swap and get the complete new state and fee
	amountOut, newPoolState, fee, err := s.calculateSwapInWithState(swap0To1, param.TokenAmountIn.Amount, s.PoolState)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee},
		Gas:            defaultGas.Swap,
		SwapInfo:       SwapInfo{NewPoolState: newPoolState},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if param.TokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	swap0To1 := param.TokenAmountOut.Token == s.Info.Tokens[1]

	// Simulate the swap and get the complete new state and fee
	amountIn, newPoolState, fee, err := s.calculateSwapOutWithState(swap0To1, param.TokenAmountOut.Amount, s.PoolState)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: fee},
		Gas:           defaultGas.Swap,
		SwapInfo:      SwapInfo{NewPoolState: newPoolState},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// Update the PoolState (source of truth for FluidDexLite calculations)
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.PoolState = swapInfo.NewPoolState

		// Also update entity.Pool reserves for efficiency and consistency
		// Extract new supplies from updated dexVariables
		unpackedVars := s.unpackDexVariables(swapInfo.NewPoolState.DexVariables)

		token0Supply := s.adjustFromInternalDecimals(unpackedVars.Token0TotalSupplyAdjusted, true, unpackedVars)
		token1Supply := s.adjustFromInternalDecimals(unpackedVars.Token1TotalSupplyAdjusted, false, unpackedVars)

		s.Info.Reserves[0] = token0Supply
		s.Info.Reserves[1] = token1Supply
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", s.GetAddress())
}

// ------------------------------------------------------------------------------------------------
// FluidDexLite Math Implementation
// Implementing exactly the same logic as in the FluidDexLite contract
// ------------------------------------------------------------------------------------------------

// calculateSwapInWithState implements the exact same logic as _swapIn in the FluidDexLite contract
// Returns: amountOut, newPoolState (all 4 variables), fee, error
func (s *PoolSimulator) calculateSwapInWithState(swap0To1 bool, amountIn *big.Int, currentPoolState PoolState) (*big.Int, PoolState, *big.Int, error) {
	if currentPoolState.DexVariables.Sign() == 0 {
		return nil, PoolState{}, nil, ErrPoolNotInitialized
	}

	// Clone the current state - all 4 variables can potentially change
	newPoolState := currentPoolState.Clone()

	// Unpack dex variables to get current state
	dexVars := s.unpackDexVariables(currentPoolState.DexVariables)

	// Get current supplies
	token0AdjustedSupply := dexVars.Token0TotalSupplyAdjusted
	token1AdjustedSupply := dexVars.Token1TotalSupplyAdjusted

	// Calculate pricing and imaginary reserves with complete shifting logic
	centerPrice, token0ImaginaryReserves, token1ImaginaryReserves, err := s.getPricesAndReservesWithState(dexVars, &newPoolState)
	if err != nil {
		return nil, PoolState{}, nil, err
	}

	var amountOut *big.Int
	var newToken0Supply, newToken1Supply *big.Int
	var feeInAdjusted *big.Int // Fee in adjusted (9-decimal) precision

	if swap0To1 {
		// Adjust input amount to internal decimals (9 precision as in contract)
		amountInAdjusted := s.adjustToInternalDecimals(amountIn, true, dexVars)

		// Validate amount
		if amountInAdjusted.Cmp(FourDecimals) < 0 || amountInAdjusted.Cmp(X60) > 0 {
			return nil, PoolState{}, nil, ErrInvalidAmountIn
		}
		if amountInAdjusted.Cmp(new(big.Int).Div(token0ImaginaryReserves, big.NewInt(2))) > 0 {
			return nil, PoolState{}, nil, ErrExcessiveSwapAmount
		}

		// Calculate fee: fee = (amountIn * fee) / SIX_DECIMALS
		feeInAdjusted = new(big.Int).Mul(amountInAdjusted, dexVars.Fee)
		feeInAdjusted = feeInAdjusted.Div(feeInAdjusted, SixDecimals)

		// Calculate amount out: amountOut = (amountIn * iReserveOut) / (iReserveIn + amountIn)
		amountInAfterFee := new(big.Int).Sub(amountInAdjusted, feeInAdjusted)
		numerator := new(big.Int).Mul(amountInAfterFee, token1ImaginaryReserves)
		denominator := new(big.Int).Add(token0ImaginaryReserves, amountInAfterFee)
		amountOut = new(big.Int).Div(numerator, denominator)

		// Calculate revenue cut
		revenueCut := new(big.Int).Mul(feeInAdjusted, dexVars.RevenueCut)
		revenueCut = revenueCut.Div(revenueCut, TwoDecimals)

		// Update supplies
		newToken0Supply = new(big.Int).Add(token0AdjustedSupply, new(big.Int).Sub(amountInAdjusted, revenueCut))
		newToken1Supply = new(big.Int).Sub(token1AdjustedSupply, amountOut)

		// Validate reserves
		if newToken1Supply.Sign() < 0 {
			return nil, PoolState{}, nil, ErrInsufficientReserve
		}

		// Check ratio: token1Supply >= (token0Supply * centerPrice) / (PRICE_PRECISION * MINIMUM_LIQUIDITY_SWAP)
		minToken1 := new(big.Int).Mul(newToken0Supply, centerPrice)
		minToken1 = minToken1.Div(minToken1, new(big.Int).Mul(PricePrecision, big.NewInt(MinimumLiquiditySwap)))
		if newToken1Supply.Cmp(minToken1) < 0 {
			return nil, PoolState{}, nil, ErrTokenReservesRatioTooHigh
		}

		// Convert output back to token decimals
		amountOut = s.adjustFromInternalDecimals(amountOut, false, dexVars)

	} else {
		// Adjust input amount to internal decimals
		amountInAdjusted := s.adjustToInternalDecimals(amountIn, false, dexVars)

		// Validate amount
		if amountInAdjusted.Cmp(FourDecimals) < 0 || amountInAdjusted.Cmp(X60) > 0 {
			return nil, PoolState{}, nil, ErrInvalidAmountIn
		}
		if amountInAdjusted.Cmp(new(big.Int).Div(token1ImaginaryReserves, big.NewInt(2))) > 0 {
			return nil, PoolState{}, nil, ErrExcessiveSwapAmount
		}

		// Calculate fee
		feeInAdjusted = new(big.Int).Mul(amountInAdjusted, dexVars.Fee)
		feeInAdjusted = feeInAdjusted.Div(feeInAdjusted, SixDecimals)

		// Calculate amount out
		amountInAfterFee := new(big.Int).Sub(amountInAdjusted, feeInAdjusted)
		numerator := new(big.Int).Mul(amountInAfterFee, token0ImaginaryReserves)
		denominator := new(big.Int).Add(token1ImaginaryReserves, amountInAfterFee)
		amountOut = new(big.Int).Div(numerator, denominator)

		// Calculate revenue cut
		revenueCut := new(big.Int).Mul(feeInAdjusted, dexVars.RevenueCut)
		revenueCut = revenueCut.Div(revenueCut, TwoDecimals)

		// Update supplies
		newToken1Supply = new(big.Int).Add(token1AdjustedSupply, new(big.Int).Sub(amountInAdjusted, revenueCut))
		newToken0Supply = new(big.Int).Sub(token0AdjustedSupply, amountOut)

		// Validate reserves
		if newToken0Supply.Sign() < 0 {
			return nil, PoolState{}, nil, ErrInsufficientReserve
		}

		// Check ratio: token0Supply >= (token1Supply * PRICE_PRECISION) / (centerPrice * MINIMUM_LIQUIDITY_SWAP)
		minToken0 := new(big.Int).Mul(newToken1Supply, PricePrecision)
		minToken0 = minToken0.Div(minToken0, new(big.Int).Mul(centerPrice, big.NewInt(MinimumLiquiditySwap)))
		if newToken0Supply.Cmp(minToken0) < 0 {
			return nil, PoolState{}, nil, ErrTokenReservesRatioTooHigh
		}

		// Convert output back to token decimals
		amountOut = s.adjustFromInternalDecimals(amountOut, true, dexVars)
	}

	// Check for overflow
	if newToken0Supply.Cmp(X60) > 0 || newToken1Supply.Cmp(X60) > 0 {
		return nil, PoolState{}, nil, ErrAdjustedSupplyOverflow
	}

	// Calculate the current price after swap for rebalancing status check
	var currentPrice *big.Int
	if swap0To1 {
		// price = (token1ImaginaryReserves - amountOut) * PRICE_PRECISION / (token0ImaginaryReserves + amountIn)
		adjustedToken1 := new(big.Int).Sub(token1ImaginaryReserves, s.adjustToInternalDecimals(amountOut, false, dexVars))
		adjustedToken0 := new(big.Int).Add(token0ImaginaryReserves, s.adjustToInternalDecimals(amountIn, true, dexVars))
		currentPrice = new(big.Int).Mul(adjustedToken1, PricePrecision)
		currentPrice = currentPrice.Div(currentPrice, adjustedToken0)
	} else {
		// price = (token1ImaginaryReserves + amountIn) * PRICE_PRECISION / (token0ImaginaryReserves - amountOut)
		adjustedToken1 := new(big.Int).Add(token1ImaginaryReserves, s.adjustToInternalDecimals(amountIn, false, dexVars))
		adjustedToken0 := new(big.Int).Sub(token0ImaginaryReserves, s.adjustToInternalDecimals(amountOut, true, dexVars))
		currentPrice = new(big.Int).Mul(adjustedToken1, PricePrecision)
		currentPrice = currentPrice.Div(currentPrice, adjustedToken0)
	}

	// Update rebalancing status and check for state changes
	rebalancingStatus := new(big.Int).And(new(big.Int).Rsh(newPoolState.DexVariables, BitsDexLiteDexVariablesRebalancingStatus), X2)
	if rebalancingStatus.Cmp(big.NewInt(0)) > 0 {
		blockTimestamp := s.BlockTimestamp
		newRebalancingStatus := s.getRebalancingStatus(newPoolState.DexVariables, &newPoolState, s.DexId, rebalancingStatus, currentPrice, centerPrice, blockTimestamp)

		// Update centerPriceShift timestamp if rebalancing is active or center price shift is active
		centerPriceShiftActive := new(big.Int).And(new(big.Int).Rsh(newPoolState.DexVariables, BitsDexLiteDexVariablesCenterPriceShiftActive), X1)
		if newRebalancingStatus.Cmp(big.NewInt(1)) > 0 || centerPriceShiftActive.Cmp(big.NewInt(1)) == 0 {
			// Update last interaction timestamp: _centerPriceShift[dexId_] = _centerPriceShift[dexId_] & ~(X33 << BITS_DEX_LITE_CENTER_PRICE_SHIFT_LAST_INTERACTION_TIMESTAMP) | (block.timestamp << BITS_DEX_LITE_CENTER_PRICE_SHIFT_LAST_INTERACTION_TIMESTAMP)
			clearMask := new(big.Int).Lsh(X33, BitsDexLiteCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift = new(big.Int).AndNot(newPoolState.CenterPriceShift, clearMask)
			newTimestamp := new(big.Int).Lsh(big.NewInt(int64(blockTimestamp)), BitsDexLiteCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift = new(big.Int).Or(newPoolState.CenterPriceShift, newTimestamp)
		}
	}

	// Update dex variables with new supplies
	newPoolState.DexVariables = s.updateSuppliesInDexVariables(newPoolState.DexVariables, newToken0Supply, newToken1Supply)

	// Convert fee from adjusted precision back to token decimals
	fee := s.adjustFromInternalDecimals(feeInAdjusted, swap0To1, dexVars)

	return amountOut, newPoolState, fee, nil
}

// calculateSwapOutWithState implements the exact same logic as _swapOut in the FluidDexLite contract
// Returns: amountIn, newPoolState (all 4 variables), fee, error
func (s *PoolSimulator) calculateSwapOutWithState(swap0To1 bool, amountOut *big.Int, currentPoolState PoolState) (*big.Int, PoolState, *big.Int, error) {
	if currentPoolState.DexVariables.Sign() == 0 {
		return nil, PoolState{}, nil, ErrPoolNotInitialized
	}

	// Clone the current state - all 4 variables can potentially change
	newPoolState := currentPoolState.Clone()

	// Unpack dex variables to get current state
	dexVars := s.unpackDexVariables(currentPoolState.DexVariables)

	// Get current supplies
	token0AdjustedSupply := dexVars.Token0TotalSupplyAdjusted
	token1AdjustedSupply := dexVars.Token1TotalSupplyAdjusted

	// Calculate pricing and imaginary reserves with complete shifting logic
	centerPrice, token0ImaginaryReserves, token1ImaginaryReserves, err := s.getPricesAndReservesWithState(dexVars, &newPoolState)
	if err != nil {
		return nil, PoolState{}, nil, err
	}

	var amountIn *big.Int
	var newToken0Supply, newToken1Supply *big.Int
	var feeInAdjusted *big.Int // Fee in adjusted (9-decimal) precision

	if swap0To1 {
		// Adjust output amount to internal decimals
		amountOutAdjusted := s.adjustToInternalDecimals(amountOut, false, dexVars)

		// Validate amount
		if amountOutAdjusted.Cmp(FourDecimals) < 0 || amountOutAdjusted.Cmp(X60) > 0 {
			return nil, PoolState{}, nil, ErrInvalidAmountOut
		}
		if amountOutAdjusted.Cmp(new(big.Int).Div(token1ImaginaryReserves, big.NewInt(2))) > 0 {
			return nil, PoolState{}, nil, ErrExcessiveSwapAmount
		}

		// Calculate amount in: amountIn = (amountOut * iReserveIn) / (iReserveOut - amountOut)
		numerator := new(big.Int).Mul(amountOutAdjusted, token0ImaginaryReserves)
		denominator := new(big.Int).Sub(token1ImaginaryReserves, amountOutAdjusted)
		if denominator.Sign() <= 0 {
			return nil, PoolState{}, nil, ErrInsufficientReserve
		}
		amountIn = new(big.Int).Div(numerator, denominator)

		// Calculate fee and total amount in
		feeRate := dexVars.Fee
		feeDenominator := new(big.Int).Sub(SixDecimals, feeRate)
		if feeDenominator.Sign() <= 0 {
			return nil, PoolState{}, nil, ErrInvalidFeeRate
		}

		totalAmountIn := new(big.Int).Mul(amountIn, SixDecimals)
		totalAmountIn = totalAmountIn.Div(totalAmountIn, feeDenominator)
		feeInAdjusted = new(big.Int).Sub(totalAmountIn, amountIn)
		amountIn = totalAmountIn

		// Calculate revenue cut
		revenueCut := new(big.Int).Mul(feeInAdjusted, dexVars.RevenueCut)
		revenueCut = revenueCut.Div(revenueCut, TwoDecimals)

		// Update supplies
		newToken0Supply = new(big.Int).Add(token0AdjustedSupply, new(big.Int).Sub(amountIn, revenueCut))
		newToken1Supply = new(big.Int).Sub(token1AdjustedSupply, amountOutAdjusted)

		// Validate reserves
		if newToken1Supply.Sign() < 0 {
			return nil, PoolState{}, nil, ErrInsufficientReserve
		}

		// Check ratio
		minToken1 := new(big.Int).Mul(newToken0Supply, centerPrice)
		minToken1 = minToken1.Div(minToken1, new(big.Int).Mul(PricePrecision, big.NewInt(MinimumLiquiditySwap)))
		if newToken1Supply.Cmp(minToken1) < 0 {
			return nil, PoolState{}, nil, ErrTokenReservesRatioTooHigh
		}

		// Convert input back to token decimals
		amountIn = s.adjustFromInternalDecimals(amountIn, true, dexVars)

	} else {
		// Adjust output amount to internal decimals
		amountOutAdjusted := s.adjustToInternalDecimals(amountOut, true, dexVars)

		// Validate amount
		if amountOutAdjusted.Cmp(FourDecimals) < 0 || amountOutAdjusted.Cmp(X60) > 0 {
			return nil, PoolState{}, nil, ErrInvalidAmountOut
		}
		if amountOutAdjusted.Cmp(new(big.Int).Div(token0ImaginaryReserves, big.NewInt(2))) > 0 {
			return nil, PoolState{}, nil, ErrExcessiveSwapAmount
		}

		// Calculate amount in
		numerator := new(big.Int).Mul(amountOutAdjusted, token1ImaginaryReserves)
		denominator := new(big.Int).Sub(token0ImaginaryReserves, amountOutAdjusted)
		if denominator.Sign() <= 0 {
			return nil, PoolState{}, nil, ErrInsufficientReserve
		}
		amountIn = new(big.Int).Div(numerator, denominator)

		// Calculate fee and total amount in
		feeRate := dexVars.Fee
		feeDenominator := new(big.Int).Sub(SixDecimals, feeRate)
		if feeDenominator.Sign() <= 0 {
			return nil, PoolState{}, nil, ErrInvalidFeeRate
		}

		totalAmountIn := new(big.Int).Mul(amountIn, SixDecimals)
		totalAmountIn = totalAmountIn.Div(totalAmountIn, feeDenominator)
		feeInAdjusted = new(big.Int).Sub(totalAmountIn, amountIn)
		amountIn = totalAmountIn

		// Calculate revenue cut
		revenueCut := new(big.Int).Mul(feeInAdjusted, dexVars.RevenueCut)
		revenueCut = revenueCut.Div(revenueCut, TwoDecimals)

		// Update supplies
		newToken1Supply = new(big.Int).Add(token1AdjustedSupply, new(big.Int).Sub(amountIn, revenueCut))
		newToken0Supply = new(big.Int).Sub(token0AdjustedSupply, amountOutAdjusted)

		// Validate reserves
		if newToken0Supply.Sign() < 0 {
			return nil, PoolState{}, nil, ErrInsufficientReserve
		}

		// Check ratio
		minToken0 := new(big.Int).Mul(newToken1Supply, PricePrecision)
		minToken0 = minToken0.Div(minToken0, new(big.Int).Mul(centerPrice, big.NewInt(MinimumLiquiditySwap)))
		if newToken0Supply.Cmp(minToken0) < 0 {
			return nil, PoolState{}, nil, ErrTokenReservesRatioTooHigh
		}

		// Convert input back to token decimals
		amountIn = s.adjustFromInternalDecimals(amountIn, false, dexVars)
	}

	// Check for overflow
	if newToken0Supply.Cmp(X60) > 0 || newToken1Supply.Cmp(X60) > 0 {
		return nil, PoolState{}, nil, ErrAdjustedSupplyOverflow
	}

	// Convert fee from adjusted precision back to token decimals
	fee := s.adjustFromInternalDecimals(feeInAdjusted, swap0To1, dexVars)

	// Calculate the current price after swap for rebalancing status check (same logic as in calculateSwapInWithState)
	var currentPrice *big.Int
	if swap0To1 {
		// price = (token1ImaginaryReserves - amountOut) * PRICE_PRECISION / (token0ImaginaryReserves + amountIn)
		adjustedToken1 := new(big.Int).Sub(token1ImaginaryReserves, s.adjustToInternalDecimals(amountOut, false, dexVars))
		adjustedToken0 := new(big.Int).Add(token0ImaginaryReserves, s.adjustToInternalDecimals(amountIn, true, dexVars))
		currentPrice = new(big.Int).Mul(adjustedToken1, PricePrecision)
		currentPrice = currentPrice.Div(currentPrice, adjustedToken0)
	} else {
		// price = (token1ImaginaryReserves + amountIn) * PRICE_PRECISION / (token0ImaginaryReserves - amountOut)
		adjustedToken1 := new(big.Int).Add(token1ImaginaryReserves, s.adjustToInternalDecimals(amountIn, false, dexVars))
		adjustedToken0 := new(big.Int).Sub(token0ImaginaryReserves, s.adjustToInternalDecimals(amountOut, true, dexVars))
		currentPrice = new(big.Int).Mul(adjustedToken1, PricePrecision)
		currentPrice = currentPrice.Div(currentPrice, adjustedToken0)
	}

	// Update rebalancing status and check for state changes
	rebalancingStatus := new(big.Int).And(new(big.Int).Rsh(newPoolState.DexVariables, BitsDexLiteDexVariablesRebalancingStatus), X2)
	if rebalancingStatus.Cmp(big.NewInt(0)) > 0 {
		blockTimestamp := s.BlockTimestamp
		newRebalancingStatus := s.getRebalancingStatus(newPoolState.DexVariables, &newPoolState, s.DexId, rebalancingStatus, currentPrice, centerPrice, blockTimestamp)

		// Update centerPriceShift timestamp if rebalancing is active or center price shift is active
		centerPriceShiftActive := new(big.Int).And(new(big.Int).Rsh(newPoolState.DexVariables, BitsDexLiteDexVariablesCenterPriceShiftActive), X1)
		if newRebalancingStatus.Cmp(big.NewInt(1)) > 0 || centerPriceShiftActive.Cmp(big.NewInt(1)) == 0 {
			// Update last interaction timestamp
			clearMask := new(big.Int).Lsh(X33, BitsDexLiteCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift = new(big.Int).AndNot(newPoolState.CenterPriceShift, clearMask)
			newTimestamp := new(big.Int).Lsh(big.NewInt(int64(blockTimestamp)), BitsDexLiteCenterPriceShiftLastInteractionTimestamp)
			newPoolState.CenterPriceShift = new(big.Int).Or(newPoolState.CenterPriceShift, newTimestamp)
		}
	}

	// Update dex variables with new supplies
	newPoolState.DexVariables = s.updateSuppliesInDexVariables(newPoolState.DexVariables, newToken0Supply, newToken1Supply)

	return amountIn, newPoolState, fee, nil
}

// ------------------------------------------------------------------------------------------------
// FluidDexLite Shifting Helper Functions
// Implementing exactly the same logic as in the FluidDexLite contract helpers
// ------------------------------------------------------------------------------------------------

// calcShiftingDone implements _calcShiftingDone from the contract
func (s *PoolSimulator) calcShiftingDone(current, old, timePassed, shiftDuration *big.Int) *big.Int {
	if current.Cmp(old) > 0 {
		// current > old: old + ((current - old) * timePassed) / shiftDuration
		diff := new(big.Int).Sub(current, old)
		shifted := new(big.Int).Mul(diff, timePassed)
		shifted = shifted.Div(shifted, shiftDuration)
		return new(big.Int).Add(old, shifted)
	} else {
		// current <= old: old - ((old - current) * timePassed) / shiftDuration
		diff := new(big.Int).Sub(old, current)
		shifted := new(big.Int).Mul(diff, timePassed)
		shifted = shifted.Div(shifted, shiftDuration)
		return new(big.Int).Sub(old, shifted)
	}
}

// calcRangeShifting implements _calcRangeShifting from the contract
func (s *PoolSimulator) calcRangeShifting(upperRange, lowerRange *big.Int, poolState *PoolState, dexId [8]byte, blockTimestamp uint64) (*big.Int, *big.Int) {
	rangeShift := poolState.RangeShift

	// Extract shift data
	shiftDuration := new(big.Int).And(new(big.Int).Rsh(rangeShift, BitsDexLiteRangeShiftTimeToShift), X20)
	startTimestamp := new(big.Int).And(new(big.Int).Rsh(rangeShift, BitsDexLiteRangeShiftTimestamp), X33)

	currentTime := big.NewInt(int64(blockTimestamp))
	endTime := new(big.Int).Add(startTimestamp, shiftDuration)

	if currentTime.Cmp(endTime) >= 0 {
		// Shifting fully done - clear the range shift and deactivate
		poolState.RangeShift = big.NewInt(0) // delete _rangeShift[dexId_]

		// Clear range shift active bit in dexVariables
		mask := new(big.Int).Lsh(big.NewInt(1), BitsDexLiteDexVariablesRangePercentShiftActive)
		poolState.DexVariables = new(big.Int).AndNot(poolState.DexVariables, mask)

		return upperRange, lowerRange
	}

	timePassed := new(big.Int).Sub(currentTime, startTimestamp)

	// Extract old values
	oldUpperRange := new(big.Int).And(rangeShift, X14) // first 14 bits
	oldLowerRange := new(big.Int).And(new(big.Int).Rsh(rangeShift, BitsDexLiteRangeShiftOldLowerRangePercent), X14)

	// Calculate shifted values
	newUpperRange := s.calcShiftingDone(upperRange, oldUpperRange, timePassed, shiftDuration)
	newLowerRange := s.calcShiftingDone(lowerRange, oldLowerRange, timePassed, shiftDuration)

	return newUpperRange, newLowerRange
}

// calcThresholdShifting implements _calcThresholdShifting from the contract
func (s *PoolSimulator) calcThresholdShifting(upperThreshold, lowerThreshold *big.Int, poolState *PoolState, dexId [8]byte, blockTimestamp uint64) (*big.Int, *big.Int) {
	thresholdShift := poolState.ThresholdShift

	// Extract shift data
	shiftDuration := new(big.Int).And(new(big.Int).Rsh(thresholdShift, BitsDexLiteThresholdShiftTimeToShift), X20)
	startTimestamp := new(big.Int).And(new(big.Int).Rsh(thresholdShift, BitsDexLiteThresholdShiftTimestamp), X33)

	currentTime := big.NewInt(int64(blockTimestamp))
	endTime := new(big.Int).Add(startTimestamp, shiftDuration)

	if currentTime.Cmp(endTime) >= 0 {
		// Shifting fully done - clear the threshold shift and deactivate
		poolState.ThresholdShift = big.NewInt(0) // delete _thresholdShift[dexId_]

		// Clear threshold shift active bit in dexVariables
		mask := new(big.Int).Lsh(big.NewInt(1), BitsDexLiteDexVariablesThresholdPercentShiftActive)
		poolState.DexVariables = new(big.Int).AndNot(poolState.DexVariables, mask)

		return upperThreshold, lowerThreshold
	}

	timePassed := new(big.Int).Sub(currentTime, startTimestamp)

	// Extract old values - 7 bits each
	oldUpperThreshold := new(big.Int).And(thresholdShift, X7) // first 7 bits
	oldLowerThreshold := new(big.Int).And(new(big.Int).Rsh(thresholdShift, BitsDexLiteThresholdShiftOldLowerThresholdPercent), X7)

	// Calculate shifted values
	newUpperThreshold := s.calcShiftingDone(upperThreshold, oldUpperThreshold, timePassed, shiftDuration)
	newLowerThreshold := s.calcShiftingDone(lowerThreshold, oldLowerThreshold, timePassed, shiftDuration)

	return newUpperThreshold, newLowerThreshold
}

// getRebalancingStatus implements _getRebalancingStatus from the contract
func (s *PoolSimulator) getRebalancingStatus(dexVariables *big.Int, poolState *PoolState, dexId [8]byte, rebalancingStatus *big.Int, price, centerPrice *big.Int, blockTimestamp uint64) *big.Int {
	// Extract range percents from dexVariables
	upperRangePercent := new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperPercent), X14)
	lowerRangePercent := new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerPercent), X14)

	// Check if range shift is active and calculate if needed
	rangeShiftActive := new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRangePercentShiftActive), X1)
	if rangeShiftActive.Cmp(big.NewInt(1)) == 0 {
		upperRangePercent, lowerRangePercent = s.calcRangeShifting(upperRangePercent, lowerRangePercent, poolState, dexId, blockTimestamp)
	}

	// Calculate range prices
	// upperRangePrice = (centerPrice * FOUR_DECIMALS) / (FOUR_DECIMALS - upperRangePercent)
	denominator := new(big.Int).Sub(FourDecimals, upperRangePercent)
	upperRangePrice := new(big.Int).Mul(centerPrice, FourDecimals)
	upperRangePrice = upperRangePrice.Div(upperRangePrice, denominator)

	// lowerRangePrice = (centerPrice * (FOUR_DECIMALS - lowerRangePercent)) / FOUR_DECIMALS
	numerator := new(big.Int).Sub(FourDecimals, lowerRangePercent)
	lowerRangePrice := new(big.Int).Mul(centerPrice, numerator)
	lowerRangePrice = lowerRangePrice.Div(lowerRangePrice, FourDecimals)

	// Extract threshold percents
	upperThresholdPercent := new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperShiftThresholdPercent), X7)
	lowerThresholdPercent := new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerShiftThresholdPercent), X7)

	// Check if threshold shift is active and calculate if needed
	thresholdShiftActive := new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesThresholdPercentShiftActive), X1)
	if thresholdShiftActive.Cmp(big.NewInt(1)) == 0 {
		upperThresholdPercent, lowerThresholdPercent = s.calcThresholdShifting(upperThresholdPercent, lowerThresholdPercent, poolState, dexId, blockTimestamp)
	}

	// Calculate threshold prices
	// upperThreshold = centerPrice + ((upperRangePrice - centerPrice) * (TWO_DECIMALS - upperThresholdPercent)) / TWO_DECIMALS
	rangeDiff := new(big.Int).Sub(upperRangePrice, centerPrice)
	thresholdFactor := new(big.Int).Sub(TwoDecimals, upperThresholdPercent)
	adjustment := new(big.Int).Mul(rangeDiff, thresholdFactor)
	adjustment = adjustment.Div(adjustment, TwoDecimals)
	upperThreshold := new(big.Int).Add(centerPrice, adjustment)

	// lowerThreshold = centerPrice - ((centerPrice - lowerRangePrice) * (TWO_DECIMALS - lowerThresholdPercent)) / TWO_DECIMALS
	rangeDiff = new(big.Int).Sub(centerPrice, lowerRangePrice)
	thresholdFactor = new(big.Int).Sub(TwoDecimals, lowerThresholdPercent)
	adjustment = new(big.Int).Mul(rangeDiff, thresholdFactor)
	adjustment = adjustment.Div(adjustment, TwoDecimals)
	lowerThreshold := new(big.Int).Sub(centerPrice, adjustment)

	// Check thresholds and update rebalancing status
	if price.Cmp(upperThreshold) > 0 {
		if rebalancingStatus.Cmp(big.NewInt(2)) != 0 {
			// Update dexVariables with rebalancing status = 2
			clearMask := new(big.Int).Lsh(X2, BitsDexLiteDexVariablesRebalancingStatus)
			poolState.DexVariables = new(big.Int).AndNot(poolState.DexVariables, clearMask)
			newStatus := new(big.Int).Lsh(big.NewInt(2), BitsDexLiteDexVariablesRebalancingStatus)
			poolState.DexVariables = new(big.Int).Or(poolState.DexVariables, newStatus)
			return big.NewInt(2)
		}
	} else if price.Cmp(lowerThreshold) < 0 {
		if rebalancingStatus.Cmp(big.NewInt(3)) != 0 {
			// Update dexVariables with rebalancing status = 3
			clearMask := new(big.Int).Lsh(X2, BitsDexLiteDexVariablesRebalancingStatus)
			poolState.DexVariables = new(big.Int).AndNot(poolState.DexVariables, clearMask)
			newStatus := new(big.Int).Lsh(big.NewInt(3), BitsDexLiteDexVariablesRebalancingStatus)
			poolState.DexVariables = new(big.Int).Or(poolState.DexVariables, newStatus)
			return big.NewInt(3)
		}
	} else {
		// Price is within normal range
		if rebalancingStatus.Cmp(big.NewInt(1)) != 0 {
			// Update dexVariables with rebalancing status = 1
			clearMask := new(big.Int).Lsh(X2, BitsDexLiteDexVariablesRebalancingStatus)
			poolState.DexVariables = new(big.Int).AndNot(poolState.DexVariables, clearMask)
			newStatus := new(big.Int).Lsh(big.NewInt(1), BitsDexLiteDexVariablesRebalancingStatus)
			poolState.DexVariables = new(big.Int).Or(poolState.DexVariables, newStatus)
			return big.NewInt(1)
		}
	}

	return rebalancingStatus
}

// unpackDexVariables unpacks the packed dex variables exactly like in the contract
func (s *PoolSimulator) unpackDexVariables(dexVariables *big.Int) *UnpackedDexVariables {
	return &UnpackedDexVariables{
		Fee:                         new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesFee), X13),
		RevenueCut:                  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRevenueCut), X7),
		RebalancingStatus:           new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRebalancingStatus), X2),
		CenterPriceShiftActive:      new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPriceShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		CenterPrice:                 new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPrice), X40),
		CenterPriceContractAddress:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPriceContractAddress), X19),
		RangePercentShiftActive:     new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRangePercentShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		UpperPercent:                new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperPercent), X14),
		LowerPercent:                new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerPercent), X14),
		ThresholdPercentShiftActive: new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesThresholdPercentShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		UpperShiftThresholdPercent:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperShiftThresholdPercent), X7),
		LowerShiftThresholdPercent:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerShiftThresholdPercent), X7),
		Token0Decimals:              new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken0Decimals), X5),
		Token1Decimals:              new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken1Decimals), X5),
		Token0TotalSupplyAdjusted:   new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken0TotalSupplyAdjusted), X60),
		Token1TotalSupplyAdjusted:   new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken1TotalSupplyAdjusted), X60),
	}
}

// Helper function to adjust amounts to internal decimals (TOKENS_DECIMALS_PRECISION = 9)
func (s *PoolSimulator) adjustToInternalDecimals(amount *big.Int, isToken0 bool, dexVars *UnpackedDexVariables) *big.Int {
	var decimals uint64
	if isToken0 {
		decimals = dexVars.Token0Decimals.Uint64()
	} else {
		decimals = dexVars.Token1Decimals.Uint64()
	}

	if decimals > TokensDecimalsPrecision {
		return new(big.Int).Div(amount, tenPow(int(decimals-TokensDecimalsPrecision)))
	} else {
		return new(big.Int).Mul(amount, tenPow(int(TokensDecimalsPrecision-decimals)))
	}
}

// Helper function to adjust amounts from internal decimals back to token decimals
func (s *PoolSimulator) adjustFromInternalDecimals(amount *big.Int, isToken0 bool, dexVars *UnpackedDexVariables) *big.Int {
	var decimals uint64
	if isToken0 {
		decimals = dexVars.Token0Decimals.Uint64()
	} else {
		decimals = dexVars.Token1Decimals.Uint64()
	}

	if decimals > TokensDecimalsPrecision {
		return new(big.Int).Mul(amount, tenPow(int(decimals-TokensDecimalsPrecision)))
	} else {
		return new(big.Int).Div(amount, tenPow(int(TokensDecimalsPrecision-decimals)))
	}
}

// expandCenterPrice expands the compressed center price
func (s *PoolSimulator) expandCenterPrice(centerPrice *big.Int) *big.Int {
	coefficient := new(big.Int).Rsh(centerPrice, uint(DefaultExponentSize))
	exponent := new(big.Int).And(centerPrice, DefaultExponentMask)
	return new(big.Int).Lsh(coefficient, uint(exponent.Uint64()))
}

// getPricesAndReservesWithState implements complete _getPricesAndReserves with all shifting logic
func (s *PoolSimulator) getPricesAndReservesWithState(dexVars *UnpackedDexVariables, poolState *PoolState) (*big.Int, *big.Int, *big.Int, error) {
	// Use the actual block timestamp from when the pool state was fetched
	blockTimestamp := s.BlockTimestamp

	// Check for external center price functionality that we don't support
	centerPriceShiftActive := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesCenterPriceShiftActive), X1)
	centerPriceContractAddress := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesCenterPriceContractAddress), X19)

	if centerPriceShiftActive.Cmp(big.NewInt(1)) == 0 || centerPriceContractAddress.Sign() > 0 {
		return nil, nil, nil, ErrExternalCenterPriceNotSupported
	}

	// Extract center price with exponential encoding (static price only)
	centerPriceRaw := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesCenterPrice), X40)
	exponent := new(big.Int).And(centerPriceRaw, DefaultExponentMask)
	coefficient := new(big.Int).Rsh(centerPriceRaw, uint(DefaultExponentSize))
	centerPrice := new(big.Int).Lsh(coefficient, uint(exponent.Uint64()))

	// Extract range percents
	upperRangePercent := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesUpperPercent), X14)
	lowerRangePercent := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesLowerPercent), X14)

	// Check if range shift is active
	rangeShiftActive := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesRangePercentShiftActive), X1)
	if rangeShiftActive.Cmp(big.NewInt(1)) == 0 {
		// An active range shift is going on
		upperRangePercent, lowerRangePercent = s.calcRangeShifting(upperRangePercent, lowerRangePercent, poolState, s.DexId, blockTimestamp)
	}

	// Calculate range prices
	var upperRangePrice, lowerRangePrice *big.Int
	// upperRangePrice = (centerPrice * FOUR_DECIMALS) / (FOUR_DECIMALS - upperRangePercent)
	denominator := new(big.Int).Sub(FourDecimals, upperRangePercent)
	upperRangePrice = new(big.Int).Mul(centerPrice, FourDecimals)
	upperRangePrice = upperRangePrice.Div(upperRangePrice, denominator)

	// lowerRangePrice = (centerPrice * (FOUR_DECIMALS - lowerRangePercent)) / FOUR_DECIMALS
	numerator := new(big.Int).Sub(FourDecimals, lowerRangePercent)
	lowerRangePrice = new(big.Int).Mul(centerPrice, numerator)
	lowerRangePrice = lowerRangePrice.Div(lowerRangePrice, FourDecimals)

	// Handle rebalancing if status > 1
	rebalancingStatus := new(big.Int).And(new(big.Int).Rsh(poolState.DexVariables, BitsDexLiteDexVariablesRebalancingStatus), X2)
	if rebalancingStatus.Cmp(big.NewInt(1)) > 0 {
		centerPriceShift := poolState.CenterPriceShift
		if centerPriceShift.Sign() > 0 {
			shiftingTime := new(big.Int).And(new(big.Int).Rsh(centerPriceShift, BitsDexLiteCenterPriceShiftShiftingTime), X24)
			lastInteractionTimestamp := new(big.Int).And(centerPriceShift, X33) // BitsDexLiteCenterPriceShiftLastInteractionTimestamp = 0
			timeElapsed := new(big.Int).Sub(big.NewInt(int64(blockTimestamp)), lastInteractionTimestamp)

			if rebalancingStatus.Cmp(big.NewInt(2)) == 0 {
				// Price shifting towards upper range
				if timeElapsed.Cmp(shiftingTime) < 0 {
					// Partial shift: centerPrice + ((upperRangePrice - centerPrice) * timeElapsed) / shiftingTime
					diff := new(big.Int).Sub(upperRangePrice, centerPrice)
					shift := new(big.Int).Mul(diff, timeElapsed)
					shift = shift.Div(shift, shiftingTime)
					centerPrice = new(big.Int).Add(centerPrice, shift)
				} else {
					// 100% price shifted
					centerPrice = new(big.Int).Set(upperRangePrice)
				}
			} else if rebalancingStatus.Cmp(big.NewInt(3)) == 0 {
				// Price shifting towards lower range
				if timeElapsed.Cmp(shiftingTime) < 0 {
					// Partial shift: centerPrice - ((centerPrice - lowerRangePrice) * timeElapsed) / shiftingTime
					diff := new(big.Int).Sub(centerPrice, lowerRangePrice)
					shift := new(big.Int).Mul(diff, timeElapsed)
					shift = shift.Div(shift, shiftingTime)
					centerPrice = new(big.Int).Sub(centerPrice, shift)
				} else {
					// 100% price shifted
					centerPrice = new(big.Int).Set(lowerRangePrice)
				}
			}

			// Check min/max bounds if rebalancing actually happened
			maxCenterPrice := new(big.Int).And(new(big.Int).Rsh(centerPriceShift, BitsDexLiteCenterPriceShiftMaxCenterPrice), X28)
			maxCenterPriceExpanded := s.expandCenterPrice(maxCenterPrice)
			if centerPrice.Cmp(maxCenterPriceExpanded) > 0 {
				centerPrice = maxCenterPriceExpanded
			} else {
				minCenterPrice := new(big.Int).And(new(big.Int).Rsh(centerPriceShift, BitsDexLiteCenterPriceShiftMinCenterPrice), X28)
				minCenterPriceExpanded := s.expandCenterPrice(minCenterPrice)
				if centerPrice.Cmp(minCenterPriceExpanded) < 0 {
					centerPrice = minCenterPriceExpanded
				}
			}

			// Update range prices as center price moved
			denominator = new(big.Int).Sub(FourDecimals, upperRangePercent)
			upperRangePrice = new(big.Int).Mul(centerPrice, FourDecimals)
			upperRangePrice = upperRangePrice.Div(upperRangePrice, denominator)

			numerator = new(big.Int).Sub(FourDecimals, lowerRangePercent)
			lowerRangePrice = new(big.Int).Mul(centerPrice, numerator)
			lowerRangePrice = lowerRangePrice.Div(lowerRangePrice, FourDecimals)
		}
	}

	// Calculate geometric mean price
	var geometricMeanPrice *big.Int
	threshold1e38 := new(big.Int)
	threshold1e38.SetString("100000000000000000000000000000000000000", 10) // 1e38
	if upperRangePrice.Cmp(threshold1e38) < 0 {
		// upperRangePrice * lowerRangePrice < 1e76 (within safe limits)
		product := new(big.Int).Mul(upperRangePrice, lowerRangePrice)
		geometricMeanPrice = s.sqrt(product)
	} else {
		// Scale down to prevent overflow
		scaledUpper := new(big.Int).Div(upperRangePrice, big.NewInt(1e18))
		scaledLower := new(big.Int).Div(lowerRangePrice, big.NewInt(1e18))
		product := new(big.Int).Mul(scaledUpper, scaledLower)
		geometricMeanPrice = new(big.Int).Mul(s.sqrt(product), big.NewInt(1e18))
	}

	// Get token supplies
	token0Supply := dexVars.Token0TotalSupplyAdjusted
	token1Supply := dexVars.Token1TotalSupplyAdjusted

	// Calculate imaginary reserves
	var token0ImaginaryReserves, token1ImaginaryReserves *big.Int

	if geometricMeanPrice.Cmp(PricePrecision) < 0 { // < 1e27
		token0ImaginaryReserves, token1ImaginaryReserves = s.calculateReservesOutsideRange(
			geometricMeanPrice, upperRangePrice, token0Supply, token1Supply)
	} else {
		// Inverse calculation for large prices
		inverseGeometricMean := new(big.Int).Div(new(big.Int).Mul(PricePrecision, PricePrecision), geometricMeanPrice) // 1e54 / geometricMeanPrice
		inverseLowerRange := new(big.Int).Div(new(big.Int).Mul(PricePrecision, PricePrecision), lowerRangePrice)       // 1e54 / lowerRangePrice

		token1ImaginaryReserves, token0ImaginaryReserves = s.calculateReservesOutsideRange(
			inverseGeometricMean, inverseLowerRange, token1Supply, token0Supply)
	}

	// Add real supplies to imaginary reserves
	token0ImaginaryReserves = new(big.Int).Add(token0ImaginaryReserves, token0Supply)
	token1ImaginaryReserves = new(big.Int).Add(token1ImaginaryReserves, token1Supply)

	return centerPrice, token0ImaginaryReserves, token1ImaginaryReserves, nil
}

// calculateReservesOutsideRange calculates reserves outside the range (simplified)
func (s *PoolSimulator) calculateReservesOutsideRange(gp, pa, rx, ry *big.Int) (*big.Int, *big.Int) {
	// Simplified calculation based on the contract formula
	p1 := new(big.Int).Sub(pa, gp)
	if p1.Sign() <= 0 {
		return big.NewInt(0), big.NewInt(0)
	}

	p2 := new(big.Int).Mul(gp, rx)
	p2 = p2.Add(p2, new(big.Int).Mul(ry, PricePrecision))
	p2 = p2.Div(p2, new(big.Int).Mul(big.NewInt(2), p1))

	discriminant := new(big.Int).Mul(rx, ry)
	discriminant = discriminant.Mul(discriminant, PricePrecision)
	discriminant = discriminant.Div(discriminant, p1)
	discriminant = discriminant.Add(discriminant, new(big.Int).Mul(p2, p2))

	xa := new(big.Int).Add(p2, s.sqrt(discriminant))
	yb := new(big.Int).Mul(xa, gp)
	yb = yb.Div(yb, PricePrecision)

	return xa, yb
}

// sqrt calculates square root using Newton's method
func (s *PoolSimulator) sqrt(value *big.Int) *big.Int {
	if value.Sign() < 0 {
		return big.NewInt(0)
	}
	if value.Cmp(big.NewInt(2)) < 0 {
		return new(big.Int).Set(value)
	}

	x := new(big.Int).Set(value)
	result := new(big.Int).Set(value)

	for x.Sign() > 0 {
		x = new(big.Int).Add(result, new(big.Int).Div(value, result))
		x = x.Div(x, big.NewInt(2))
		if x.Cmp(result) >= 0 {
			break
		}
		result = new(big.Int).Set(x)
	}

	return result
}

// updateSuppliesInDexVariables updates the token supplies in the packed dex variables
func (s *PoolSimulator) updateSuppliesInDexVariables(dexVariables, token0Supply, token1Supply *big.Int) *big.Int {
	// Clear existing supply bits
	clearMask0 := new(big.Int).Lsh(X60, BitsDexLiteDexVariablesToken0TotalSupplyAdjusted)
	clearMask1 := new(big.Int).Lsh(X60, BitsDexLiteDexVariablesToken1TotalSupplyAdjusted)
	clearMask := new(big.Int).Or(clearMask0, clearMask1)
	clearMask.Not(clearMask)

	newDexVars := new(big.Int).And(dexVariables, clearMask)

	// Set new supplies
	newSupply0 := new(big.Int).Lsh(new(big.Int).And(token0Supply, X60), BitsDexLiteDexVariablesToken0TotalSupplyAdjusted)
	newSupply1 := new(big.Int).Lsh(new(big.Int).And(token1Supply, X60), BitsDexLiteDexVariablesToken1TotalSupplyAdjusted)

	newDexVars.Or(newDexVars, newSupply0)
	newDexVars.Or(newDexVars, newSupply1)

	return newDexVars
}
