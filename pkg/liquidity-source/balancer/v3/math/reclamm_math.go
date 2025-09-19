package math

import (
	"errors"
	"fmt"

	"github.com/holiman/uint256"
)

var (
	ErrReClammAmountOutGreaterThanBalance = errors.New("reClammMath: AmountOutGreaterThanBalance")
	thirtyDaysSeconds                     = uint256.NewInt(30 * 24 * 60 * 60) // 30 days in seconds
)

var ReClammMath *reClammMath

type reClammMath struct{}

// PriceRatioState represents the state of price ratio updates
type PriceRatioState struct {
	PriceRatioUpdateStartTime *uint256.Int
	PriceRatioUpdateEndTime   *uint256.Int
	StartFourthRootPriceRatio *uint256.Int
	EndFourthRootPriceRatio   *uint256.Int
}

// VirtualBalancesResult represents the result of virtual balance computation
type VirtualBalancesResult struct {
	CurrentVirtualBalanceA *uint256.Int
	CurrentVirtualBalanceB *uint256.Int
	Changed                bool
}

// CenterednessResult represents the result of centeredness computation
type CenterednessResult struct {
	PoolCenteredness  *uint256.Int
	IsPoolAboveCenter bool
}

// PriceRangeResult represents the result of price range computation
type PriceRangeResult struct {
	MinPrice *uint256.Int
	MaxPrice *uint256.Int
}

// VirtualBalancesUpdatingPriceRatioResult represents the result of virtual balance computation when updating price ratio
type VirtualBalancesUpdatingPriceRatioResult struct {
	VirtualBalanceA *uint256.Int
	VirtualBalanceB *uint256.Int
}

const (
	// Token indices
	TokenA = 0
	TokenB = 1

	// Thirty days in seconds
	ThirtyDaysSeconds = 30 * 24 * 60 * 60 // 2,592,000 seconds
)

func init() {
	ReClammMath = &reClammMath{}
}

// ComputeCurrentVirtualBalances computes the current virtual balances based on timestamp and price ratio state
func (r *reClammMath) ComputeCurrentVirtualBalances(
	currentTimestamp *uint256.Int,
	balancesScaled18 []*uint256.Int,
	lastVirtualBalanceA *uint256.Int,
	lastVirtualBalanceB *uint256.Int,
	dailyPriceShiftBase *uint256.Int,
	lastTimestamp *uint256.Int,
	centerednessMargin *uint256.Int,
	priceRatioState *PriceRatioState,
) (*VirtualBalancesResult, error) {
	if lastTimestamp.Eq(currentTimestamp) {
		return &VirtualBalancesResult{
			CurrentVirtualBalanceA: lastVirtualBalanceA,
			CurrentVirtualBalanceB: lastVirtualBalanceB,
			Changed:                false,
		}, nil
	}

	currentVirtualBalanceA := new(uint256.Int).Set(lastVirtualBalanceA)
	currentVirtualBalanceB := new(uint256.Int).Set(lastVirtualBalanceB)

	currentFourthRootPriceRatio, err := r.ComputeFourthRootPriceRatio(
		currentTimestamp,
		priceRatioState.StartFourthRootPriceRatio,
		priceRatioState.EndFourthRootPriceRatio,
		priceRatioState.PriceRatioUpdateStartTime,
		priceRatioState.PriceRatioUpdateEndTime,
	)
	if err != nil {
		return nil, err
	}

	changed := false

	// If the price ratio is updating, shrink/expand the price interval by recalculating the virtual balances.
	if currentTimestamp.Gt(priceRatioState.PriceRatioUpdateStartTime) &&
		lastTimestamp.Lt(priceRatioState.PriceRatioUpdateEndTime) {

		result, err := r.ComputeVirtualBalancesUpdatingPriceRatio(
			currentFourthRootPriceRatio,
			balancesScaled18,
			lastVirtualBalanceA,
			lastVirtualBalanceB,
		)
		if err != nil {
			return nil, err
		}

		currentVirtualBalanceA = result.VirtualBalanceA
		currentVirtualBalanceB = result.VirtualBalanceB
		changed = true
	}

	centerednessResult, err := r.ComputeCenteredness(
		balancesScaled18,
		currentVirtualBalanceA,
		currentVirtualBalanceB,
	)
	if err != nil {
		return nil, err
	}

	// If the pool is outside the target range, track the market price by moving the price interval.
	if centerednessResult.PoolCenteredness.Lt(centerednessMargin) {
		newVirtualBalanceA, newVirtualBalanceB, err := r.ComputeVirtualBalancesUpdatingPriceRange(
			balancesScaled18,
			currentVirtualBalanceA,
			currentVirtualBalanceB,
			centerednessResult.IsPoolAboveCenter,
			dailyPriceShiftBase,
			currentTimestamp,
			lastTimestamp,
		)
		if err != nil {
			return nil, err
		}

		currentVirtualBalanceA = newVirtualBalanceA
		currentVirtualBalanceB = newVirtualBalanceB
		changed = true
	}

	return &VirtualBalancesResult{
		CurrentVirtualBalanceA: currentVirtualBalanceA,
		CurrentVirtualBalanceB: currentVirtualBalanceB,
		Changed:                changed,
	}, nil
}

// ComputeVirtualBalancesUpdatingPriceRatio computes the virtual balances when the price ratio is updating
func (r *reClammMath) ComputeVirtualBalancesUpdatingPriceRatio(
	currentFourthRootPriceRatio *uint256.Int,
	balancesScaled18 []*uint256.Int,
	lastVirtualBalanceA *uint256.Int,
	lastVirtualBalanceB *uint256.Int,
) (*VirtualBalancesUpdatingPriceRatioResult, error) {
	// Compute the current pool centeredness, which will remain constant.
	centerednessResult, err := r.ComputeCenteredness(
		balancesScaled18,
		lastVirtualBalanceA,
		lastVirtualBalanceB,
	)
	if err != nil {
		return nil, err
	}

	// The overvalued token is the one with a lower token balance (therefore, rarer and more valuable).
	var balanceTokenUndervalued, lastVirtualBalanceUndervalued, lastVirtualBalanceOvervalued *uint256.Int
	var isPoolAboveCenter bool

	if centerednessResult.IsPoolAboveCenter {
		balanceTokenUndervalued = balancesScaled18[TokenA]
		lastVirtualBalanceUndervalued = lastVirtualBalanceA
		lastVirtualBalanceOvervalued = lastVirtualBalanceB
		isPoolAboveCenter = true
	} else {
		balanceTokenUndervalued = balancesScaled18[TokenB]
		lastVirtualBalanceUndervalued = lastVirtualBalanceB
		lastVirtualBalanceOvervalued = lastVirtualBalanceA
		isPoolAboveCenter = false
	}

	// Calculate sqrtPriceRatio = currentFourthRootPriceRatio^2
	sqrtPriceRatio, err := FixPoint.MulDown(currentFourthRootPriceRatio, currentFourthRootPriceRatio)
	if err != nil {
		return nil, err
	}

	// Calculate the discriminant for the quadratic equation
	// discriminant = poolCenteredness * (poolCenteredness + 4 * sqrtPriceRatio - 2 * WAD) + WAD^2
	term2, err := FixPoint.Mul(sqrtPriceRatio, U4)
	if err != nil {
		return nil, err
	}

	term3, err := FixPoint.Sub(term2, U2e18)
	if err != nil {
		return nil, err
	}

	term4, err := FixPoint.Add(centerednessResult.PoolCenteredness, term3)
	if err != nil {
		return nil, err
	}

	discriminant, err := FixPoint.Mul(centerednessResult.PoolCenteredness, term4)
	if err != nil {
		return nil, err
	}

	discriminant, err = FixPoint.Add(discriminant, U1e36)
	if err != nil {
		return nil, err
	}

	// Calculate square root of discriminant
	sqrtDiscriminant, err := r.Sqrt(discriminant)
	if err != nil {
		return nil, err
	}

	// Calculate virtualBalanceUndervalued using the simplified Bhaskara formula
	// Vu = Ru(1 + C + sqrt(1 + C (C + 4 Q0 - 2))) / 2(Q0 - 1)
	numerator, err := FixPoint.Add(U1e18, centerednessResult.PoolCenteredness)
	if err != nil {
		return nil, err
	}

	numerator, err = FixPoint.Add(numerator, sqrtDiscriminant)
	if err != nil {
		return nil, err
	}

	numerator, err = FixPoint.Mul(balanceTokenUndervalued, numerator)
	if err != nil {
		return nil, err
	}

	denominator, err := FixPoint.Sub(sqrtPriceRatio, U1e18)
	if err != nil {
		return nil, err
	}

	denominator, err = FixPoint.Mul(denominator, U2)
	if err != nil {
		return nil, err
	}

	virtualBalanceUndervalued, err := FixPoint.Div(numerator, denominator)
	if err != nil {
		return nil, err
	}

	// Calculate virtualBalanceOvervalued maintaining the ratio
	virtualBalanceOvervalued, err := FixPoint.Mul(virtualBalanceUndervalued, lastVirtualBalanceOvervalued)
	if err != nil {
		return nil, err
	}

	virtualBalanceOvervalued, err = FixPoint.Div(virtualBalanceOvervalued, lastVirtualBalanceUndervalued)
	if err != nil {
		return nil, err
	}

	var virtualBalanceA, virtualBalanceB *uint256.Int
	if isPoolAboveCenter {
		virtualBalanceA = virtualBalanceUndervalued
		virtualBalanceB = virtualBalanceOvervalued
	} else {
		virtualBalanceA = virtualBalanceOvervalued
		virtualBalanceB = virtualBalanceUndervalued
	}

	return &VirtualBalancesUpdatingPriceRatioResult{
		VirtualBalanceA: virtualBalanceA,
		VirtualBalanceB: virtualBalanceB,
	}, nil
}

// ComputeVirtualBalancesUpdatingPriceRange computes virtual balances when updating price range
// computeVirtualBalancesUpdatingPriceRange converts the TypeScript function to Go
func (r *reClammMath) ComputeVirtualBalancesUpdatingPriceRange(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
	isPoolAboveCenter bool,
	dailyPriceShiftBase *uint256.Int,
	currentTimestamp *uint256.Int,
	lastTimestamp *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {

	priceRatio, err := r.ComputePriceRatio(balancesScaled18, virtualBalanceA, virtualBalanceB)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute price ratio: %w", err)
	}

	sqrtPriceRatio, err := r.SqrtScaled18(priceRatio)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute sqrt price ratio: %w", err)
	}

	// The overvalued token is the one with a lower token balance (therefore, rarer and more valuable).
	var balancesScaledUndervalued, balancesScaledOvervalued *uint256.Int
	var virtualBalanceUndervalued, virtualBalanceOvervalued *uint256.Int

	if isPoolAboveCenter {
		balancesScaledUndervalued = balancesScaled18[0]
		balancesScaledOvervalued = balancesScaled18[1]
		virtualBalanceOvervalued = new(uint256.Int).Set(virtualBalanceB)
	} else {
		balancesScaledUndervalued = balancesScaled18[1]
		balancesScaledOvervalued = balancesScaled18[0]
		virtualBalanceOvervalued = new(uint256.Int).Set(virtualBalanceA)
	}

	// +-----------------------------------------+
	// |                      (Tc - Tl)          |
	// |      Vo = Vo * (Psb)^                   |
	// +-----------------------------------------+
	// |  Where:                                 |
	// |    Vo = Virtual balance overvalued      |
	// |    Psb = Price shift daily rate base    |
	// |    Tc = Current timestamp               |
	// |    Tl = Last timestamp                  |
	// +-----------------------------------------+
	// |               Ru * (Vo + Ro)            |
	// |      Vu = ----------------------        |
	// |             (Qo - 1) * Vo - Ro          |
	// +-----------------------------------------+
	// |  Where:                                 |
	// |    Vu = Virtual balance undervalued     |
	// |    Vo = Virtual balance overvalued      |
	// |    Ru = Real balance undervalued        |
	// |    Ro = Real balance overvalued         |
	// |    Qo = Square root of price ratio      |
	// +-----------------------------------------+

	// Cap the duration (time between operations) at 30 days, to ensure `powDown` does not overflow.
	duration := new(uint256.Int).Sub(currentTimestamp, lastTimestamp)
	duration, err = FixPoint.Min(duration, thirtyDaysSeconds)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute min duration: %w", err)
	}

	// Calculate virtualBalanceOvervalued * (dailyPriceShiftBase ^ (duration * WAD))
	exponent := new(uint256.Int).Mul(duration, U1e18)
	powerResult, err := FixPoint.PowDown(dailyPriceShiftBase, exponent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute power: %w", err)
	}

	virtualBalanceOvervalued, err = FixPoint.MulDown(virtualBalanceOvervalued, powerResult)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to multiply virtual balance overvalued: %w", err)
	}

	// Ensure that Vo does not go below the minimum allowed value (corresponding to centeredness == 1).
	sqrtSqrtPriceRatio, err := r.SqrtScaled18(sqrtPriceRatio)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute sqrt of sqrt price ratio: %w", err)
	}

	denominator := new(uint256.Int).Sub(sqrtSqrtPriceRatio, U1e18)
	minVirtualBalance, err := FixPoint.DivDown(balancesScaledOvervalued, denominator)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute min virtual balance: %w", err)
	}

	virtualBalanceOvervalued, err = FixPoint.Max(virtualBalanceOvervalued, minVirtualBalance)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute max virtual balance: %w", err)
	}

	// Calculate virtualBalanceUndervalued
	// Vu = (Ru * (Vo + Ro)) / ((Qo - 1) * Vo - Ro)
	numerator := new(uint256.Int).Add(virtualBalanceOvervalued, balancesScaledOvervalued)
	numerator = new(uint256.Int).Mul(balancesScaledUndervalued, numerator)

	sqrtMinusOne := new(uint256.Int).Sub(sqrtPriceRatio, U1e18)
	denominatorPart1, err := FixPoint.MulDown(sqrtMinusOne, virtualBalanceOvervalued)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute denominator part 1: %w", err)
	}

	denominator = new(uint256.Int).Sub(denominatorPart1, balancesScaledOvervalued)
	virtualBalanceUndervalued = new(uint256.Int).Div(numerator, denominator)

	if isPoolAboveCenter {
		return virtualBalanceUndervalued, virtualBalanceOvervalued, nil
	} else {
		return virtualBalanceOvervalued, virtualBalanceUndervalued, nil
	}
}

// ComputePriceRatio computes the price ratio of the pool
func (r *reClammMath) ComputePriceRatio(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
) (*uint256.Int, error) {
	priceRange, err := r.ComputePriceRange(balancesScaled18, virtualBalanceA, virtualBalanceB)
	if err != nil {
		return nil, err
	}

	return FixPoint.DivUp(priceRange.MaxPrice, priceRange.MinPrice)
}

// ComputePriceRange computes the minimum and maximum prices for the pool
func (r *reClammMath) ComputePriceRange(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
) (*PriceRangeResult, error) {
	currentInvariant, err := r.ComputeInvariant(balancesScaled18, virtualBalanceA, virtualBalanceB, false)
	if err != nil {
		return nil, err
	}

	// P_min(a) = Vb^2 / invariant
	virtualBalanceBSquared, err := FixPoint.Mul(virtualBalanceB, virtualBalanceB)
	if err != nil {
		return nil, err
	}

	minPrice, err := FixPoint.Div(virtualBalanceBSquared, currentInvariant)
	if err != nil {
		return nil, err
	}

	// P_max(a) = invariant / Va^2
	virtualBalanceASquared, err := FixPoint.MulDown(virtualBalanceA, virtualBalanceA)
	if err != nil {
		return nil, err
	}

	maxPrice, err := FixPoint.DivDown(currentInvariant, virtualBalanceASquared)
	if err != nil {
		return nil, err
	}

	return &PriceRangeResult{
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

// ComputeFourthRootPriceRatio computes the fourth root of price ratio based on time
func (r *reClammMath) ComputeFourthRootPriceRatio(
	currentTime *uint256.Int,
	startFourthRootPriceRatio *uint256.Int,
	endFourthRootPriceRatio *uint256.Int,
	priceRatioUpdateStartTime *uint256.Int,
	priceRatioUpdateEndTime *uint256.Int,
) (*uint256.Int, error) {
	// if start and end time are the same, return end value.
	if currentTime.Gt(priceRatioUpdateEndTime) || currentTime.Eq(priceRatioUpdateEndTime) {
		return endFourthRootPriceRatio, nil
	} else if currentTime.Lt(priceRatioUpdateStartTime) || currentTime.Eq(priceRatioUpdateStartTime) {
		return startFourthRootPriceRatio, nil
	}

	// Calculate exponent = (currentTime - priceRatioUpdateStartTime) / (priceRatioUpdateEndTime - priceRatioUpdateStartTime)
	numerator, err := FixPoint.Sub(currentTime, priceRatioUpdateStartTime)
	if err != nil {
		return nil, err
	}

	denominator, err := FixPoint.Sub(priceRatioUpdateEndTime, priceRatioUpdateStartTime)
	if err != nil {
		return nil, err
	}

	exponent, err := FixPoint.DivDown(numerator, denominator)
	if err != nil {
		return nil, err
	}

	// Calculate ratio = endFourthRootPriceRatio / startFourthRootPriceRatio
	ratio, err := FixPoint.DivDown(endFourthRootPriceRatio, startFourthRootPriceRatio)
	if err != nil {
		return nil, err
	}

	// Calculate power = ratio^exponent
	power, err := FixPoint.PowUp(ratio, exponent)
	if err != nil {
		return nil, err
	}

	// Calculate currentFourthRootPriceRatio = startFourthRootPriceRatio * power
	currentFourthRootPriceRatio, err := FixPoint.MulDown(startFourthRootPriceRatio, power)
	if err != nil {
		return nil, err
	}

	// Since we're rounding current fourth root price ratio down, we only need to check the lower boundary.
	minimumFourthRootPriceRatio := startFourthRootPriceRatio
	if endFourthRootPriceRatio.Lt(startFourthRootPriceRatio) {
		minimumFourthRootPriceRatio = endFourthRootPriceRatio
	}

	if currentFourthRootPriceRatio.Lt(minimumFourthRootPriceRatio) {
		return minimumFourthRootPriceRatio, nil
	}

	return currentFourthRootPriceRatio, nil
}

// ComputeCenteredness computes the centeredness of the pool
func (r *reClammMath) ComputeCenteredness(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
) (*CenterednessResult, error) {
	if balancesScaled18[TokenA].IsZero() {
		// Also return false if both are 0 to be consistent with the logic below.
		return &CenterednessResult{
			PoolCenteredness:  U0,
			IsPoolAboveCenter: false,
		}, nil
	} else if balancesScaled18[TokenB].IsZero() {
		return &CenterednessResult{
			PoolCenteredness:  U0,
			IsPoolAboveCenter: true,
		}, nil
	}

	numerator, err := FixPoint.Mul(balancesScaled18[TokenA], virtualBalanceB)
	if err != nil {
		return nil, err
	}

	denominator, err := FixPoint.Mul(virtualBalanceA, balancesScaled18[TokenB])
	if err != nil {
		return nil, err
	}

	var poolCenteredness *uint256.Int
	var isPoolAboveCenter bool

	// The centeredness is defined between 0 and 1. If the numerator is greater than the denominator, we compute
	// the inverse ratio.
	if numerator.Lt(denominator) || numerator.Eq(denominator) {
		poolCenteredness, err = FixPoint.DivDown(numerator, denominator)
		if err != nil {
			return nil, err
		}
		isPoolAboveCenter = false
	} else {
		poolCenteredness, err = FixPoint.DivDown(denominator, numerator)
		if err != nil {
			return nil, err
		}
		isPoolAboveCenter = true
	}

	return &CenterednessResult{
		PoolCenteredness:  poolCenteredness,
		IsPoolAboveCenter: isPoolAboveCenter,
	}, nil
}

// ComputeInvariant computes the invariant of the pool
func (r *reClammMath) ComputeInvariant(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
	roundUp bool,
) (*uint256.Int, error) {
	// invariant = (Ra + Va) * (Rb + Vb)
	balanceAWithVirtual, err := FixPoint.Add(balancesScaled18[0], virtualBalanceA)
	if err != nil {
		return nil, err
	}

	balanceBWithVirtual, err := FixPoint.Add(balancesScaled18[1], virtualBalanceB)
	if err != nil {
		return nil, err
	}

	if roundUp {
		return FixPoint.MulUp(balanceAWithVirtual, balanceBWithVirtual)
	} else {
		return FixPoint.MulDown(balanceAWithVirtual, balanceBWithVirtual)
	}
}

// ComputeOutGivenIn computes the amount out given an exact amount in
func (r *reClammMath) ComputeOutGivenIn(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
	tokenInIndex int,
	tokenOutIndex int,
	amountInScaled18 *uint256.Int,
) (*uint256.Int, error) {
	var virtualBalanceTokenIn, virtualBalanceTokenOut *uint256.Int

	if tokenInIndex == 0 {
		virtualBalanceTokenIn = virtualBalanceA
		virtualBalanceTokenOut = virtualBalanceB
	} else {
		virtualBalanceTokenIn = virtualBalanceB
		virtualBalanceTokenOut = virtualBalanceA
	}

	// Ao = ((Bo + Vo) * Ai) / (Bi + Vi + Ai)
	numerator, err := FixPoint.Add(balancesScaled18[tokenOutIndex], virtualBalanceTokenOut)
	if err != nil {
		return nil, err
	}

	numerator, err = FixPoint.Mul(numerator, amountInScaled18)
	if err != nil {
		return nil, err
	}

	denominator, err := FixPoint.Add(balancesScaled18[tokenInIndex], virtualBalanceTokenIn)
	if err != nil {
		return nil, err
	}

	denominator, err = FixPoint.Add(denominator, amountInScaled18)
	if err != nil {
		return nil, err
	}

	amountOutScaled18, err := FixPoint.Div(numerator, denominator)
	if err != nil {
		return nil, err
	}

	if amountOutScaled18.Gt(balancesScaled18[tokenOutIndex]) {
		// Amount out cannot be greater than the real balance of the token in the pool.
		return nil, ErrReClammAmountOutGreaterThanBalance
	}

	return amountOutScaled18, nil
}

// ComputeInGivenOut computes the amount in given an exact amount out
func (r *reClammMath) ComputeInGivenOut(
	balancesScaled18 []*uint256.Int,
	virtualBalanceA *uint256.Int,
	virtualBalanceB *uint256.Int,
	tokenInIndex int,
	tokenOutIndex int,
	amountOutScaled18 *uint256.Int,
) (*uint256.Int, error) {
	if amountOutScaled18.Gt(balancesScaled18[tokenOutIndex]) {
		// Amount out cannot be greater than the real balance of the token in the pool.
		return nil, ErrReClammAmountOutGreaterThanBalance
	}

	var virtualBalanceTokenIn, virtualBalanceTokenOut *uint256.Int

	if tokenInIndex == 0 {
		virtualBalanceTokenIn = virtualBalanceA
		virtualBalanceTokenOut = virtualBalanceB
	} else {
		virtualBalanceTokenIn = virtualBalanceB
		virtualBalanceTokenOut = virtualBalanceA
	}

	// Ai = ((Bi + Vi) * Ao) / (Bo + Vo - Ao)
	a, err := FixPoint.Add(balancesScaled18[tokenInIndex], virtualBalanceTokenIn)
	if err != nil {
		return nil, err
	}

	c, err := FixPoint.Add(balancesScaled18[tokenOutIndex], virtualBalanceTokenOut)
	if err != nil {
		return nil, err
	}

	c, err = FixPoint.Sub(c, amountOutScaled18)
	if err != nil {
		return nil, err
	}

	// Round up to favor the vault (i.e. request larger amount in from the user).
	return FixPoint.MulDivUp(a, amountOutScaled18, c)
}

func (r *reClammMath) Sqrt(valueScaled18 *uint256.Int) (*uint256.Int, error) {
	// Multiply by WAD to get 36 decimals, then take square root to get 18 decimals
	return new(uint256.Int).Sqrt(valueScaled18), nil
}

// SqrtScaled18 calculates the square root of a value scaled by 18 decimals
// @param valueScaled18 The value to calculate the square root of, scaled by 18 decimals
// @return sqrtValueScaled18 The square root of the value scaled by 18 decimals
func (r *reClammMath) SqrtScaled18(valueScaled18 *uint256.Int) (*uint256.Int, error) {
	// Multiply by WAD to get 36 decimals, then take square root to get 18 decimals
	valueScaled36 := new(uint256.Int).Mul(valueScaled18, U1e18)
	return r.Sqrt(valueScaled36)
}
