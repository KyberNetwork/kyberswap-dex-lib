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
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	binMap := extra.BinMap
	binPositions := extra.BinPositions

	// Parse accumValueD8 from string to uint256
	var accumValueD8 *uint256.Int
	if extra.AccumValueD8 != "" {
		var err error
		accumValueD8, err = uint256.FromDecimal(extra.AccumValueD8)
		if err != nil {
			accumValueD8 = new(uint256.Int)
		}
	} else {
		accumValueD8 = new(uint256.Int)
	}

	// Default lookback to 10 minutes if not specified
	lookbackSec := extra.LookbackSec
	if lookbackSec == 0 {
		lookbackSec = 600 // 10 minutes in seconds
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
				Reserves: []*big.Int{utils.NewBig10(entityPool.Reserves[0]), utils.NewBig10(entityPool.Reserves[1])},
			},
		},
		decimals: []uint8{entityPool.Tokens[0].Decimals, entityPool.Tokens[1].Decimals},
		state: &MaverickPoolState{
			FeeAIn:           extra.FeeAIn,
			FeeBIn:           extra.FeeBIn,
			ProtocolFeeRatio: extra.ProtocolFeeRatio,
			Bins:             extra.Bins,
			BinPositions:     binPositions,
			BinMap:           binMap,
			TickSpacing:      staticExtra.TickSpacing,
			ActiveTick:       extra.ActiveTick,
			LastTwaD8:        extra.LastTwaD8,
			Timestamp:        extra.Timestamp,
			AccumValueD8:     accumValueD8,
			LookbackSec:      lookbackSec,
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
	fractionalPartD8 := newState.fractionalPartD8
	if fractionalPartD8 == 0 {
		// Default to half the tick if not provided
		fractionalPartD8 = int64(BI_POWS[7].Uint64())
	}

	// Calculate full tick position with fractional part
	tickPositionD8 := int64(p.state.ActiveTick)*int64(BI_POWS[8].Uint64()) + fractionalPartD8

	// Update TWA
	updateTwaValue(p.state, tickPositionD8, timestamp)

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

	// In JS, we check if the swap limit is beyond the current tick
	if (startingTick > tickLimit && tokenAIn) || (startingTick < tickLimit && !tokenAIn) {
		return nil, nil, 0, new(uint256.Int), fmt.Errorf("beyond swap limit")
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

func swapTick(state *MaverickPoolState, delta *Delta, tickLimit int32) (*Delta, bool, error) {
	// Implementation based on maverick-v2 JavaScript logic

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
	if (activeTick > tickLimit && delta.TokenAIn) || (activeTick < tickLimit && !delta.TokenAIn) {
		state.ActiveTick += boolToInt32(!delta.TokenAIn) - boolToInt32(delta.TokenAIn)
		newDelta.SwappedToMaxPrice = true // Set this flag when we hit the tick limit
		return delta, true, nil
	}

	// Find next tick with liquidity
	crossedBin := false
	ticksSearched := 0
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
		ticksSearched++

		// Check again if we've reached the tick limit after moving
		if (activeTick > tickLimit && delta.TokenAIn) || (activeTick < tickLimit && !delta.TokenAIn) {
			state.ActiveTick += boolToInt32(!delta.TokenAIn) - boolToInt32(delta.TokenAIn)
			newDelta.SwappedToMaxPrice = true // Set this flag when we hit the tick limit
			return delta, true, nil
		}

		// Safety check to avoid infinite loops
		if ticksSearched > 1000 {
			return nil, false, fmt.Errorf("too many ticks searched without finding liquidity")
		}
	}

	state.ActiveTick = activeTick

	// Here's the key change: Calculate the sqrt prices using tickSqrtPriceAndLiquidity
	// This matches the TypeScript code: [delta.sqrtLowerTickPrice, delta.sqrtUpperTickPrice, delta.sqrtPrice, tickData] = this.tickSqrtPriceAndLiquidity(activeTick)
	newDelta.SqrtLowerTickPrice, newDelta.SqrtUpperTickPrice, newDelta.SqrtPrice, tickData = tickSqrtPriceAndLiquidity(state, activeTick)

	// Perform the actual swap computation
	if delta.ExactOutput {
		computeSwapExactOut(state, delta.Excess, delta.TokenAIn, activeTick, newDelta)
	} else {
		computeSwapExactIn(state, delta.Excess, delta.TokenAIn, activeTick, newDelta)
	}

	// If there's excess remaining, we need to move to the next tick
	if !newDelta.Excess.IsZero() {
		nextTick := activeTick + boolToInt32(delta.TokenAIn) - boolToInt32(!delta.TokenAIn)
		state.ActiveTick = nextTick
		crossedBin = true
	}

	// Calculate the fractional part based on the position within the tick
	// In the JavaScript code, this would be used for TWA updates
	if !newDelta.SqrtPrice.IsZero() && !newDelta.SqrtLowerTickPrice.IsZero() && !newDelta.SqrtUpperTickPrice.IsZero() {
		// Calculate how far we are between the lower and upper tick prices
		_range := new(uint256.Int).Sub(newDelta.SqrtUpperTickPrice, newDelta.SqrtLowerTickPrice)
		position := new(uint256.Int).Sub(newDelta.SqrtPrice, newDelta.SqrtLowerTickPrice)

		if !_range.IsZero() {
			// Calculate the fractional part as a value between 0 and 2^8
			newDelta.FractionalPart = mulDiv(position, BI_POWS[8], _range)
		}
	}

	return newDelta, crossedBin, nil
}

func computeSwapExactIn(state *MaverickPoolState, amountIn *uint256.Int, tokenAIn bool, activeTick int32, delta *Delta) {
	tickData, _ := getTickData(state, activeTick)

	// Get the output reserve
	var amountOutAvailable *uint256.Int
	if tokenAIn {
		amountOutAvailable = new(uint256.Int).Set(tickData.ReserveB)
	} else {
		amountOutAvailable = new(uint256.Int).Set(tickData.ReserveA)
	}

	// Calculate fee
	var fee uint32
	if tokenAIn {
		fee = state.FeeAIn
	} else {
		fee = state.FeeBIn
	}

	// Calculate the amount that will go to the pool after fee
	feeAmount := mulDiv(amountIn, new(uint256.Int).SetUint64(uint64(fee)), BI_POWS[18])
	netAmountIn := new(uint256.Int).Sub(amountIn, feeAmount)

	// Protocol fee
	protocolFee := mulDiv(feeAmount, new(uint256.Int).SetUint64(uint64(state.ProtocolFeeRatio)), BI_POWS[3])
	amountToBin := new(uint256.Int).Sub(amountIn, protocolFee)

	// Calculate the amount out based on the available reserves
	// This is a simplified approximation of the JS logic
	var amountOut *uint256.Int
	if tokenAIn {
		// A to B swap (token0 to token1)
		// Use constant product formula: k = x * y
		// newY = k / newX where newX = x + netAmountIn
		oldK := new(uint256.Int).Mul(tickData.ReserveA, tickData.ReserveB)
		newReserveA := new(uint256.Int).Add(tickData.ReserveA, netAmountIn)
		newReserveB := divRoundingUp(oldK, newReserveA)
		amountOut = new(uint256.Int).Sub(tickData.ReserveB, newReserveB)

		// Ensure we don't output more than available
		if amountOut.Cmp(amountOutAvailable) > 0 {
			amountOut = new(uint256.Int).Set(amountOutAvailable)
		}

		// Update the reserves
		tickData.ReserveA = newReserveA
		tickData.ReserveB = new(uint256.Int).Sub(tickData.ReserveB, amountOut)
	} else {
		// B to A swap (token1 to token0)
		oldK := new(uint256.Int).Mul(tickData.ReserveA, tickData.ReserveB)
		newReserveB := new(uint256.Int).Add(tickData.ReserveB, netAmountIn)
		newReserveA := divRoundingUp(oldK, newReserveB)
		amountOut = new(uint256.Int).Sub(tickData.ReserveA, newReserveA)

		if amountOut.Cmp(amountOutAvailable) > 0 {
			amountOut = new(uint256.Int).Set(amountOutAvailable)
		}

		tickData.ReserveB = newReserveB
		tickData.ReserveA = new(uint256.Int).Sub(tickData.ReserveA, amountOut)
	}

	// Set delta values
	delta.DeltaInBinInternal = amountToBin
	delta.DeltaInErc = amountIn
	delta.DeltaOutErc = amountOut

	// If we've consumed all reserves, mark as excess
	if amountOut.Cmp(amountOutAvailable) >= 0 {
		delta.Excess = new(uint256.Int).Set(amountIn)
	} else {
		delta.Excess = new(uint256.Int)
	}

	// Update the bin data in the state
	updateBinData(state, activeTick, tickData)
}

func computeSwapExactOut(state *MaverickPoolState, amountOut *uint256.Int, tokenAIn bool, activeTick int32, delta *Delta) {
	tickData, _ := getTickData(state, activeTick)

	// Get the output reserve
	var amountOutAvailable *uint256.Int
	if tokenAIn {
		amountOutAvailable = new(uint256.Int).Set(tickData.ReserveB)
	} else {
		amountOutAvailable = new(uint256.Int).Set(tickData.ReserveA)
	}

	// Check if we have enough liquidity
	var swapped bool
	if amountOutAvailable.Cmp(amountOut) <= 0 {
		swapped = true
		delta.DeltaOutErc = new(uint256.Int).Set(amountOutAvailable)
	} else {
		delta.DeltaOutErc = new(uint256.Int).Set(amountOut)
	}

	// Calculate fee
	var fee uint32
	if tokenAIn {
		fee = state.FeeAIn
	} else {
		fee = state.FeeBIn
	}

	// Calculate input amount based on output amount
	// This is a simplified approximation of the JS logic
	var amountIn *uint256.Int
	if tokenAIn {
		// A to B swap (token0 to token1)
		oldK := new(uint256.Int).Mul(tickData.ReserveA, tickData.ReserveB)
		newReserveB := new(uint256.Int).Sub(tickData.ReserveB, delta.DeltaOutErc)
		newReserveA := divRoundingUp(oldK, newReserveB)
		rawAmountIn := new(uint256.Int).Sub(newReserveA, tickData.ReserveA)

		// Add fee to get gross amount in
		feeMultiplier := divRoundingUp(BI_POWS[18], new(uint256.Int).Sub(BI_POWS[18], new(uint256.Int).SetUint64(uint64(fee))))
		amountIn = mulDiv(rawAmountIn, feeMultiplier, BI_POWS[18])

		// Update the reserves
		tickData.ReserveA = newReserveA
		tickData.ReserveB = newReserveB
	} else {
		// B to A swap (token1 to token0)
		oldK := new(uint256.Int).Mul(tickData.ReserveA, tickData.ReserveB)
		newReserveA := new(uint256.Int).Sub(tickData.ReserveA, delta.DeltaOutErc)
		newReserveB := divRoundingUp(oldK, newReserveA)
		rawAmountIn := new(uint256.Int).Sub(newReserveB, tickData.ReserveB)

		feeMultiplier := divRoundingUp(BI_POWS[18], new(uint256.Int).Sub(BI_POWS[18], new(uint256.Int).SetUint64(uint64(fee))))
		amountIn = mulDiv(rawAmountIn, feeMultiplier, BI_POWS[18])

		tickData.ReserveA = newReserveA
		tickData.ReserveB = newReserveB
	}

	// Protocol fee
	feeAmount := mulDiv(amountIn, new(uint256.Int).SetUint64(uint64(fee)), BI_POWS[18])
	protocolFee := mulDiv(feeAmount, new(uint256.Int).SetUint64(uint64(state.ProtocolFeeRatio)), BI_POWS[3])
	amountToBin := new(uint256.Int).Sub(amountIn, protocolFee)

	// Set delta values
	delta.DeltaInBinInternal = amountToBin
	delta.DeltaInErc = amountIn

	// If we've consumed all reserves, mark as excess
	if swapped {
		delta.Excess = new(uint256.Int).Set(amountOut)
	} else {
		delta.Excess = new(uint256.Int)
	}

	// Update the bin data in the state
	updateBinData(state, activeTick, tickData)
}

func getTickData(state *MaverickPoolState, tick int32) (Bin, bool) {
	bins, ok := state.BinPositions[tick]
	if !ok || len(bins) == 0 {
		return Bin{
			ReserveA: new(uint256.Int),
			ReserveB: new(uint256.Int),
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
		ReserveA: consolidatedReserveA,
		ReserveB: consolidatedReserveB,
	}, true
}

func updateBinData(state *MaverickPoolState, tick int32, tickData Bin) {
	bins, ok := state.BinPositions[tick]
	if !ok || len(bins) == 0 {
		return
	}

	// Distribute the new reserves proportionally across bins
	// This is a simplification - in the actual implementation, each bin may have specific logic
	totalReserveA := new(uint256.Int)
	totalReserveB := new(uint256.Int)

	for _, binId := range bins {
		bin, ok := state.Bins[binId]
		if ok {
			totalReserveA = new(uint256.Int).Add(totalReserveA, bin.ReserveA)
			totalReserveB = new(uint256.Int).Add(totalReserveB, bin.ReserveB)
		}
	}

	// Update each bin proportionally
	for _, binId := range bins {
		bin, ok := state.Bins[binId]
		if ok {
			// Calculate new reserves proportionally
			if !totalReserveA.IsZero() {
				ratio := divRoundingDown(mulDiv(bin.ReserveA, BI_POWS[18], totalReserveA), BI_POWS[18])
				bin.ReserveA = mulDiv(tickData.ReserveA, ratio, BI_POWS[18])
			}

			if !totalReserveB.IsZero() {
				ratio := divRoundingDown(mulDiv(bin.ReserveB, BI_POWS[18], totalReserveB), BI_POWS[18])
				bin.ReserveB = mulDiv(tickData.ReserveB, ratio, BI_POWS[18])
			}

			state.Bins[binId] = bin
		}
	}
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
		LookbackSec:      state.LookbackSec,
		BinCounter:       state.BinCounter,
	}

	// Clone the accumulated value
	if state.AccumValueD8 != nil {
		cloned.AccumValueD8 = new(uint256.Int).Set(state.AccumValueD8)
	} else {
		cloned.AccumValueD8 = new(uint256.Int)
	}

	for k, v := range state.Bins {
		clonedBin := Bin{
			ReserveA:        new(uint256.Int).Set(v.ReserveA),
			ReserveB:        new(uint256.Int).Set(v.ReserveB),
			MergeBinBalance: new(uint256.Int).Set(v.MergeBinBalance),
			TotalSupply:     new(uint256.Int).Set(v.TotalSupply),
			TickBalance:     new(uint256.Int).Set(v.TickBalance),
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
	FeeAIn           uint32
	FeeBIn           uint32
	ProtocolFeeRatio uint8
	Bins             map[uint32]Bin
	BinPositions     map[int32][]uint32
	BinMap           map[int32]uint32
	TickSpacing      uint32
	ActiveTick       int32
	LastTwaD8        int64        // Time-weighted average tick data
	Timestamp        int64        // Current timestamp
	AccumValueD8     *uint256.Int // Accumulated TWA value with 8 decimals
	LookbackSec      int64        // Lookback period in seconds
	BinCounter       uint32       // Counter for bin IDs
}

type Extra struct {
	FeeAIn           uint32             `json:"feeAIn"`
	FeeBIn           uint32             `json:"feeBIn"`
	ProtocolFeeRatio uint8              `json:"protocolFeeRatio"`
	Bins             map[uint32]Bin     `json:"bins"`
	BinPositions     map[int32][]uint32 `json:"binPositions"`
	BinMap           map[int32]uint32   `json:"binMap"`
	ActiveTick       int32              `json:"activeTick"`
	LastTwaD8        int64              `json:"lastTwaD8"`
	Timestamp        int64              `json:"timestamp"`
	AccumValueD8     string             `json:"accumValueD8"`
	LookbackSec      int64              `json:"lookbackSec"`
}

type StaticExtra struct {
	TickSpacing uint32 `json:"tickSpacing"`
}

type Bin struct {
	MergeBinBalance *uint256.Int `json:"mergeBinBalance"`
	MergeId         uint32       `json:"mergeId"`
	TotalSupply     *uint256.Int `json:"totalSupply"`
	Kind            uint8        `json:"kind"`
	Tick            int32        `json:"tick"`
	TickBalance     *uint256.Int `json:"tickBalance"`
	ReserveA        *uint256.Int `json:"reserveA"`
	ReserveB        *uint256.Int `json:"reserveB"`
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

// TWA and Bin movement related functions
func updateTwaValue(state *MaverickPoolState, newValueD8 int64, timestamp int64) {
	// Skip if no timestamp change
	if state.Timestamp == timestamp {
		return
	}

	// Handle initial case
	if state.LastTwaD8 == 0 {
		state.LastTwaD8 = newValueD8
		state.Timestamp = timestamp
		return
	}

	// Calculate time delta
	timeDelta := timestamp - state.Timestamp

	// Ensure we don't have negative time
	if timeDelta <= 0 {
		return
	}

	// Calculate weighted value to add to accumulator
	weightedValue := new(uint256.Int).SetUint64(uint64(state.LastTwaD8 * timeDelta))

	// Add to accumulator
	if state.AccumValueD8 == nil {
		state.AccumValueD8 = new(uint256.Int)
	}
	state.AccumValueD8.Add(state.AccumValueD8, weightedValue)

	// Update state
	state.LastTwaD8 = newValueD8
	state.Timestamp = timestamp
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

// Square root price and tick calculations matching the TypeScript implementation
func calculateSqrtPrice(tick int32) *uint256.Int {
	// Implementation matching TypeScript's tickSqrtPrice function
	// First apply tick spacing - for now assuming spacing is 1
	subTick := uint64(tick)
	if tick < 0 {
		subTick = uint64(-tick)
	}

	// Check if the tick is within valid range
	const MAX_TICK = 460540
	if subTick > MAX_TICK {
		return new(uint256.Int) // Return 0 if out of bounds
	}

	// Initialize ratio
	ratio := new(uint256.Int)
	if subTick&0x1 != 0 {
		ratio.SetFromHex("0xfffcb933bd6fad9d3af5f0b9f25db4d6")
	} else {
		ratio.SetFromHex("0x100000000000000000000000000000000")
	}

	// Apply bit shifts and multiplications matching the TypeScript implementation
	if subTick&0x2 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xfff97272373d41fd789c8cb37ffcaa1c")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x4 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xfff2e50f5f656ac9229c67059486f389")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x8 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xffe5caca7e10e81259b3cddc7a064941")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x10 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xffcb9843d60f67b19e8887e0bd251eb7")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x20 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xff973b41fa98cd2e57b660be99eb2c4a")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x40 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xff2ea16466c9838804e327cb417cafcb")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x80 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xfe5dee046a99d51e2cc356c2f617dbe0")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x100 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xfcbe86c7900aecf64236ab31f1f9dcb5")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x200 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xf987a7253ac4d9194200696907cf2e37")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x400 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xf3392b0822b88206f8abe8a3b44dd9be")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x800 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xe7159475a2c578ef4f1d17b2b235d480")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x1000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xd097f3bdfd254ee83bdd3f248e7e785e")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x2000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0xa9f746462d8f7dd10e744d913d033333")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x4000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x70d869a156ddd32a39e257bc3f50aa9b")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x8000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x31be135f97da6e09a19dc367e3b6da40")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x10000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x9aa508b5b7e5a9780b0cc4e25d61a56")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x20000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x5d6af8dedbcb3a6ccb7ce618d14225")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x40000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x2216e584f630389b2052b8db590e")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x80000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x48a1703920644d4030024fe")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}
	if subTick&0x100000 != 0 {
		mul := new(uint256.Int)
		mul.SetFromHex("0x149b34ee7b4532")
		ratio.Mul(ratio, mul)
		ratio.Rsh(ratio, 128)
	}

	// If tick is positive, invert the ratio
	if tick > 0 {
		max := new(uint256.Int)
		max.SetFromHex("0xffffffffffffffffffffffffffffffff")
		ratio = new(uint256.Int).Div(max, ratio)
	}

	// Multiply by 10^18 and shift right by 128
	ratio.Mul(ratio, BI_POWS[18])
	ratio.Rsh(ratio, 128)

	return ratio
}

func tickSqrtPriceAndLiquidity(state *MaverickPoolState, tick int32) (*uint256.Int, *uint256.Int, *uint256.Int, Bin) {
	// Calculate the square root prices at the tick boundaries
	sqrtLowerTickPrice := calculateSqrtPrice(tick)
	sqrtUpperTickPrice := calculateSqrtPrice(tick + 1)

	// Get the consolidated bin data
	tickData, _ := getTickData(state, tick)

	// Calculate the current sqrt price based on reserves
	// In the TypeScript code, this would be a complex calculation
	// Here we're using a simpler approach based on the reserves ratio
	var sqrtPrice *uint256.Int

	if tickData.ReserveA.IsZero() || tickData.ReserveB.IsZero() {
		// Default to the lower tick price if we don't have both reserves
		sqrtPrice = new(uint256.Int).Set(sqrtLowerTickPrice)
	} else {
		// Sqrt(reserveB / reserveA) * sqrtLowerTickPrice
		ratio := new(uint256.Int).Div(tickData.ReserveB, tickData.ReserveA)
		sqrtRatio := new(uint256.Int).Sqrt(ratio)
		sqrtPrice = mulDiv(sqrtRatio, sqrtLowerTickPrice, new(uint256.Int).SetUint64(1<<48)) // Adjust for Q format
	}

	// Ensure the price is within the tick bounds
	if sqrtPrice.Cmp(sqrtLowerTickPrice) < 0 {
		sqrtPrice = new(uint256.Int).Set(sqrtLowerTickPrice)
	}
	if sqrtPrice.Cmp(sqrtUpperTickPrice) > 0 {
		sqrtPrice = new(uint256.Int).Set(sqrtUpperTickPrice)
	}

	return sqrtLowerTickPrice, sqrtUpperTickPrice, sqrtPrice, tickData
}

// Errors
var (
	ErrEmptyBins = fmt.Errorf("empty bins")
	ErrOverflow  = fmt.Errorf("overflow")
)

// BigInt powers for various calculations
var (
	BI_POWS = func() [20]*uint256.Int {
		pows := [20]*uint256.Int{}
		for i := 0; i < 20; i++ {
			pows[i] = new(uint256.Int).Exp(new(uint256.Int).SetUint64(10), new(uint256.Int).SetUint64(uint64(i)))
		}
		return pows
	}()
)
