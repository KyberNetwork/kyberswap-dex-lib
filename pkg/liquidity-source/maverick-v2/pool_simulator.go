package maverickv2

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	decimals []uint8
	state    *MaverickPoolState
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if len(extra.Bins) == 0 {
		return nil, ErrEmptyBins
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
				Reserves: []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]), bignumber.NewBig10(entityPool.Reserves[1])},
			},
		},
		decimals: []uint8{entityPool.Tokens[0].Decimals, entityPool.Tokens[1].Decimals},
		state: &MaverickPoolState{
			FeeAIn:           extra.FeeAIn,
			FeeBIn:           extra.FeeBIn,
			ProtocolFeeRatio: extra.ProtocolFeeRatio,
			Bins:             extra.Bins,
			BinPositions:     extra.BinPositions,
			BinMap:           extra.BinMap,
			TickSpacing:      staticExtra.TickSpacing,
			ActiveTick:       extra.ActiveTick,
			LastTwaD8:        extra.LastTwaD8,
			Timestamp:        extra.Timestamp,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	tokenAIn := strings.EqualFold(tokenAmountIn.Token, p.Pool.Info.Tokens[0])

	var scaleAmount *uint256.Int
	var err error
	// scale to AMM Amount 10^18
	if tokenAIn {
		scaleAmount, err = scaleFromAmount(amountIn, p.decimals[0])
	} else {
		scaleAmount, err = scaleFromAmount(amountIn, p.decimals[1])
	}
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	newState := p.state.Clone()
	_, amountOut, binCrossed, fractionalPart, err := swap(newState, scaleAmount, tokenAIn, false, false)
	if err != nil {
		return nil, fmt.Errorf("can not get amount out, err: %v", err)
	}

	// scale back to token amount
	var scaleAmountOut *uint256.Int
	if tokenAIn {
		scaleAmountOut, err = ScaleToAmount(amountOut, p.decimals[1])
	} else {
		scaleAmountOut, err = ScaleToAmount(amountOut, p.decimals[0])
	}
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	// Use fractional part directly from swap result (matches TypeScript implementation)
	var fractionalPartD8 int64
	if fractionalPart != nil && !fractionalPart.IsZero() {
		fractionalPartD8 = int64(fractionalPart.Uint64())
	} else {
		// Default to half-tick if not provided
		fractionalPartD8 = int64(BI_POWS[7].Uint64())
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: scaleAmountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		},
		Gas: GasSwap + GasCrossBin*int64(binCrossed),
		SwapInfo: maverickSwapInfo{
			activeTick:       newState.ActiveTick,
			bins:             newState.Bins,
			binPositions:     newState.BinPositions,
			fractionalPartD8: fractionalPartD8,
			timestamp:        getCurrentTimestamp(),
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.state = p.state.Clone()
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(maverickSwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalancer for Maverick pool, wrong swapInfo type")
		return
	}

	// Store old values for TWA and bin movements
	startingTick := p.state.ActiveTick
	lastTwaD8 := p.state.LastTwaD8

	// Update timestamp if provided, otherwise use current time
	timestamp := newState.timestamp
	if timestamp == 0 {
		timestamp = getCurrentTimestamp()
	}
	p.state.Timestamp = timestamp

	// Update the primary state values
	p.state.Bins = newState.bins
	p.state.BinPositions = newState.binPositions
	p.state.ActiveTick = newState.activeTick

	// Update time-weighted average
	// fractionalPartD8 := newState.fractionalPartD8
	// if fractionalPartD8 == 0 {
	// 	// Default to half the tick if not provided
	// 	fractionalPartD8 = int64(BI_POWS[7].Uint64())
	// }

	// Calculate full tick position with fractional part
	// tickPositionD8 := int64(p.state.ActiveTick)*int64(BI_POWS[8].Uint64()) + fractionalPartD8

	// Update TWA
	// no need to updatw time weighted average price, as UpdateBalance only call in same Route in one block
	// updateTwaValue(p.state, tickPositionD8, timestamp)

	// Move bins based on tick changes
	threshold := new(uint256.Int).Mul(new(uint256.Int).SetUint64(5), BI_POWS[7])
	moveBins(p.state, startingTick, p.state.ActiveTick, lastTwaD8, p.state.LastTwaD8, threshold)

	// Update pool reserves
	tokenAmountIn := params.TokenAmountIn
	tokenAmountOut := params.TokenAmountOut
	isTokenAIn := strings.EqualFold(tokenAmountIn.Token, p.Pool.Info.Tokens[0])

	// Calculate new internal balance (same as TypeScript's implementation)
	newInternalBalance := new(big.Int)
	if isTokenAIn {
		newInternalBalance = new(big.Int).Add(p.Pool.Info.Reserves[0], tokenAmountIn.Amount)
		p.Pool.Info.Reserves[0] = newInternalBalance
		p.Pool.Info.Reserves[1] = new(big.Int).Sub(p.Pool.Info.Reserves[1], tokenAmountOut.Amount)
	} else {
		newInternalBalance = new(big.Int).Add(p.Pool.Info.Reserves[1], tokenAmountIn.Amount)
		p.Pool.Info.Reserves[0] = new(big.Int).Sub(p.Pool.Info.Reserves[0], tokenAmountOut.Amount)
		p.Pool.Info.Reserves[1] = newInternalBalance
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

type maverickSwapInfo struct {
	activeTick       int32
	bins             map[uint32]Bin
	binPositions     map[int32][]uint32
	fractionalPartD8 int64
	timestamp        int64
}

// pastMaxTick checks if we've reached the tick limit and zeros out excess if so
// This is equivalent to MaverickDeltaMath.pastMaxTick() in TypeScript
func pastMaxTick(delta *Delta, activeTick, tickLimit int32) bool {
	swappedToMaxPrice := false
	if delta.TokenAIn {
		swappedToMaxPrice = tickLimit < activeTick
	} else {
		swappedToMaxPrice = tickLimit > activeTick
	}

	if swappedToMaxPrice {
		delta.Excess = new(uint256.Int) // CRITICAL: Zero out excess to terminate main loop
		delta.SkipCombine = true
		delta.SwappedToMaxPrice = true
	}

	return swappedToMaxPrice
}

// Helper functions for swap implementation
func swap(state *MaverickPoolState, amount *uint256.Int, tokenAIn bool, exactOutput bool, bypassLimit bool) (*uint256.Int, *uint256.Int, uint32, *uint256.Int, error) {
	// Implementation based on maverick-v2-pool-math.ts estimateSwap function

	delta := &Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int).Set(amount),
		TokenAIn:           tokenAIn,
		ExactOutput:        exactOutput,
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int),
		FractionalPart:     new(uint256.Int),
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
	}

	startingTick := state.ActiveTick

	// Set tickLimit based on tokenAIn (in JS this is computed in estimateSwap)
	var tickLimit int32
	if tokenAIn {
		tickLimit = startingTick + 100
	} else {
		tickLimit = startingTick - 100
	}

	// Handle main swap operation
	binCrossed := uint32(0)

	// Iteratively swap through ticks until the amount is consumed
	for !delta.Excess.IsZero() {
		newDelta, crossedBin, err := swapTick(state, delta, tickLimit)
		if err != nil {
			return nil, nil, 0, new(uint256.Int), err
		}

		if crossedBin {
			binCrossed++
		}

		combine(delta, newDelta)
	}

	return delta.DeltaInErc, delta.DeltaOutErc, binCrossed, delta.FractionalPart, nil
}

// swapTick
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/2108e064319bf14f98c321a8acd4762d3e9e3560/src/dex/maverick-v2/maverick-math/maverick-pool-math.ts#L621
func swapTick(state *MaverickPoolState, delta *Delta, tickLimit int32) (*Delta, bool, error) {
	newDelta := &Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int),
		TokenAIn:           delta.TokenAIn,
		ExactOutput:        delta.ExactOutput,
		TickLimit:          tickLimit,
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int),
		FractionalPart:     new(uint256.Int),
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
	}

	activeTick := state.ActiveTick

	// Check if we've reached the tick limit - equivalent to TypeScript pastMaxTick function
	if pastMaxTick(delta, activeTick, tickLimit) {
		state.ActiveTick += boolToInt32(!delta.TokenAIn) - boolToInt32(delta.TokenAIn)
		return delta, true, nil
	}

	// Find next tick with liquidity
	crossedBin := false
	var tickData Bin
	var ok bool

	for {
		// Get the tick data using a separate function - we'll reuse this in the final step
		tickData, ok = getTickData(state, activeTick)

		if ok && (tickData.ReserveA.BitLen() > 0 || tickData.ReserveB.BitLen() > 0) {
			break
		}

		// Move to next tick in correct direction
		activeTick += boolToInt32(delta.TokenAIn) - boolToInt32(!delta.TokenAIn)
		crossedBin = true

		// Check again if we've reached the tick limit after moving
		if pastMaxTick(delta, activeTick, tickLimit) {
			state.ActiveTick += boolToInt32(!delta.TokenAIn) - boolToInt32(delta.TokenAIn)
			return delta, true, nil
		}
	}

	state.ActiveTick = activeTick

	// Here's the key change: Calculate the sqrt prices using tickSqrtPriceAndLiquidity
	// This matches the TypeScript code: [delta.sqrtLowerTickPrice, delta.sqrtUpperTickPrice, delta.sqrtPrice, tickData] = this.tickSqrtPriceAndLiquidity(activeTick)
	newDelta.SqrtLowerTickPrice, newDelta.SqrtUpperTickPrice, newDelta.SqrtPrice, tickData = tickSqrtPriceAndLiquidity(state, activeTick)

	// Perform the actual swap computation
	if delta.ExactOutput {
		computeSwapExactOut(state, delta.Excess, delta.TokenAIn, tickData, newDelta)
	} else {
		computeSwapExactIn(state, delta.Excess, delta.TokenAIn, tickData, newDelta)
	}

	// Match TypeScript logic exactly
	if newDelta.Excess.IsZero() {
		computeEndPrice(delta, newDelta, tickData)
	}

	// don't need allocateSwapValuesToTick (but don't mutate state in simulation)
	// allocateSwapValuesToTick(newDelta, delta.TokenAIn, activeTick, tickData)

	// If there's excess remaining, we need to move to the next tick
	if !newDelta.Excess.IsZero() {
		nextTick := activeTick + boolToInt32(delta.TokenAIn) - boolToInt32(!delta.TokenAIn)
		state.ActiveTick = nextTick
		crossedBin = true
	}

	return newDelta, crossedBin, nil
}

// allocateSwapValuesToTick simulates the tick reserve updates (for calculation purposes only)
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/2108e064319bf14f98c321a8acd4762d3e9e3560/src/dex/maverick-v2/maverick-math/maverick-pool-math.ts#L692
func allocateSwapValuesToTick(delta *Delta, tokenAIn bool, tick int32, tickData Bin) {
	// In a pool simulator, we don't actually mutate state, but we simulate the logic
	// to ensure calculations are consistent with real swaps

	// This matches the TypeScript logic but doesn't persist changes:
	// let reserveA = tickState.reserveA;
	// let reserveB = tickState.reserveB;

	if tokenAIn {
		// reserveA = reserveA + delta.deltaInBinInternal;
		// reserveB = delta.excess > 0n ? 0n : MaverickBasicMath.clip(reserveB, delta.deltaOutErc);
		if !delta.Excess.IsZero() {
			// If there's excess, this tick is fully consumed (reserveB becomes 0)
			// No actual state mutation needed in simulator
		} else {
			// Normal case: reduce reserveB by deltaOutErc
			// tickState.reserveB = clip(tickState.reserveB, delta.deltaOutErc)
			// No actual state mutation needed in simulator
		}
	} else {
		// reserveA = delta.excess > 0n ? 0n : MaverickBasicMath.clip(reserveA, delta.deltaOutErc);
		// reserveB = reserveB + delta.deltaInBinInternal;
		if !delta.Excess.IsZero() {
			// If there's excess, this tick is fully consumed (reserveA becomes 0)
			// No actual state mutation needed in simulator
		} else {
			// Normal case: reduce reserveA by deltaOutErc
			// tickState.reserveA = clip(tickState.reserveA, delta.deltaOutErc)
			// No actual state mutation needed in simulator
		}
	}

	// Note: In the real implementation (UpdateBalance), this would update the actual bins
	// But in simulation, we only need to ensure the calculation logic is consistent
}

// computeEndPrice calculates the end price and fractional part when there's no excess remaining
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-swap-math.ts#L178
func computeEndPrice(delta *Delta, newDelta *Delta, tickData Bin) {
	// Calculate endSqrtPrice following TypeScript logic exactly
	// endSqrtPrice = divDown(newDelta.deltaInBinInternal, tickData.currentLiquidity) +
	//                (delta.tokenAIn ? delta.sqrtPrice : invFloor(delta.sqrtPrice))

	var endSqrtPrice *uint256.Int
	if tickData.CurrentLiquidity.IsZero() {
		endSqrtPrice = new(uint256.Int)
	} else {
		endSqrtPrice = mulDivDown(newDelta.DeltaInBinInternal, BI_POWS[18], tickData.CurrentLiquidity)
	}

	if delta.TokenAIn {
		endSqrtPrice.Add(endSqrtPrice, delta.SqrtPrice)
	} else {
		endSqrtPrice.Add(endSqrtPrice, invFloor(delta.SqrtPrice))
	}

	// If not tokenAIn, apply invFloor to the result
	if !delta.TokenAIn {
		endSqrtPrice = invFloor(endSqrtPrice)
	}

	// Calculate fractional part
	// newDelta.fractionalPart = min(BI_POWS[8], divDown(clip(endSqrtPrice, delta.sqrtLowerTickPrice), BI_POWS[10] * (delta.sqrtUpperTickPrice - delta.sqrtLowerTickPrice)))
	clippedPrice := clip(endSqrtPrice, delta.SqrtLowerTickPrice)
	denominator := new(uint256.Int).Sub(delta.SqrtUpperTickPrice, delta.SqrtLowerTickPrice)
	denominator.Mul(denominator, BI_POWS[10])

	if !denominator.IsZero() {
		fractionalPart := mulDivDown(clippedPrice, BI_POWS[18], denominator)
		newDelta.FractionalPart = minUint256(BI_POWS[8], fractionalPart)
	} else {
		newDelta.FractionalPart = new(uint256.Int)
	}
}

func computeSwapExactIn(state *MaverickPoolState, amountIn *uint256.Int, tokenAIn bool, tickData Bin, delta *Delta) {

	// Initialize delta with proper values - matching TypeScript exactly
	delta.TokenAIn = tokenAIn
	delta.ExactOutput = false
	delta.SwappedToMaxPrice = false
	delta.SkipCombine = false

	// Set initial deltaOutErc to all available reserves - line 68-70 in TypeScript
	if tokenAIn {
		delta.DeltaOutErc = new(uint256.Int).Set(tickData.ReserveB) // currentReserveB in TypeScript
	} else {
		delta.DeltaOutErc = new(uint256.Int).Set(tickData.ReserveA) // currentReserveA in TypeScript
	}

	// Calculate remaining bin input space given the output - lines 72-77 in TypeScript
	binAmountIn := remainingBinInputSpaceGivenOutput(
		tickData.CurrentLiquidity, // currentLiquidity in TypeScript
		delta.DeltaOutErc,
		delta.SqrtPrice,
		tokenAIn,
	)

	// Get fee basis
	var fee *uint256.Int
	if tokenAIn {
		fee = new(uint256.Int).SetUint64(uint64(state.FeeAIn))
	} else {
		fee = new(uint256.Int).SetUint64(uint64(state.FeeBIn))
	}

	// Calculate user bin amount in - lines 80-83 in TypeScript
	userBinAmountIn := mulDown(amountIn, new(uint256.Int).Sub(BI_POWS[18], fee))

	var feeBasis *uint256.Int

	// Logic for determining actual binAmountIn and fees - lines 85-97 in TypeScript
	if userBinAmountIn.Cmp(binAmountIn) < 0 {
		binAmountIn = new(uint256.Int).Set(userBinAmountIn)
		delta.DeltaInErc = new(uint256.Int).Set(amountIn)
		feeBasis = new(uint256.Int).Sub(delta.DeltaInErc, userBinAmountIn)
	} else {
		feeBasis = mulDivUp(binAmountIn, fee, new(uint256.Int).Sub(BI_POWS[18], fee))
		delta.DeltaInErc = new(uint256.Int).Add(binAmountIn, feeBasis)
		delta.Excess = clip(amountIn, delta.DeltaInErc)
	}

	// Calculate amount to bin net of protocol fee - lines 99-103 in TypeScript
	delta.DeltaInBinInternal = amountToBinNetOfProtocolFee(
		delta.DeltaInErc,
		feeBasis,
		state.ProtocolFeeRatio,
	)

	// Early return if excess exists - line 105 in TypeScript
	if !delta.Excess.IsZero() {
		return
	}

	// Calculate inOverL - lines 107-110 in TypeScript
	inOverL := divUp(binAmountIn, new(uint256.Int).Add(tickData.CurrentLiquidity, new(uint256.Int).SetUint64(1)))

	// Calculate final deltaOutErc - lines 112-119 in TypeScript
	var calculatedOut *uint256.Int
	if tokenAIn {
		// delta.deltaOutErc = MaverickBasicMath.min(delta.deltaOutErc, MaverickBasicMath.mulDivDown(
		//   binAmountIn, MaverickBasicMath.invFloor(sqrtPrice), inOverL + sqrtPrice))
		denominator := new(uint256.Int).Add(inOverL, delta.SqrtPrice)
		calculatedOut = mulDivDown(binAmountIn, invFloor(delta.SqrtPrice), denominator)
	} else {
		// delta.deltaOutErc = MaverickBasicMath.min(delta.deltaOutErc, MaverickBasicMath.mulDivDown(
		//   binAmountIn, sqrtPrice, inOverL + MaverickBasicMath.invCeil(sqrtPrice)))
		denominator := new(uint256.Int).Add(inOverL, invCeil(delta.SqrtPrice))
		calculatedOut = mulDivDown(binAmountIn, delta.SqrtPrice, denominator)
	}

	delta.DeltaOutErc = minUint256(delta.DeltaOutErc, calculatedOut)
}

func computeSwapExactOut(state *MaverickPoolState, amountOut *uint256.Int, tokenAIn bool, tickData Bin, delta *Delta) {
	// Initialize delta with proper values - matching TypeScript exactly
	delta.TokenAIn = tokenAIn
	delta.ExactOutput = true
	delta.SwappedToMaxPrice = false
	delta.SkipCombine = false

	// Determine available output amount - lines 148-150 in TypeScript
	var amountOutAvailable *uint256.Int
	if tokenAIn {
		amountOutAvailable = new(uint256.Int).Set(tickData.ReserveB) // currentReserveB in TypeScript
	} else {
		amountOutAvailable = new(uint256.Int).Set(tickData.ReserveA) // currentReserveA in TypeScript
	}

	// Check if we have enough liquidity - lines 151-152 in TypeScript
	swapped := amountOutAvailable.Cmp(amountOut) <= 0
	if swapped {
		delta.DeltaOutErc = new(uint256.Int).Set(amountOutAvailable)
	} else {
		delta.DeltaOutErc = new(uint256.Int).Set(amountOut)
	}

	// Calculate required input using remainingBinInputSpaceGivenOutput - lines 153-158 in TypeScript
	binAmountIn := remainingBinInputSpaceGivenOutput(
		tickData.CurrentLiquidity, // currentLiquidity in TypeScript
		delta.DeltaOutErc,
		delta.SqrtPrice,
		tokenAIn,
	)

	// Calculate fee - lines 160-164 in TypeScript
	var fee uint64
	if tokenAIn {
		fee = state.FeeAIn
	} else {
		fee = state.FeeBIn
	}

	// feeBasis = MaverickBasicMath.mulDivUp(binAmountIn, fee, BI_POWS[18] - fee)
	feeBasis := mulDivUp(
		binAmountIn,
		new(uint256.Int).SetUint64(uint64(fee)),
		new(uint256.Int).Sub(BI_POWS[18], new(uint256.Int).SetUint64(uint64(fee))),
	)

	// delta.deltaInErc = binAmountIn + feeBasis - line 165 in TypeScript
	delta.DeltaInErc = new(uint256.Int).Add(binAmountIn, feeBasis)

	// delta.deltaInBinInternal = this.amountToBinNetOfProtocolFee(...) - lines 166-170 in TypeScript
	delta.DeltaInBinInternal = amountToBinNetOfProtocolFee(
		delta.DeltaInErc,
		feeBasis,
		state.ProtocolFeeRatio,
	)

	// delta.excess = swapped ? MaverickBasicMath.clip(amountOut, delta.deltaOutErc) : 0n - lines 171-173 in TypeScript
	if swapped {
		delta.Excess = clip(amountOut, delta.DeltaOutErc)
	} else {
		delta.Excess = new(uint256.Int)
	}
}

func getTickData(state *MaverickPoolState, tick int32) (Bin, bool) {
	bins, ok := state.BinPositions[tick]
	if !ok || len(bins) == 0 {
		return Bin{
			ReserveA:         new(uint256.Int),
			ReserveB:         new(uint256.Int),
			CurrentLiquidity: new(uint256.Int), // Initialize to 0
		}, false
	}

	// Consolidate reserves for all bins at this tick
	consolidatedReserveA := new(uint256.Int)
	consolidatedReserveB := new(uint256.Int)

	for _, binId := range bins {
		bin, ok := state.Bins[binId]
		if ok {
			consolidatedReserveA = new(uint256.Int).Add(consolidatedReserveA, bin.ReserveA)
			consolidatedReserveB = new(uint256.Int).Add(consolidatedReserveB, bin.ReserveB)
		}
	}

	return Bin{
		ReserveA:         consolidatedReserveA,
		ReserveB:         consolidatedReserveB,
		CurrentLiquidity: new(uint256.Int), // Will be calculated and set by tickSqrtPriceAndLiquidity
	}, true
}

func combine(self *Delta, delta *Delta) {
	if !self.SkipCombine {
		self.DeltaInBinInternal = new(uint256.Int).Add(self.DeltaInBinInternal, delta.DeltaInBinInternal)
		self.DeltaInErc = new(uint256.Int).Add(self.DeltaInErc, delta.DeltaInErc)
		self.DeltaOutErc = new(uint256.Int).Add(self.DeltaOutErc, delta.DeltaOutErc)
	}

	// Always update these fields regardless of SkipCombine
	self.Excess = new(uint256.Int).Set(delta.Excess)
	self.SwappedToMaxPrice = delta.SwappedToMaxPrice

	// Set the sqrt prices and fractional part from the latest delta
	if delta.SqrtLowerTickPrice != nil && !delta.SqrtLowerTickPrice.IsZero() {
		self.SqrtLowerTickPrice = new(uint256.Int).Set(delta.SqrtLowerTickPrice)
	}
	if delta.SqrtUpperTickPrice != nil && !delta.SqrtUpperTickPrice.IsZero() {
		self.SqrtUpperTickPrice = new(uint256.Int).Set(delta.SqrtUpperTickPrice)
	}
	if delta.SqrtPrice != nil && !delta.SqrtPrice.IsZero() {
		self.SqrtPrice = new(uint256.Int).Set(delta.SqrtPrice)
	}
	if delta.FractionalPart != nil && !delta.FractionalPart.IsZero() {
		self.FractionalPart = new(uint256.Int).Set(delta.FractionalPart)
	}
}

func scaleFromAmount(amount *uint256.Int, decimals uint8) (*uint256.Int, error) {
	scale := getScale(decimals)
	return mulDiv(amount, BI_POWS[18], scale), nil
}

func ScaleToAmount(amount *uint256.Int, decimals uint8) (*uint256.Int, error) {
	scale := getScale(decimals)
	return mulDiv(amount, scale, BI_POWS[18]), nil
}

func getScale(decimals uint8) *uint256.Int {
	return new(uint256.Int).Exp(new(uint256.Int).SetUint64(10), new(uint256.Int).SetUint64(uint64(decimals)))
}

func (state *MaverickPoolState) Clone() *MaverickPoolState {
	cloned := &MaverickPoolState{
		FeeAIn:           state.FeeAIn,
		FeeBIn:           state.FeeBIn,
		ProtocolFeeRatio: state.ProtocolFeeRatio,
		TickSpacing:      state.TickSpacing,
		ActiveTick:       state.ActiveTick,
		Bins:             make(map[uint32]Bin, len(state.Bins)),
		BinPositions:     make(map[int32][]uint32, len(state.BinPositions)),
		BinMap:           make(map[int32]uint32, len(state.BinMap)),
		LastTwaD8:        state.LastTwaD8,
		Timestamp:        state.Timestamp,
		BinCounter:       state.BinCounter,
	}

	for k, v := range state.Bins {
		clonedBin := Bin{
			ReserveA:        safeCloneUint256(v.ReserveA),
			ReserveB:        safeCloneUint256(v.ReserveB),
			MergeBinBalance: safeCloneUint256(v.MergeBinBalance),
			TotalSupply:     safeCloneUint256(v.TotalSupply),
			TickBalance:     safeCloneUint256(v.TickBalance),
			MergeId:         v.MergeId,
			Kind:            v.Kind,
			Tick:            v.Tick,
		}
		cloned.Bins[k] = clonedBin
	}

	for k, v := range state.BinPositions {
		cloned.BinPositions[k] = make([]uint32, len(v))
		copy(cloned.BinPositions[k], v)
	}

	for k, v := range state.BinMap {
		cloned.BinMap[k] = v
	}

	return cloned
}

// Helper math functions
func mulDiv(a, b, denominator *uint256.Int) *uint256.Int {
	product := new(uint256.Int).Mul(a, b)
	return new(uint256.Int).Div(product, denominator)
}

// Helper function to get token index (0 for tokenA, 1 for tokenB)
func getTokenIndex(isTokenA bool) int {
	if isTokenA {
		return 0
	}
	return 1
}

func divRoundingUp(a, b *uint256.Int) *uint256.Int {
	numerator := new(uint256.Int).Add(a, new(uint256.Int).Sub(b, new(uint256.Int).SetUint64(1)))
	return new(uint256.Int).Div(numerator, b)
}

func divRoundingDown(a, b *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(a, b)
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// Types and constants section
type MaverickPoolState struct {
	FeeAIn           uint64
	FeeBIn           uint64
	ProtocolFeeRatio uint8
	Bins             map[uint32]Bin
	BinPositions     map[int32][]uint32
	BinMap           map[int32]uint32
	TickSpacing      uint32
	ActiveTick       int32
	LastTwaD8        int64  // Time-weighted average tick data
	Timestamp        int64  // Current timestamp
	BinCounter       uint32 // Counter for bin IDs
}

type Extra struct {
	FeeAIn           uint64             `json:"feeAIn"`
	FeeBIn           uint64             `json:"feeBIn"`
	ProtocolFeeRatio uint8              `json:"protocolFeeRatio"`
	Bins             map[uint32]Bin     `json:"bins"`
	BinPositions     map[int32][]uint32 `json:"binPositions"`
	BinMap           map[int32]uint32   `json:"binMap"`
	ActiveTick       int32              `json:"activeTick"`
	LastTwaD8        int64              `json:"lastTwaD8"`
	Timestamp        int64              `json:"timestamp"`
	AccumValueD8     string             `json:"accumValueD8"`
}

type Bin struct {
	MergeBinBalance  *uint256.Int `json:"mergeBinBalance"`
	MergeId          uint32       `json:"mergeId"`
	TotalSupply      *uint256.Int `json:"totalSupply"`
	Kind             uint8        `json:"kind"`
	Tick             int32        `json:"tick"`
	TickBalance      *uint256.Int `json:"tickBalance"`
	ReserveA         *uint256.Int `json:"reserveA"`
	ReserveB         *uint256.Int `json:"reserveB"`
	CurrentLiquidity *uint256.Int `json:"currentLiquidity,omitempty"` // Added for TypeScript compatibility
}

type Delta struct {
	DeltaInBinInternal *uint256.Int
	DeltaInErc         *uint256.Int
	DeltaOutErc        *uint256.Int
	Excess             *uint256.Int
	TokenAIn           bool
	ExactOutput        bool
	TickLimit          int32
	SqrtLowerTickPrice *uint256.Int
	SqrtUpperTickPrice *uint256.Int
	SqrtPrice          *uint256.Int
	FractionalPart     *uint256.Int
	SwappedToMaxPrice  bool
	SkipCombine        bool
}

// TickState represents a tick's liquidity state
type TickState struct {
	ReserveA     *uint256.Int
	ReserveB     *uint256.Int
	TotalSupply  *uint256.Int
	BinIdsByTick map[uint8]uint32
}

func moveBins(state *MaverickPoolState, startingTick, activeTick int32, lastTwapD8, newTwapD8 int64, threshold *uint256.Int) {
	// Skip if no tick change
	if startingTick == activeTick {
		return
	}

	// Implementation matching TypeScript moveBins function
	// First handle upward movement
	newTwap := floorD8Unchecked(newTwapD8 - int64(threshold.Uint64()))
	lastTwap := floorD8Unchecked(lastTwapD8 - int64(threshold.Uint64()))

	if activeTick > startingTick || newTwap > lastTwap {
		// Create moveData equivalent to MoveData in TypeScript
		moveData := &MoveData{
			Kind:            0,
			TickSearchStart: 0,
			TickSearchEnd:   0,
			TickLimit:       0,
			FirstBinTick:    0,
			FirstBinId:      0,
			MergeBinBalance: new(uint256.Int),
			TotalReserveA:   new(uint256.Int),
			TotalReserveB:   new(uint256.Int),
			MergeBins:       make(map[uint32]bool),
			Counter:         0,
		}

		// Calculate tickLimit as min(activeTick - 1, newTwap)
		moveData.TickLimit = activeTick - 1
		if int32(newTwap) < moveData.TickLimit {
			moveData.TickLimit = int32(newTwap)
		}

		if int32(lastTwap)-1 < moveData.TickLimit {
			moveData.TickSearchStart = int32(lastTwap) - 1
			moveData.TickSearchEnd = moveData.TickLimit
			moveData.Kind = 1 // Kind 1 = moving up
			moveDirection(state, moveData)
			moveData.Kind = 3 // Kind 3 = special case
			moveDirection(state, moveData)

			// We'll never move in both directions in one swap
			return
		}
	}

	// Handle downward movement
	newTwap = floorD8Unchecked(newTwapD8 + int64(threshold.Uint64()))
	lastTwap = floorD8Unchecked(lastTwapD8 + int64(threshold.Uint64()))

	if activeTick < startingTick || newTwap < lastTwap {
		// Create moveData equivalent to MoveData in TypeScript
		moveData := &MoveData{
			Kind:            0,
			TickSearchStart: 0,
			TickSearchEnd:   0,
			TickLimit:       0,
			FirstBinTick:    0,
			FirstBinId:      0,
			MergeBinBalance: new(uint256.Int),
			TotalReserveA:   new(uint256.Int),
			TotalReserveB:   new(uint256.Int),
			MergeBins:       make(map[uint32]bool),
			Counter:         0,
		}

		// Calculate tickLimit as max(newTwap, activeTick + 1)
		moveData.TickLimit = activeTick + 1
		if int32(newTwap) > moveData.TickLimit {
			moveData.TickLimit = int32(newTwap)
		}

		if moveData.TickLimit < int32(lastTwap)+1 {
			moveData.TickSearchStart = moveData.TickLimit
			moveData.TickSearchEnd = int32(lastTwap) + 1
			moveData.Kind = 2 // Kind 2 = moving down
			moveDirection(state, moveData)
			moveData.Kind = 3 // Kind 3 = special case
			moveDirection(state, moveData)
		}
	}
}

// Helper function for TWA calculations
func floorD8Unchecked(value int64) int64 {
	return value / 256
}

// Implementation of moveDirection from TypeScript
func moveDirection(state *MaverickPoolState, moveData *MoveData) {
	// Reset values
	moveData.FirstBinTick = 0
	moveData.FirstBinId = 0
	moveData.MergeBinBalance = new(uint256.Int)
	moveData.TotalReserveA = new(uint256.Int)
	moveData.TotalReserveB = new(uint256.Int)
	moveData.Counter = 0

	// Find movement bins in the range
	getMovementBinsInRange(state, moveData)

	// Skip if no bins found or only one bin at the limit
	if moveData.FirstBinId == 0 || (moveData.Counter == 1 && moveData.TickLimit == moveData.FirstBinTick) {
		return
	}

	// Get the first bin and its tick state - exactly like TypeScript
	firstBin, ok := state.Bins[moveData.FirstBinId]
	if !ok {
		return
	}

	// Get first bin tick state - equivalent to this.state.ticks[moveData.firstBinTick.toString()]
	firstBinTickState := getTickState(state, moveData.FirstBinTick)

	// Merge bins in the list - this modifies firstBinTickState
	mergeBinsInList(state, &firstBin, firstBinTickState, moveData)

	// Move bin to new tick if needed
	if moveData.TickLimit != moveData.FirstBinTick {
		// Get ending tick state - equivalent to this.state.ticks[moveData.TickLimit.toString()]
		endingTickState := getTickState(state, moveData.TickLimit)
		// Pass the same firstBinTickState that was modified by mergeBinsInList
		moveBinToNewTick(state, &firstBin, firstBinTickState, endingTickState, moveData)
	}
}

// Implementation of getMovementBinsInRange from TypeScript
func getMovementBinsInRange(state *MaverickPoolState, moveData *MoveData) {
	for tick := moveData.TickSearchStart; tick <= moveData.TickSearchEnd; tick++ {
		if moveData.Counter == 3 {
			return
		}

		// Get bin ID by tick and kind
		binId := binIdByTickKind(state, tick, moveData.Kind)
		if binId == 0 {
			continue
		}

		// Record this bin
		moveData.MergeBins[binId] = true
		moveData.Counter++

		// Update first bin info if needed
		if moveData.FirstBinId == 0 || binId < moveData.FirstBinId {
			moveData.FirstBinId = binId
			moveData.FirstBinTick = tick
		}
	}
}

// Helper function to get bin ID by tick and kind
func binIdByTickKind(state *MaverickPoolState, tick int32, kind uint8) uint32 {
	// Get bin positions at this tick
	binPositions, ok := state.BinPositions[tick]
	if !ok || len(binPositions) == 0 {
		return 0
	}

	// Find bin with matching kind
	for _, binId := range binPositions {
		bin, ok := state.Bins[binId]
		if ok && bin.Kind == kind {
			return binId
		}
	}

	return 0
}

// Implementation of mergeBinsInList from TypeScript
func mergeBinsInList(state *MaverickPoolState, firstBin *Bin, firstBinTickState *TickState, moveData *MoveData) {
	mergeOccured := false

	// Iterate through all the merge bins
	for binId := range moveData.MergeBins {
		if binId == moveData.FirstBinId {
			continue
		}

		// Merge this bin
		mergeOccured = true

		// Get bin info
		bin, ok := state.Bins[binId]
		if !ok {
			continue
		}

		// Get tick info
		tickData, ok := getTickData(state, bin.Tick)
		if !ok {
			continue
		}

		// Calculate bin reserves
		binA := new(uint256.Int).Div(new(uint256.Int).Mul(bin.TickBalance, tickData.ReserveA), new(uint256.Int).Add(tickData.TotalSupply, new(uint256.Int).SetUint64(1)))
		binB := new(uint256.Int).Div(new(uint256.Int).Mul(bin.TickBalance, tickData.ReserveB), new(uint256.Int).Add(tickData.TotalSupply, new(uint256.Int).SetUint64(1)))

		// Mark bin as merged
		bin.MergeId = moveData.FirstBinId

		// Calculate merge bin balance - simplified for now
		mergeBinBalance := calculateMergeBinBalance(*firstBin, binA, binB)
		bin.MergeBinBalance = mergeBinBalance

		// Update tick info
		tickState, ok := state.BinPositions[bin.Tick]
		if ok && len(tickState) > 0 {
			// Remove this bin from its current tick
			newTickState := make([]uint32, 0, len(tickState))
			for _, id := range tickState {
				if id != binId {
					newTickState = append(newTickState, id)
				}
			}
			state.BinPositions[bin.Tick] = newTickState
		}

		// Update total reserves
		moveData.TotalReserveA = new(uint256.Int).Add(moveData.TotalReserveA, binA)
		moveData.TotalReserveB = new(uint256.Int).Add(moveData.TotalReserveB, binB)
		moveData.MergeBinBalance = new(uint256.Int).Add(moveData.MergeBinBalance, mergeBinBalance)

		// Update the bin in state
		state.Bins[binId] = bin
	}

	// Add the merged liquidity to the first bin if any merges happened
	if mergeOccured {
		// Add liquidity to the first bin - equivalent to MaverickBinMath.addLiquidityByReserves
		addLiquidityByReserves(state, *firstBin, firstBinTickState, moveData.TotalReserveA, moveData.TotalReserveB, moveData.MergeBinBalance)

		// Update the bin in state
		state.Bins[moveData.FirstBinId] = *firstBin
	}
}

// Helper function to calculate merge bin balance
func calculateMergeBinBalance(parentBin Bin, binA, binB *uint256.Int) *uint256.Int {
	// This is a simplified calculation for now
	return new(uint256.Int).Add(binA, binB)
}

// Helper function to add liquidity by reserves
func addLiquidityByReserves(state *MaverickPoolState, bin Bin, tickState *TickState, reserveA, reserveB, mergeBinBalance *uint256.Int) {
	// Update the bin's reserves
	bin.ReserveA = new(uint256.Int).Add(bin.ReserveA, reserveA)
	bin.ReserveB = new(uint256.Int).Add(bin.ReserveB, reserveB)

	// Update the bin's merge bin balance
	bin.MergeBinBalance = new(uint256.Int).Add(bin.MergeBinBalance, mergeBinBalance)

	// Update the tick state reserves as well
	tickState.ReserveA = new(uint256.Int).Add(tickState.ReserveA, reserveA)
	tickState.ReserveB = new(uint256.Int).Add(tickState.ReserveB, reserveB)
}

// Implementation of moveBinToNewTick from TypeScript - exact mapping
func moveBinToNewTick(state *MaverickPoolState, firstBin *Bin, startingTickState *TickState, endingTickState *TickState, moveData *MoveData) {
	// Step 1: Get bin reserves using binReserves equivalent to MaverickPoolLib.binReserves
	// Convert TickState back to Bin format for binReserves function
	startingTickData := Bin{
		ReserveA:    startingTickState.ReserveA,
		ReserveB:    startingTickState.ReserveB,
		TotalSupply: startingTickState.TotalSupply,
	}
	firstBinA, firstBinB := binReserves(*firstBin, startingTickData)

	// Step 2: Update starting tick state using clip (equivalent to MaverickBasicMath.clip)
	startingTickState.ReserveA = clip(startingTickState.ReserveA, firstBinA)
	startingTickState.ReserveB = clip(startingTickState.ReserveB, firstBinB)
	startingTickState.TotalSupply = clip(startingTickState.TotalSupply, firstBin.TickBalance)
	startingTickState.BinIdsByTick[moveData.Kind] = 0

	// Step 3: Delete tick if totalSupply is zero (exact TypeScript logic)
	if startingTickState.TotalSupply.IsZero() {
		delete(state.BinPositions, firstBin.Tick)
		// Remove all bins from this tick
		if binPositions, ok := state.BinPositions[firstBin.Tick]; ok {
			for _, binId := range binPositions {
				if bin, exists := state.Bins[binId]; exists {
					// Don't delete the bin, just remove it from positions
					_ = bin
				}
			}
			delete(state.BinPositions, firstBin.Tick)
		}
	} else {
		// Update the starting tick state back to pool state
		updateTickState(state, moveData.FirstBinTick, startingTickState)
	}

	// Step 4: Update ending tick state
	endingTickState.BinIdsByTick[moveData.Kind] = moveData.FirstBinId
	firstBin.Tick = moveData.TickLimit

	// Step 5: Calculate deltaTickBalance using exact TypeScript logic
	var deltaTickBalance *uint256.Int
	if firstBinA.Cmp(firstBinB) > 0 {
		// Using mulDivDown equivalent to MaverickBasicMath.mulDivDown
		deltaTickBalance = mulDivDown(
			firstBinA,
			max(new(uint256.Int).SetUint64(1), endingTickState.TotalSupply),
			max(new(uint256.Int).SetUint64(1), endingTickState.ReserveA),
		)
	} else {
		deltaTickBalance = mulDivDown(
			firstBinB,
			max(new(uint256.Int).SetUint64(1), endingTickState.TotalSupply),
			max(new(uint256.Int).SetUint64(1), endingTickState.ReserveB),
		)
	}

	// Step 6: Update ending tick state reserves and total supply
	endingTickState.ReserveA = new(uint256.Int).Add(endingTickState.ReserveA, firstBinA)
	endingTickState.ReserveB = new(uint256.Int).Add(endingTickState.ReserveB, firstBinB)
	firstBin.TickBalance = deltaTickBalance
	endingTickState.TotalSupply = new(uint256.Int).Add(endingTickState.TotalSupply, deltaTickBalance)

	// Step 7: Update ending tick state back to pool state
	updateTickState(state, moveData.TickLimit, endingTickState)

	// Step 8: Update the bin in state
	state.Bins[moveData.FirstBinId] = *firstBin
}

// Helper function to remove bin from tick
func removeBinFromTick(state *MaverickPoolState, tick int32, binId uint32, kind uint8) {
	// Get bins at this tick
	tickState, ok := state.BinPositions[tick]
	if !ok || len(tickState) == 0 {
		return
	}

	// Remove this bin
	newTickState := make([]uint32, 0, len(tickState))
	for _, id := range tickState {
		if id != binId {
			newTickState = append(newTickState, id)
		}
	}

	// Update tick state
	state.BinPositions[tick] = newTickState
}

// Helper function to add bin to tick
func addBinToTick(state *MaverickPoolState, tick int32, binId uint32, kind uint8) {
	// Get bins at this tick
	tickState, ok := state.BinPositions[tick]
	if !ok {
		// Create new tick state if it doesn't exist
		state.BinPositions[tick] = []uint32{binId}
		return
	}

	// Add bin to tick
	state.BinPositions[tick] = append(tickState, binId)
}

// Helper to get max of two uint256.Int
func max(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) > 0 {
		return new(uint256.Int).Set(a)
	}
	return new(uint256.Int).Set(b)
}

// Helper function equivalent to MaverickBasicMath.clip - safe subtraction
func clip(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) >= 0 {
		return new(uint256.Int).Sub(a, b)
	}
	return new(uint256.Int)
}

// Helper function equivalent to MaverickBasicMath.mulDivDown
func mulDivDown(a, b, denominator *uint256.Int) *uint256.Int {
	if denominator.IsZero() {
		return new(uint256.Int)
	}
	product := new(uint256.Int).Mul(a, b)
	return new(uint256.Int).Div(product, denominator)
}

// Helper function equivalent to MaverickBasicMath.mulDivUp
func mulDivUp(a, b, denominator *uint256.Int) *uint256.Int {
	if denominator.IsZero() {
		return new(uint256.Int)
	}
	product := new(uint256.Int).Mul(a, b)
	result := new(uint256.Int).Div(product, denominator)
	// Add 1 if there's a remainder (for ceiling division)
	remainder := new(uint256.Int).Mod(product, denominator)
	if !remainder.IsZero() {
		result.Add(result, new(uint256.Int).SetUint64(1))
	}
	return result
}

// Helper function equivalent to MaverickBasicMath.divUp
func divUp(a, b *uint256.Int) *uint256.Int {
	if b.IsZero() {
		return new(uint256.Int)
	}
	result := new(uint256.Int).Div(a, b)
	remainder := new(uint256.Int).Mod(a, b)
	if !remainder.IsZero() {
		result.Add(result, new(uint256.Int).SetUint64(1))
	}
	return result
}

// Helper function equivalent to MaverickBasicMath.invFloor
func invFloor(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return new(uint256.Int)
	}
	// invFloor(x) = BI_POWS[36] / x (with floor division)
	return new(uint256.Int).Div(BI_POWS[36], x)
}

// Helper function equivalent to MaverickBasicMath.invCeil
func invCeil(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return new(uint256.Int)
	}
	// invCeil(x) = (BI_POWS[36] + x - 1) / x (ceiling division)
	numerator := new(uint256.Int).Add(BI_POWS[36], x)
	numerator.Sub(numerator, new(uint256.Int).SetUint64(1))
	return new(uint256.Int).Div(numerator, x)
}

// Helper function equivalent to MaverickBasicMath.mulDown
func mulDown(a, b *uint256.Int) *uint256.Int {
	return mulDivDown(a, b, BI_POWS[18])
}

// Helper function equivalent to MaverickBasicMath.min
func minUint256(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) < 0 {
		return new(uint256.Int).Set(a)
	}
	return new(uint256.Int).Set(b)
}

// remainingBinInputSpaceGivenOutput calculates remaining input space, matching TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-swap-math.ts#L21
func remainingBinInputSpaceGivenOutput(binLiquidity, output, sqrtPrice *uint256.Int, tokenAIn bool) *uint256.Int {
	var outOverL *uint256.Int
	if binLiquidity.IsZero() {
		outOverL = new(uint256.Int)
	} else {
		outOverL = divUp(output, binLiquidity)
	}

	if tokenAIn {
		// return MaverickBasicMath.mulDivUp(output, sqrtPrice, MaverickBasicMath.invFloor(sqrtPrice) - outOverL)
		invSqrtPrice := invFloor(sqrtPrice)
		denominator := new(uint256.Int).Sub(invSqrtPrice, outOverL)
		return mulDivUp(output, sqrtPrice, denominator)
	} else {
		// return MaverickBasicMath.divUp(output, MaverickBasicMath.mulDown(sqrtPrice, sqrtPrice - outOverL))
		numerator := new(uint256.Int).Sub(sqrtPrice, outOverL)
		denominator := mulDown(sqrtPrice, numerator)
		return divUp(output, denominator)
	}
}

// amountToBinNetOfProtocolFee calculates amount to bin after protocol fee, matching TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-swap-math.ts#L8
func amountToBinNetOfProtocolFee(deltaInErc, feeBasis *uint256.Int, protocolFeeD3 uint8) *uint256.Int {
	if protocolFeeD3 != 0 {
		protocolFee := mulDivUp(feeBasis, new(uint256.Int).SetUint64(uint64(protocolFeeD3)), BI_POWS[3])
		return clip(deltaInErc, protocolFee)
	}
	return new(uint256.Int).Set(deltaInErc)
}

// Helper function equivalent to MaverickPoolLib.binReserves
func binReserves(bin Bin, tickState Bin) (*uint256.Int, *uint256.Int) {
	if tickState.TotalSupply.IsZero() {
		return new(uint256.Int), new(uint256.Int)
	}

	// Calculate bin reserves proportionally based on tickBalance
	binA := mulDivDown(bin.TickBalance, tickState.ReserveA, tickState.TotalSupply)
	binB := mulDivDown(bin.TickBalance, tickState.ReserveB, tickState.TotalSupply)

	return binA, binB
}

// Helper function to convert tick data to tick state
func getTickState(state *MaverickPoolState, tick int32) *TickState {
	tickData, _ := getTickData(state, tick)

	// Create binIdsByTick map
	binIdsByTick := make(map[uint8]uint32)
	if binPositions, ok := state.BinPositions[tick]; ok {
		for _, binId := range binPositions {
			if bin, exists := state.Bins[binId]; exists {
				binIdsByTick[bin.Kind] = binId
			}
		}
	}

	return &TickState{
		ReserveA:     new(uint256.Int).Set(tickData.ReserveA),
		ReserveB:     new(uint256.Int).Set(tickData.ReserveB),
		TotalSupply:  new(uint256.Int).Set(tickData.TotalSupply),
		BinIdsByTick: binIdsByTick,
	}
}

// Helper function to update tick state back to the pool state
func updateTickState(state *MaverickPoolState, tick int32, tickState *TickState) {
	// Update individual bins based on the new tick state
	if binPositions, ok := state.BinPositions[tick]; ok {
		// Distribute the tick state reserves proportionally to bins
		for _, binId := range binPositions {
			if bin, exists := state.Bins[binId]; exists {
				// Update bin reserves proportionally (simplified)
				if !tickState.TotalSupply.IsZero() {
					bin.ReserveA = mulDivDown(bin.TickBalance, tickState.ReserveA, tickState.TotalSupply)
					bin.ReserveB = mulDivDown(bin.TickBalance, tickState.ReserveB, tickState.TotalSupply)
				}
				state.Bins[binId] = bin
			}
		}
	}

	// Update bin mappings based on tickState.BinIdsByTick
	for kind, binId := range tickState.BinIdsByTick {
		if binId == 0 {
			// Remove bin from this tick for this kind
			if binPositions, ok := state.BinPositions[tick]; ok {
				newPositions := make([]uint32, 0, len(binPositions))
				for _, id := range binPositions {
					if bin, exists := state.Bins[id]; exists && bin.Kind != kind {
						newPositions = append(newPositions, id)
					}
				}
				if len(newPositions) > 0 {
					state.BinPositions[tick] = newPositions
				} else {
					delete(state.BinPositions, tick)
				}
			}
		} else {
			// Ensure bin is in the tick positions
			if binPositions, ok := state.BinPositions[tick]; ok {
				found := false
				for _, id := range binPositions {
					if id == binId {
						found = true
						break
					}
				}
				if !found {
					state.BinPositions[tick] = append(binPositions, binId)
				}
			} else {
				state.BinPositions[tick] = []uint32{binId}
			}
		}
	}
}

// Helper for moveBins - processes a single tick
func processTick(state *MaverickPoolState, tick int32, direction int32) {
	// Skip if no bins at this tick
	binIds, ok := state.BinPositions[tick]
	if !ok || len(binIds) == 0 {
		return
	}

	// Process all bins at this tick
	for _, binId := range binIds {
		bin, ok := state.Bins[binId]
		if !ok {
			continue
		}

		// Skip if no reserves
		if bin.ReserveA.IsZero() && bin.ReserveB.IsZero() {
			continue
		}

		// Here we would implement the rebalancing logic based on direction
		// This is a simplified version that just shifts bins
		if direction > 0 {
			// Moving up, increase A, decrease B (simplified)
			shiftBin(&bin, true)
		} else {
			// Moving down, decrease A, increase B (simplified)
			shiftBin(&bin, false)
		}

		// Update bin
		state.Bins[binId] = bin
	}
}

// Helper to shift bin reserves when moving bins
func shiftBin(bin *Bin, increaseA bool) {
	// Skip if empty bin
	if bin.ReserveA.IsZero() && bin.ReserveB.IsZero() {
		return
	}

	// This is a simplified bin shift - actual implementation would depend on
	// Maverick's specific bin rebalancing formulas
	if increaseA {
		// Increase A, decrease B by a small percentage
		adjustment := mulDiv(bin.ReserveB, new(uint256.Int).SetUint64(1), new(uint256.Int).SetUint64(100))
		bin.ReserveA.Add(bin.ReserveA, adjustment)
		bin.ReserveB.Sub(bin.ReserveB, adjustment)
		if bin.ReserveB.IsZero() {
			bin.ReserveB = new(uint256.Int).SetUint64(1) // Ensure non-zero
		}
	} else {
		// Decrease A, increase B by a small percentage
		adjustment := mulDiv(bin.ReserveA, new(uint256.Int).SetUint64(1), new(uint256.Int).SetUint64(100))
		bin.ReserveA.Sub(bin.ReserveA, adjustment)
		bin.ReserveB.Add(bin.ReserveB, adjustment)
		if bin.ReserveA.IsZero() {
			bin.ReserveA = new(uint256.Int).SetUint64(1) // Ensure non-zero
		}
	}
}

// Implementation of getOrCreateBin from TypeScript
func getOrCreateBin(state *MaverickPoolState, kind uint8, tick int32) (uint32, Bin) {
	// First check if bin exists
	binId := binIdByTickKind(state, tick, kind)

	if binId == 0 {
		// Create a new bin
		state.BinCounter++
		binId = state.BinCounter

		// Initialize the new bin
		bin := Bin{
			Tick:            tick,
			Kind:            kind,
			MergeBinBalance: new(uint256.Int),
			MergeId:         0,
			TotalSupply:     new(uint256.Int),
			TickBalance:     new(uint256.Int),
			ReserveA:        new(uint256.Int),
			ReserveB:        new(uint256.Int),
		}

		// Store the bin
		state.Bins[binId] = bin

		// Create tick state if it doesn't exist
		_, ok := state.BinPositions[tick]
		if !ok {
			state.BinPositions[tick] = []uint32{}
		}

		// Add bin to tick
		state.BinPositions[tick] = append(state.BinPositions[tick], binId)

		// Create bin map entry
		state.BinMap[tick] = binId

		return binId, bin
	}

	// Return existing bin
	bin := state.Bins[binId]
	return binId, bin
}

// Helper to get absolute difference between two int64
func absDiff(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

// Helper to get current unix timestamp in seconds
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// Helper to safely clone uint256.Int (handles nil values)
func safeCloneUint256(value *uint256.Int) *uint256.Int {
	if value == nil {
		return new(uint256.Int)
	}
	return new(uint256.Int).Set(value)
}

// subTickIndex applies tick spacing and validates bounds
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L113
func subTickIndex(tickSpacing uint32, tick int32) *big.Int {
	// Get absolute value of tick using big.Int for precision
	subTick := big.NewInt(int64(tick))
	subTick.Abs(subTick)

	// Multiply by tickSpacing
	tickSpacingBig := big.NewInt(int64(tickSpacing))
	subTick.Mul(subTick, tickSpacingBig)

	// Check bounds
	if subTick.Cmp(MAX_TICK) > 0 {
		panic("OB") // Out of bounds - matching TypeScript error
	}

	return subTick
}

// Square root price and tick calculations matching the TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L122
func calculateSqrtPrice(tickSpacing uint32, tick int32) *uint256.Int {
	// Implementation matching TypeScript's tickSqrtPrice function
	// Apply tick spacing using subTickIndex
	subTick := subTickIndex(tickSpacing, tick)

	// Initialize ratio using big.Int for precise calculations, then convert to uint256.Int
	ratio := new(big.Int)
	if new(big.Int).And(subTick, big.NewInt(0x1)).Cmp(big.NewInt(0)) != 0 {
		ratio.SetString("fffcb933bd6fad9d3af5f0b9f25db4d6", 16)
	} else {
		ratio.SetString("100000000000000000000000000000000", 16)
	}

	// Apply bit shifts and multiplications matching the TypeScript implementation
	if new(big.Int).And(subTick, big.NewInt(0x2)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("fff97272373d41fd789c8cb37ffcaa1c", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x4)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("fff2e50f5f656ac9229c67059486f389", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x8)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("ffe5caca7e10e81259b3cddc7a064941", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x10)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("ffcb9843d60f67b19e8887e0bd251eb7", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x20)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("ff973b41fa98cd2e57b660be99eb2c4a", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x40)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("ff2ea16466c9838804e327cb417cafcb", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x80)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("fe5dee046a99d51e2cc356c2f617dbe0", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x100)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("fcbe86c7900aecf64236ab31f1f9dcb5", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x200)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("f987a7253ac4d9194200696907cf2e37", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x400)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("f3392b0822b88206f8abe8a3b44dd9be", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x800)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("e7159475a2c578ef4f1d17b2b235d480", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x1000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("d097f3bdfd254ee83bdd3f248e7e785e", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x2000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("a9f746462d8f7dd10e744d913d033333", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x4000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("70d869a156ddd32a39e257bc3f50aa9b", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x8000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("31be135f97da6e09a19dc367e3b6da40", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x10000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("9aa508b5b7e5a9780b0cc4e25d61a56", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x20000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("5d6af8dedbcb3a6ccb7ce618d14225", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x40000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("2216e584f630389b2052b8db590e", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x80000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("48a1703920644d4030024fe", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if new(big.Int).And(subTick, big.NewInt(0x100000)).Cmp(big.NewInt(0)) != 0 {
		mul := new(big.Int)
		mul.SetString("149b34ee7b4532", 16)
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}

	// If tick is positive, invert the ratio
	if tick > 0 {
		max := new(big.Int)
		max.SetString("ffffffffffffffffffffffffffffffff", 16)
		ratio = new(big.Int).Div(max, ratio)
	}

	// Multiply by 10^18 and shift right by 128, then convert to uint256.Int
	pow18 := new(big.Int)
	pow18.Exp(big.NewInt(10), big.NewInt(18), nil)
	ratio.Mul(ratio, pow18)
	ratio.Rsh(ratio, 128)

	result := new(uint256.Int)
	result.SetFromBig(ratio)
	return result
}

// getTickSqrtPriceAndL calculates both sqrt price and liquidity for a tick, matching TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L8
func getTickSqrtPriceAndL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *uint256.Int) (*uint256.Int, *uint256.Int) {
	// First calculate liquidity using getTickL logic
	liquidity := getTickL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice)

	// Then calculate sqrt price using getSqrtPrice logic
	sqrtPrice := getSqrtPrice(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice, liquidity)

	return sqrtPrice, liquidity
}

// getTickL calculates liquidity for a tick (internal helper)
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L60
func getTickL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *uint256.Int) *uint256.Int {
	precisionBump := uint(0)

	// Check if reserves are small (< 2^78) and apply precision bump
	if reserveA.Rsh(new(uint256.Int).Set(reserveA), 78).IsZero() && reserveB.Rsh(new(uint256.Int).Set(reserveB), 78).IsZero() {
		precisionBump = 57
		reserveA = new(uint256.Int).Lsh(reserveA, 57)
		reserveB = new(uint256.Int).Lsh(reserveB, 57)
	}

	// Calculate diff = sqrtUpperTickPrice - sqrtLowerTickPrice
	diff := new(uint256.Int).Sub(sqrtUpperTickPrice, sqrtLowerTickPrice)

	// Calculate b = divDown(reserveA, sqrtUpperTickPrice) + mulDown(reserveB, sqrtLowerTickPrice)
	term1 := mulDivDown(reserveA, BI_POWS[18], sqrtUpperTickPrice)
	term2 := mulDivDown(reserveB, sqrtLowerTickPrice, BI_POWS[18])
	b := new(uint256.Int).Add(term1, term2)

	// Handle special case: if either reserve is zero
	if reserveA.IsZero() || reserveB.IsZero() {
		result := mulDivDown(b, sqrtUpperTickPrice, diff)
		if precisionBump > 0 {
			result.Rsh(result, precisionBump)
		}
		return result
	}

	// b >>= 1 (divide by 2)
	b.Rsh(b, 1)

	/* Reference:
	TypeScript equivalent:

	MaverickBasicMath.mulDiv(
	       b +
	         MaverickBasicMath.sqrt(
	           MaverickBasicMath.mulDiv(b, b, BI_POWS[18]) +
	             MaverickBasicMath.mulDiv(
	               MaverickBasicMath.mulFloor(reserveB, reserveA),
	               diff,
	               sqrtUpperTickPrice,
	             ),
	         ) *
	           BI_POWS[9],
	       sqrtUpperTickPrice,
	       diff,
	     ) >> precisionBump
	*/
	// Calculate complex liquidity formula
	// sqrt(mulDiv(b, b, BI_POWS[18]) + mulDiv(mulFloor(reserveB, reserveA), diff, sqrtUpperTickPrice))
	bSquared := mulDivDown(b, b, BI_POWS[18])

	// mulFloor(reserveB, reserveA) = mulDiv(reserveB, reserveA, BI_POWS[18])
	mulFloorResult := mulDivDown(reserveB, reserveA, BI_POWS[18])
	term3 := mulDivDown(mulFloorResult, diff, sqrtUpperTickPrice)

	sqrtArg := new(uint256.Int).Add(bSquared, term3)
	sqrtResult := new(uint256.Int).Sqrt(sqrtArg)

	// Multiply by BI_POWS[9] (10^9)
	sqrtResult.Mul(sqrtResult, BI_POWS[9])

	// Final calculation: mulDiv(b + sqrtResult, sqrtUpperTickPrice, diff)
	numerator := new(uint256.Int).Add(b, sqrtResult)
	result := mulDivDown(numerator, sqrtUpperTickPrice, diff)

	if precisionBump > 0 {
		result.Rsh(result, precisionBump)
	}

	return result
}

// getSqrtPrice calculates the sqrt price based on reserves and liquidity (internal helper)
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L32
func getSqrtPrice(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice, liquidity *uint256.Int) *uint256.Int {
	if reserveA.IsZero() {
		return new(uint256.Int).Set(sqrtLowerTickPrice)
	}
	if reserveB.IsZero() {
		return new(uint256.Int).Set(sqrtUpperTickPrice)
	}

	// Calculate sqrtPrice = sqrt(BI_POWS[18] * divDown(
	//   reserveA + mulDown(liquidity, sqrtLowerTickPrice),
	//   reserveB + divDown(liquidity, sqrtUpperTickPrice)
	// ))

	// Numerator: reserveA + mulDown(liquidity, sqrtLowerTickPrice)
	liquidityTermA := mulDivDown(liquidity, sqrtLowerTickPrice, BI_POWS[18])
	numerator := new(uint256.Int).Add(reserveA, liquidityTermA)

	// Denominator: reserveB + divDown(liquidity, sqrtUpperTickPrice)
	liquidityTermB := mulDivDown(liquidity, BI_POWS[18], sqrtUpperTickPrice)
	denominator := new(uint256.Int).Add(reserveB, liquidityTermB)

	// Calculate ratio and apply BI_POWS[18] scaling
	ratio := mulDivDown(numerator, BI_POWS[18], denominator)
	sqrtPrice := new(uint256.Int).Sqrt(ratio)

	// Ensure the price is within bounds: min(max(sqrtPrice, sqrtLowerTickPrice), sqrtUpperTickPrice)
	if sqrtPrice.Cmp(sqrtLowerTickPrice) < 0 {
		sqrtPrice = new(uint256.Int).Set(sqrtLowerTickPrice)
	}
	if sqrtPrice.Cmp(sqrtUpperTickPrice) > 0 {
		sqrtPrice = new(uint256.Int).Set(sqrtUpperTickPrice)
	}

	return sqrtPrice
}

func tickSqrtPriceAndLiquidity(state *MaverickPoolState, tick int32) (*uint256.Int, *uint256.Int, *uint256.Int, Bin) {
	// Get the consolidated bin data (equivalent to this.state.ticks[tick.toString()])
	tickData, _ := getTickData(state, tick)

	// Calculate the square root prices at the tick boundaries using tickSqrtPrices
	// Equivalent to: let [sqrtLowerTickPrice, sqrtUpperTickPrice] = MaverickTickMath.tickSqrtPrices(this.tickSpacing, tick);
	sqrtLowerTickPrice := calculateSqrtPrice(state.TickSpacing, tick)
	sqrtUpperTickPrice := calculateSqrtPrice(state.TickSpacing, tick+1)

	// Calculate sqrt price and liquidity using the combined TypeScript logic
	// Equivalent to: [sqrtPrice, output.currentLiquidity] = MaverickTickMath.getTickSqrtPriceAndL(...)
	sqrtPrice, currentLiquidity := getTickSqrtPriceAndL(tickData.ReserveA, tickData.ReserveB, sqrtLowerTickPrice, sqrtUpperTickPrice)

	// Set the calculated liquidity in the tickData (matching TypeScript output.currentLiquidity)
	tickData.CurrentLiquidity = currentLiquidity

	// Return [sqrtLowerTickPrice, sqrtUpperTickPrice, sqrtPrice, output] - matches TypeScript exactly
	return sqrtLowerTickPrice, sqrtUpperTickPrice, sqrtPrice, tickData
}

// Errors
var (
	ErrEmptyBins = fmt.Errorf("empty bins")
	ErrOverflow  = fmt.Errorf("overflow")
)

// BigInt powers for various calculations
var (
	BI_POWS = func() [40]*uint256.Int {
		pows := [40]*uint256.Int{}
		for i := 0; i < 40; i++ {
			pows[i] = new(uint256.Int).Exp(new(uint256.Int).SetUint64(10), new(uint256.Int).SetUint64(uint64(i)))
		}
		return pows
	}()
)
