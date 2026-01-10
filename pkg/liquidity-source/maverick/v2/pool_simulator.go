package maverickv2

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	maverickv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maverick/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
				Reserves: []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]),
					bignumber.NewBig10(entityPool.Reserves[1])},
			},
		},
		decimals: []uint8{entityPool.Tokens[0].Decimals, entityPool.Tokens[1].Decimals},
		state: &MaverickPoolState{
			FeeAIn:           extra.FeeAIn,
			FeeBIn:           extra.FeeBIn,
			ProtocolFeeRatio: extra.ProtocolFeeRatio,
			Bins:             extra.Bins,
			Ticks:            extra.Ticks,
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
	// scale to AMM Amount 10^18
	scaledAmountIn := scaleFromAmount(amountIn, p.decimals[tokenInIndex])

	newState := p.state.Clone()
	_, amountOut, binCrossed, err := swap(newState, scaledAmountIn, tokenInIndex == 0, false, false)
	if err != nil {
		return nil, fmt.Errorf("can not get amount out, err: %v", err)
	}

	// scale back to token amount
	scaledAmountOut := ScaleToAmount(amountOut, p.decimals[tokenOutIndex])

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: scaledAmountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		},
		Gas: GasSwap + GasCrossBin*int64(binCrossed),
		SwapInfo: maverickSwapInfo{
			activeTick:       newState.ActiveTick,
			bins:             newState.Bins,
			ticks:            newState.Ticks,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	// scale to AMM Amount 10^18
	scaledAmountOut := scaleFromAmount(amountOut, p.decimals[tokenOutIndex])

	newState := p.state.Clone()
	_, amountIn, binCrossed, err := swap(newState, scaledAmountOut, tokenInIndex == 0, true, false)
	if err != nil {
		return nil, fmt.Errorf("can not get amount out, err: %v", err)
	}

	// scale back to token amount
	scaledAmountIn := ScaleToAmount(amountIn, p.decimals[tokenInIndex])

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: scaledAmountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: GasSwap + GasCrossBin*int64(binCrossed),
		SwapInfo: maverickSwapInfo{
			activeTick:       newState.ActiveTick,
			bins:             newState.Bins,
			ticks:            newState.Ticks,
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
		logger.Warn("failed to UpdateBalance for Maverick pool, wrong swapInfo type")
		return
	}

	// Store old values for TWA and bin movements
	startingTick := p.state.ActiveTick
	lastTwaD8 := p.state.LastTwaD8

	// Update the primary state values from swap result
	p.state.Bins = newState.bins
	p.state.Ticks = newState.ticks
	p.state.ActiveTick = newState.activeTick

	// Move bins based on tick changes
	moveBins(p.state, startingTick, p.state.ActiveTick, lastTwaD8, p.state.LastTwaD8)

	tokenAmountIn, tokenAmountOut := params.TokenAmountIn, params.TokenAmountOut
	// Update reserves based on swap direction
	if isTokenAIn := strings.EqualFold(tokenAmountIn.Token, p.Info.Tokens[0]); isTokenAIn {
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], tokenAmountIn.Amount)
		p.Info.Reserves[1] = new(big.Int).Sub(p.Info.Reserves[1], tokenAmountOut.Amount)
	} else {
		p.Info.Reserves[0] = new(big.Int).Sub(p.Info.Reserves[0], tokenAmountOut.Amount)
		p.Info.Reserves[1] = new(big.Int).Add(p.Info.Reserves[1], tokenAmountIn.Amount)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return nil
}

// pastMaxTick checks if we've reached the tick limit and zeros out excess if so
// This is equivalent to MaverickDeltaMath.pastMaxTick() in TypeScript
func pastMaxTick(delta *Delta, activeTick, tickLimit int32) bool {
	swappedToMaxPrice := delta.TokenAIn == (tickLimit < activeTick)
	if swappedToMaxPrice {
		delta.Excess = big256.U0 // CRITICAL: Zero out excess to terminate main loop
		delta.SkipCombine = true
		delta.SwappedToMaxPrice = true
	}
	return swappedToMaxPrice
}

// Helper functions for swap implementation
func swap(state *MaverickPoolState, amount *uint256.Int, tokenAIn bool, exactOutput bool, _ bool) (*uint256.Int,
	*uint256.Int, uint32, error) {
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
			return nil, nil, 0, err
		}

		if crossedBin {
			binCrossed++
		}

		combine(delta, newDelta)
	}

	return delta.DeltaInErc, delta.DeltaOutErc, binCrossed, nil
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
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
	}

	activeTick := state.ActiveTick

	// Check if we've reached the tick limit - equivalent to TypeScript pastMaxTick function
	if pastMaxTick(delta, activeTick, tickLimit) {
		state.ActiveTick += lo.Ternary[int32](delta.TokenAIn, -1, 1)
		return delta, true, nil
	}

	// Find next tick with liquidity
	crossedBin := false

	for {
		if tickData, ok := state.Ticks[activeTick]; ok &&
			(tickData.ReserveA.BitLen() > 0 || tickData.ReserveB.BitLen() > 0) {
			break
		}

		// Move to next tick in correct direction
		activeTick += lo.Ternary[int32](delta.TokenAIn, 1, -1)
		crossedBin = true

		// Check again if we've reached the tick limit after moving
		if pastMaxTick(delta, activeTick, tickLimit) {
			state.ActiveTick += lo.Ternary[int32](delta.TokenAIn, -1, 1)
			return delta, true, nil
		}
	}

	state.ActiveTick = activeTick

	// Here's the key change: Calculate the sqrt prices using tickSqrtPriceAndLiquidity
	// This matches the TypeScript code: [delta.sqrtLowerTickPrice, delta.sqrtUpperTickPrice, delta.sqrtPrice, tickData] = this.tickSqrtPriceAndLiquidity(activeTick)
	var tickDataFromLiquidity TickData
	newDelta.SqrtLowerTickPrice, newDelta.SqrtUpperTickPrice, newDelta.SqrtPrice, tickDataFromLiquidity =
		tickSqrtPriceAndLiquidity(state, activeTick)

	// Perform the actual swap computation
	if delta.ExactOutput {
		*newDelta = computeSwapExactOut(state, delta.Excess, delta.TokenAIn, tickDataFromLiquidity, newDelta.SqrtPrice)
	} else {
		*newDelta = computeSwapExactIn(state, delta.Excess, delta.TokenAIn, tickDataFromLiquidity, newDelta.SqrtPrice)
	}

	// Match TypeScript logic exactly
	if newDelta.Excess.IsZero() {
		computeEndPrice(delta, newDelta, tickDataFromLiquidity)
	}

	// allocateSwapValuesToTick (only mutate cloned state)
	allocateSwapValuesToTick(state, newDelta, delta.TokenAIn, state.ActiveTick)

	// If there's excess remaining, we need to move to the next tick
	if !newDelta.Excess.IsZero() {
		state.ActiveTick = activeTick + lo.Ternary[int32](delta.TokenAIn, 1, -1)
		crossedBin = true
	}

	return newDelta, crossedBin, nil
}

// computeEndPrice calculates the end price and fractional part when there's no excess remaining
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-swap-math.ts#L178
func computeEndPrice(delta *Delta, newDelta *Delta, tickData TickData) {
	// Calculate endSqrtPrice following TypeScript logic exactly
	// endSqrtPrice = divDown(newDelta.deltaInBinInternal, tickData.currentLiquidity) +
	//                (delta.tokenAIn ? delta.sqrtPrice : invFloor(delta.sqrtPrice))

	var endSqrtPrice *uint256.Int
	if tickData.CurrentLiquidity.IsZero() {
		endSqrtPrice = new(uint256.Int)
	} else {
		endSqrtPrice = mulDivDown(newDelta.DeltaInBinInternal, big256.TenPow(18), tickData.CurrentLiquidity)
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
}

func computeSwapExactIn(state *MaverickPoolState, amountIn *uint256.Int, tokenAIn bool, tickData TickData,
	sqrtPrice *uint256.Int) Delta {
	// Create a new delta - matching TypeScript exactly
	delta := Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int),
		TokenAIn:           tokenAIn,
		ExactOutput:        false,
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		TickLimit:          0,
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int).Set(sqrtPrice),
	}

	// Set initial deltaOutErc to all available reserves - line 68-70 in TypeScript
	if tokenAIn {
		delta.DeltaOutErc = new(uint256.Int).Set(tickData.CurrentReserveB) // currentReserveB in TypeScript
	} else {
		delta.DeltaOutErc = new(uint256.Int).Set(tickData.CurrentReserveA) // currentReserveA in TypeScript
	}

	// Calculate remaining bin input space given the output - lines 72-77 in TypeScript
	binAmountIn := remainingBinInputSpaceGivenOutput(
		tickData.CurrentLiquidity, // currentLiquidity in TypeScript
		delta.DeltaOutErc,
		delta.SqrtPrice,
		tokenAIn,
	)

	// Get fee basis
	fee := uint256.NewInt(lo.Ternary(tokenAIn, state.FeeAIn, state.FeeBIn))

	// Calculate user bin amount in - lines 80-83 in TypeScript
	userBinAmountIn := mulDown(amountIn, new(uint256.Int).Sub(big256.TenPow(18), fee))

	var feeBasis *uint256.Int

	// Logic for determining actual binAmountIn and fees - lines 85-97 in TypeScript
	if userBinAmountIn.Cmp(binAmountIn) < 0 {
		binAmountIn = userBinAmountIn
		delta.DeltaInErc = new(uint256.Int).Set(amountIn)
		feeBasis = new(uint256.Int).Sub(delta.DeltaInErc, userBinAmountIn)
	} else {
		feeBasis = mulDivUp(binAmountIn, fee, new(uint256.Int).Sub(big256.TenPow(18), fee))
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
		return delta
	}

	// Calculate inOverL - lines 107-110 in TypeScript
	inOverL := divUp(binAmountIn, new(uint256.Int).AddUint64(tickData.CurrentLiquidity, 1))

	// Calculate final deltaOutErc - lines 112-119 in TypeScript
	var calculatedOut *uint256.Int
	if tokenAIn {
		// delta.deltaOutErc = MaverickBasicMath.min(delta.deltaOutErc, MaverickBasicMath.mulDivDown(
		//   binAmountIn, MaverickBasicMath.invFloor(sqrtPrice), inOverL + sqrtPrice))
		denominator := inOverL.Add(inOverL, delta.SqrtPrice)
		calculatedOut = mulDivDown(binAmountIn, invFloor(delta.SqrtPrice), denominator)
	} else {
		// delta.deltaOutErc = MaverickBasicMath.min(delta.deltaOutErc, MaverickBasicMath.mulDivDown(
		//   binAmountIn, sqrtPrice, inOverL + MaverickBasicMath.invCeil(sqrtPrice)))
		denominator := inOverL.Add(inOverL, invCeil(delta.SqrtPrice))
		calculatedOut = mulDivDown(binAmountIn, delta.SqrtPrice, denominator)
	}

	delta.DeltaOutErc = big256.Min(delta.DeltaOutErc, calculatedOut)

	return delta
}

func computeSwapExactOut(state *MaverickPoolState, amountOut *uint256.Int, tokenAIn bool, tickData TickData,
	sqrtPrice *uint256.Int) Delta {
	// Create a new delta - matching TypeScript exactly
	delta := Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int),
		TokenAIn:           false, // Note: TypeScript sets this to false initially
		ExactOutput:        false, // Note: TypeScript sets this to false initially
		SwappedToMaxPrice:  false,
		SkipCombine:        false,
		TickLimit:          0,
		SqrtLowerTickPrice: new(uint256.Int),
		SqrtUpperTickPrice: new(uint256.Int),
		SqrtPrice:          new(uint256.Int).Set(sqrtPrice),
	}

	// Determine available output amount - lines 148-150 in TypeScript
	amountOutAvailable := new(uint256.Int)
	if tokenAIn {
		amountOutAvailable.Set(tickData.CurrentReserveB) // currentReserveB in TypeScript
	} else {
		amountOutAvailable.Set(tickData.CurrentReserveA) // currentReserveA in TypeScript
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
	fee := uint256.NewInt(lo.Ternary(tokenAIn, state.FeeAIn, state.FeeBIn))

	// feeBasis = MaverickBasicMath.mulDivUp(binAmountIn, fee, big256.TenPow(18) - fee)
	feeBasis := mulDivUp(
		binAmountIn, fee,
		new(uint256.Int).Sub(big256.TenPow(18), fee),
	)

	// delta.deltaInErc = binAmountIn + feeBasis - line 165 in TypeScript
	delta.DeltaInErc = binAmountIn.Add(binAmountIn, feeBasis)

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
		delta.Excess = big256.U0
	}

	return delta
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
}

func scaleFromAmount(amount *uint256.Int, decimals uint8) *uint256.Int {
	scale := getScale(decimals)
	if scale.CmpUint64(1) == 0 {
		return amount
	}
	return amount.Mul(amount, scale)
}

func ScaleToAmount(amount *uint256.Int, decimals uint8) *uint256.Int {
	scale := getScale(decimals)
	if scale.CmpUint64(1) == 0 || amount.IsZero() {
		return amount
	}
	return amount.Div(amount, scale)
}

func getScale(decimals uint8) *uint256.Int {
	if decimals == 18 {
		return big256.U1
	}
	return big256.TenPow(18 - decimals)
}

func (state *MaverickPoolState) Clone() *MaverickPoolState {
	cloned := &MaverickPoolState{
		FeeAIn:           state.FeeAIn,
		FeeBIn:           state.FeeBIn,
		ProtocolFeeRatio: state.ProtocolFeeRatio,
		TickSpacing:      state.TickSpacing,
		ActiveTick:       state.ActiveTick,
		Bins:             make(map[uint32]Bin, len(state.Bins)),
		Ticks:            make(map[int32]Tick),
		LastTwaD8:        state.LastTwaD8,
		Timestamp:        state.Timestamp,
		BinCounter:       state.BinCounter,
	}

	for k, v := range state.Bins {
		clonedBin := Bin{
			MergeBinBalance: safeCloneUint256(v.MergeBinBalance),
			TotalSupply:     safeCloneUint256(v.TotalSupply),
			TickBalance:     safeCloneUint256(v.TickBalance),
			MergeId:         v.MergeId,
			Kind:            v.Kind,
			Tick:            v.Tick,
		}
		cloned.Bins[k] = clonedBin
	}

	for k, v := range state.Ticks {
		clonedTick := Tick{
			ReserveA:     safeCloneUint256(v.ReserveA),
			ReserveB:     safeCloneUint256(v.ReserveB),
			TotalSupply:  v.TotalSupply,
			BinIdsByTick: v.BinIdsByTick,
		}
		cloned.Ticks[k] = clonedTick
	}

	return cloned
}

func moveBins(state *MaverickPoolState, startingTick, activeTick int32, lastTwapD8, newTwapD8 int64) {
	// Skip if no tick change
	if startingTick == activeTick {
		return
	}

	// Handle upward movement
	newTwap := floorD8Unchecked(newTwapD8 - threshold)
	lastTwap := floorD8Unchecked(lastTwapD8 - threshold)

	if activeTick > startingTick || newTwap > lastTwap {
		// Create moveData equivalent to MoveData in TypeScript
		moveData := &MoveData{
			MergeBinBalance: new(uint256.Int),
			TotalReserveA:   new(uint256.Int),
			TotalReserveB:   new(uint256.Int),
			MergeBins:       make(map[uint32]uint32),
		}

		// Calculate tickLimit as min(activeTick - 1, newTwap)
		moveData.TickLimit = min(activeTick-1, int32(newTwap))

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
	newTwap = floorD8Unchecked(newTwapD8 + threshold)
	lastTwap = floorD8Unchecked(lastTwapD8 + threshold)

	if activeTick < startingTick || newTwap < lastTwap {
		// Create moveData equivalent to MoveData in TypeScript
		moveData := &MoveData{
			MergeBinBalance: new(uint256.Int),
			TotalReserveA:   new(uint256.Int),
			TotalReserveB:   new(uint256.Int),
			MergeBins:       make(map[uint32]uint32),
		}

		// Calculate tickLimit as max(newTwap, activeTick + 1)
		moveData.TickLimit = max(int32(newTwap), activeTick+1)

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
	firstBinTickState := state.Ticks[moveData.FirstBinTick]

	// Merge bins in the list - this modifies firstBinTickState
	mergeBinsInList(state, &firstBin, &firstBinTickState, moveData)

	// Move bin to new tick if needed
	if moveData.TickLimit != moveData.FirstBinTick {
		// Get ending tick state - equivalent to this.state.ticks[moveData.TickLimit.toString()]
		endingTickState := state.Ticks[moveData.TickLimit]
		// Pass the same firstBinTickState that was modified by mergeBinsInList
		moveBinToNewTick(state, &firstBin, &firstBinTickState, &endingTickState, moveData)
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

		// Record this bin using counter as key (matching TypeScript)
		moveData.MergeBins[moveData.Counter] = binId
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
	bin, ok := state.Ticks[tick]
	if !ok {
		return 0
	}

	return bin.BinIdsByTick[kind]
}

// Implementation of mergeBinsInList from TypeScript
func mergeBinsInList(state *MaverickPoolState, firstBin *Bin, firstBinTickState *Tick, moveData *MoveData) {
	var mergeOccurred bool

	for i := range moveData.Counter {
		binId := moveData.MergeBins[i]
		if binId == moveData.FirstBinId {
			continue
		}
		mergeOccurred = true

		binA, binB, mergeBinBalance := mergeAndDecommissionBin(
			state,
			binId,
			moveData.FirstBinId,
			firstBin,
			firstBinTickState,
			moveData.Kind,
		)

		moveData.TotalReserveA = moveData.TotalReserveA.Add(moveData.TotalReserveA, binA)
		moveData.TotalReserveB = moveData.TotalReserveB.Add(moveData.TotalReserveB, binB)
		moveData.MergeBinBalance = moveData.MergeBinBalance.Add(moveData.MergeBinBalance, mergeBinBalance)
	}

	if mergeOccurred {
		maverickBinMathAddLiquidityByReserves(
			firstBin,
			firstBinTickState,
			moveData.TotalReserveA,
			moveData.TotalReserveB,
			moveData.MergeBinBalance,
		)
	}
}

func mergeAndDecommissionBin(
	state *MaverickPoolState,
	binIdToBeMerged uint32,
	parentBinId uint32,
	parentBin *Bin,
	parentBinTick *Tick,
	kind uint8,
) (*uint256.Int, *uint256.Int, *uint256.Int) {
	bin := state.Bins[binIdToBeMerged]
	tick := state.Ticks[bin.Tick]

	binA, binB := binReserves(bin, tick)
	bin.MergeId = parentBinId

	mergeBinBalance := maverickBinMathLpBalancesFromDeltaReserve(
		*parentBin,
		*parentBinTick,
		binA,
		binB,
	)
	bin.MergeBinBalance = mergeBinBalance

	tick.TotalSupply = clip(tick.TotalSupply, bin.TickBalance)
	tick.ReserveA = clip(tick.ReserveA, binA)
	tick.ReserveB = clip(tick.ReserveB, binB)
	delete(tick.BinIdsByTick, kind)

	// Update state
	state.Bins[binIdToBeMerged] = bin
	state.Ticks[bin.Tick] = tick

	return binA, binB, mergeBinBalance
}

// MaverickBinMath.lpBalancesFromDeltaReserve equivalent
func maverickBinMathLpBalancesFromDeltaReserve(
	self Bin,
	tick Tick,
	deltaA *uint256.Int,
	deltaB *uint256.Int,
) *uint256.Int {
	if tick.ReserveA.Cmp(tick.ReserveB) >= 0 {
		reserveA := mulDivUp(
			tick.ReserveA,
			self.TickBalance,
			tick.TotalSupply,
		)
		return mulDivDown(
			deltaA,
			big256.Max(big256.U1, self.TotalSupply),
			reserveA,
		)
	} else {
		reserveB := mulDivUp(
			tick.ReserveB,
			self.TickBalance,
			tick.TotalSupply,
		)
		return mulDivDown(
			deltaB,
			big256.Max(big256.U1, self.TotalSupply),
			reserveB,
		)
	}
}

// MaverickBinMath.addLiquidityByReserves equivalent
func maverickBinMathAddLiquidityByReserves(
	self *Bin,
	tick *Tick,
	deltaA *uint256.Int,
	deltaB *uint256.Int,
	deltaLpBalance *uint256.Int,
) {
	deltaTickBalance := mulDivDown(
		deltaLpBalance,
		big256.Max(big256.U1, self.TickBalance),
		self.TotalSupply,
	)

	maverickBinMathUpdateBinState(self, tick, deltaA, deltaB, deltaLpBalance, deltaTickBalance)
}

func maverickBinMathUpdateBinState(
	self *Bin,
	tick *Tick,
	deltaA *uint256.Int,
	deltaB *uint256.Int,
	deltaLpBalance *uint256.Int,
	deltaTickBalance *uint256.Int,
) {
	totalSupply := self.TotalSupply
	if totalSupply.IsZero() {
		minimumLiquidity := big256.TenPow(8)
		if deltaLpBalance.Cmp(minimumLiquidity) < 0 {
			panic("insufficient liquidity")
		}
		totalSupply.Set(minimumLiquidity)
	}

	self.TotalSupply = totalSupply.Add(totalSupply, deltaLpBalance)
	tick.TotalSupply = tick.TotalSupply.Add(tick.TotalSupply, deltaTickBalance)
	self.TickBalance = self.TickBalance.Add(self.TickBalance, deltaTickBalance)
	tick.ReserveA = tick.ReserveA.Add(tick.ReserveA, deltaA)
	tick.ReserveB = tick.ReserveB.Add(tick.ReserveB, deltaB)
}

// Implementation of moveBinToNewTick from TypeScript - exact mapping
func moveBinToNewTick(state *MaverickPoolState, firstBin *Bin, startingTickState *Tick, endingTickState *Tick,
	moveData *MoveData) {
	firstBinA, firstBinB := binReserves(*firstBin, *startingTickState)

	startingTickState.ReserveA = clip(startingTickState.ReserveA, firstBinA)
	startingTickState.ReserveB = clip(startingTickState.ReserveB, firstBinB)
	startingTickState.TotalSupply = clip(startingTickState.TotalSupply, firstBin.TickBalance)
	startingTickState.BinIdsByTick[moveData.Kind] = 0

	if state.Ticks[firstBin.Tick].TotalSupply.IsZero() {
		delete(state.Ticks, firstBin.Tick)
	}

	endingTickState.BinIdsByTick[moveData.Kind] = moveData.FirstBinId
	firstBin.Tick = moveData.TickLimit

	var deltaTickBalance *uint256.Int
	if firstBinA.Cmp(firstBinB) > 0 {
		deltaTickBalance = mulDivDown(
			firstBinA,
			big256.Max(big256.U1, endingTickState.TotalSupply),
			endingTickState.ReserveA,
		)
	} else {
		deltaTickBalance = mulDivDown(
			firstBinB,
			big256.Max(big256.U1, endingTickState.TotalSupply),
			endingTickState.ReserveB,
		)
	}

	endingTickState.ReserveA = new(uint256.Int).Add(endingTickState.ReserveA, firstBinA)
	endingTickState.ReserveB = new(uint256.Int).Add(endingTickState.ReserveB, firstBinB)
	firstBin.TickBalance = deltaTickBalance
	endingTickState.TotalSupply = new(uint256.Int).Add(endingTickState.TotalSupply, deltaTickBalance)

	// Update state
	state.Bins[moveData.FirstBinId] = *firstBin
	state.Ticks[moveData.TickLimit] = *endingTickState
}

// Helper function equivalent to MaverickBasicMath.clip - safe subtraction
func clip(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) >= 0 {
		return new(uint256.Int).Sub(a, b)
	}
	return big256.U0
}

// Helper function equivalent to MaverickBasicMath.mulDivDown
func mulDivDown(a, b, denominator *uint256.Int) *uint256.Int {
	rslt, _ := new(uint256.Int).MulDivOverflow(a, b, denominator)
	return rslt
}

// Helper function equivalent to MaverickBasicMath.mulDivUp
func mulDivUp(a, b, denominator *uint256.Int) *uint256.Int {
	rslt, _ := v3Utils.MulDivRoundingUp(a, b, denominator)
	return rslt
}

// Helper function equivalent to MaverickBasicMath.divUp
func divUp(a, b *uint256.Int) *uint256.Int {
	return mulDivUp(a, big256.TenPow(18), b)
}

// Helper function equivalent to MaverickBasicMath.invFloor
func invFloor(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return big256.U0
	}
	// invFloor(x) = big256.TenPow(36) / x (with floor division)
	return new(uint256.Int).Div(big256.TenPow(36), x)
}

// Helper function equivalent to MaverickBasicMath.invCeil
func invCeil(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return big256.U0
	}
	// invCeil(x) = (big256.TenPow(36) + x - 1) / x (ceiling division)
	numerator := new(uint256.Int).Add(big256.TenPow(36), x)
	return numerator.SubUint64(numerator, 1).Div(numerator, x)
}

// Helper function equivalent to MaverickBasicMath.mulDown
func mulDown(a, b *uint256.Int) *uint256.Int {
	return mulDivDown(a, b, big256.TenPow(18))
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
		denominator := outOverL.Sub(invSqrtPrice, outOverL)

		if denominator.IsZero() {
			return big256.U0
		}

		result := mulDivUp(output, sqrtPrice, denominator)
		return result
	} else {
		// return MaverickBasicMath.divUp(output, MaverickBasicMath.mulDown(sqrtPrice, sqrtPrice - outOverL))
		numerator := outOverL.Sub(sqrtPrice, outOverL)
		denominator := mulDown(sqrtPrice, numerator)

		if denominator.IsZero() {
			return big256.U0
		}

		result := divUp(output, denominator)
		return result
	}
}

// amountToBinNetOfProtocolFee calculates amount to bin after protocol fee, matching TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-swap-math.ts#L8
func amountToBinNetOfProtocolFee(deltaInErc, feeBasis *uint256.Int, protocolFeeD3 uint8) *uint256.Int {
	if protocolFeeD3 != 0 {
		protocolFee := mulDivUp(feeBasis, new(uint256.Int).SetUint64(uint64(protocolFeeD3)), big256.TenPow(3))
		return clip(deltaInErc, protocolFee)
	}
	return deltaInErc
}

// Helper function equivalent to MaverickPoolLib.binReserves
func binReserves(bin Bin, tick Tick) (*uint256.Int, *uint256.Int) {
	return binReservesCalc(
		bin.TickBalance,
		tick.ReserveA,
		tick.ReserveB,
		tick.TotalSupply,
	)
}

func binReservesCalc(
	tickBalance *uint256.Int,
	tickReserveA *uint256.Int,
	tickReserveB *uint256.Int,
	tickTotalSupply *uint256.Int,
) (*uint256.Int, *uint256.Int) {
	if !tickTotalSupply.IsZero() {
		return reserveValue(tickReserveA, tickBalance, tickTotalSupply),
			reserveValue(tickReserveB, tickBalance, tickTotalSupply)
	}
	return big256.U0, big256.U0
}

func reserveValue(
	tickReserve *uint256.Int,
	tickBalance *uint256.Int,
	tickTotalSupply *uint256.Int,
) *uint256.Int {
	reserve := mulDivDown(tickReserve, tickBalance, tickTotalSupply)
	return big256.Min(tickReserve, reserve)
}

func getTickDataWithZeroLiquidity(state *MaverickPoolState, tick int32) *TickData {
	tickState, ok := state.Ticks[tick]
	if !ok {
		return &TickData{
			CurrentReserveA:  big256.U0,
			CurrentReserveB:  big256.U0,
			CurrentLiquidity: big256.U0,
		}
	}
	return &TickData{
		CurrentReserveA:  tickState.ReserveA,
		CurrentReserveB:  tickState.ReserveB,
		CurrentLiquidity: big256.U0,
	}
}

// Helper to safely clone uint256.Int (handles nil values)
func safeCloneUint256(value *uint256.Int) *uint256.Int {
	if value == nil {
		return nil
	}
	return new(uint256.Int).Set(value)
}

// Square root price and tick calculations matching the TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L122
func calculateSqrtPrice(tickSpacing uint32, tick int32) *uint256.Int {
	sqrtP, _ := maverickv1.TickPrice(int32(tickSpacing), tick)
	return sqrtP
}

// getTickSqrtPriceAndL calculates both sqrt price and liquidity for a tick, matching TypeScript implementation
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L8
func getTickSqrtPriceAndL(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice *uint256.Int) (*uint256.Int,
	*uint256.Int) {
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
	var tmp uint256.Int

	// Check if reserves are small (< 2^78) and apply precision bump
	if tmp.Rsh(reserveA, 78).IsZero() && tmp.Rsh(reserveB, 78).IsZero() {
		precisionBump = 57
		reserveA = new(uint256.Int).Lsh(reserveA, 57)
		reserveB = new(uint256.Int).Lsh(reserveB, 57)
	}

	// Calculate diff = sqrtUpperTickPrice - sqrtLowerTickPrice
	diff := tmp.Sub(sqrtUpperTickPrice, sqrtLowerTickPrice)
	// Calculate b = divDown(reserveA, sqrtUpperTickPrice) + mulDown(reserveB, sqrtLowerTickPrice)
	term1 := mulDivDown(reserveA, big256.TenPow(18), sqrtUpperTickPrice)
	term2 := mulDivDown(reserveB, sqrtLowerTickPrice, big256.TenPow(18))
	b := term1.Add(term1, term2)

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

	// Calculate complex liquidity formula exactly matching TypeScript:
	// MaverickBasicMath.mulDiv(
	//   b + MaverickBasicMath.sqrt(
	//     MaverickBasicMath.mulDiv(b, b, big256.TenPow(18)) +
	//     MaverickBasicMath.mulDiv(
	//       MaverickBasicMath.mulFloor(reserveB, reserveA),
	//       diff,
	//       sqrtUpperTickPrice,
	//     ),
	//   ) * big256.TenPow(9),
	//   sqrtUpperTickPrice,
	//   diff,
	// )

	// Step 1: MaverickBasicMath.mulDiv(b, b, big256.TenPow(18))
	bSquared := mulDivDown(b, b, big256.TenPow(18))

	// Step 2: MaverickBasicMath.mulFloor(reserveB, reserveA) = mulDivDown(reserveB, reserveA, big256.TenPow(18))
	reserveProduct := mulDivDown(reserveB, reserveA, big256.TenPow(18))

	// Step 3: MaverickBasicMath.mulDiv(reserveProduct, diff, sqrtUpperTickPrice)
	reserveProductDiff := mulDivDown(reserveProduct, diff, sqrtUpperTickPrice)

	// Step 4: bSquared + reserveProductDiff
	bSquaredPlusReserveProduct := bSquared.Add(bSquared, reserveProductDiff)

	// Step 5: MaverickBasicMath.sqrt(...)
	sqrtTerm := bSquaredPlusReserveProduct.Sqrt(bSquaredPlusReserveProduct)

	// Step 6: sqrtTerm * big256.TenPow(9) - multiplication has higher precedence than addition!
	sqrtTermTimesPow9 := sqrtTerm.Mul(sqrtTerm, big256.TenPow(9))

	// Step 7: b + (sqrtTerm * big256.TenPow(9))
	bPlusSqrtTermTimesPow9 := sqrtTermTimesPow9.Add(b, sqrtTermTimesPow9)

	// Step 8: MaverickBasicMath.mulDiv(bPlusSqrtTermTimesPow9, sqrtUpperTickPrice, diff)
	result := mulDivDown(bPlusSqrtTermTimesPow9, sqrtUpperTickPrice, diff)

	if precisionBump > 0 {
		result.Rsh(result, precisionBump)
	}

	return result
}

// getSqrtPrice calculates the sqrt price based on reserves and liquidity (internal helper)
// ref: https://github.com/VeloraDEX/paraswap-dex-lib/blob/86f630d54658926d606a08b11e0206062886c57d/src/dex/maverick-v2/maverick-math/maverick-tick-math.ts#L32
func getSqrtPrice(reserveA, reserveB, sqrtLowerTickPrice, sqrtUpperTickPrice, liquidity *uint256.Int) *uint256.Int {
	if reserveA.IsZero() {
		return sqrtLowerTickPrice
	} else if reserveB.IsZero() {
		return sqrtUpperTickPrice
	}

	// Calculate sqrtPrice = sqrt(big256.TenPow(18) * divDown(
	//   reserveA + mulDown(liquidity, sqrtLowerTickPrice),
	//   reserveB + divDown(liquidity, sqrtUpperTickPrice)
	// ))
	//
	// Note: divDown(x, y) = mulDivDown(x, big256.TenPow(18), y) in TypeScript
	// So the calculation is: sqrt(big256.TenPow(18) * mulDivDown(numerator, big256.TenPow(18), denominator))
	// Which simplifies to: sqrt(mulDivDown(numerator, big256.TenPow(18)^2, denominator))

	// Numerator: reserveA + mulDown(liquidity, sqrtLowerTickPrice)
	liquidityTermA := mulDivDown(liquidity, sqrtLowerTickPrice, big256.TenPow(18))
	numerator := liquidityTermA.Add(reserveA, liquidityTermA)

	// Denominator: reserveB + divDown(liquidity, sqrtUpperTickPrice)
	liquidityTermB := mulDivDown(liquidity, big256.TenPow(18), sqrtUpperTickPrice)
	denominator := liquidityTermB.Add(reserveB, liquidityTermB)

	// Calculate ratio with big256.TenPow(18)^2 scaling to match TypeScript
	ratio := mulDivDown(numerator, big256.TenPow(36), denominator)
	sqrtPrice := ratio.Sqrt(ratio)

	// Ensure the price is within bounds: min(max(sqrtPrice, sqrtLowerTickPrice), sqrtUpperTickPrice)
	if sqrtPrice.Cmp(sqrtLowerTickPrice) < 0 {
		return sqrtLowerTickPrice
	} else if sqrtPrice.Cmp(sqrtUpperTickPrice) > 0 {
		return sqrtUpperTickPrice
	}

	return sqrtPrice
}

func tickSqrtPriceAndLiquidity(state *MaverickPoolState, tick int32) (lower, upper, sqrtP *uint256.Int, data TickData) {
	// Get the consolidated bin data (equivalent to this.state.ticks[tick.toString()])
	tickData := getTickDataWithZeroLiquidity(state, tick)
	// Calculate the square root prices at the tick boundaries using tickSqrtPrices
	sqrtLowerTickPrice := calculateSqrtPrice(state.TickSpacing, tick)
	sqrtUpperTickPrice := calculateSqrtPrice(state.TickSpacing, tick+1)

	// Calculate sqrt price and liquidity using the combined TypeScript logic
	sqrtPrice, currentLiquidity := getTickSqrtPriceAndL(tickData.CurrentReserveA, tickData.CurrentReserveB,
		sqrtLowerTickPrice, sqrtUpperTickPrice)

	// Set the calculated liquidity in the tickData
	tickData.CurrentLiquidity = currentLiquidity

	return sqrtLowerTickPrice, sqrtUpperTickPrice, sqrtPrice, *tickData
}

func allocateSwapValuesToTick(state *MaverickPoolState, delta *Delta, tokenAIn bool, tick int32) {
	tickState, ok := state.Ticks[tick]
	if !ok {
		return
	}

	reserveA := tickState.ReserveA
	reserveB := tickState.ReserveB

	if tokenAIn {
		reserveA.Add(reserveA, delta.DeltaInBinInternal)
		if delta.Excess.Sign() > 0 {
			reserveB.Set(big256.U0)
		} else {
			reserveB.Set(clip(reserveB, delta.DeltaOutErc))
		}
	} else {
		if delta.Excess.Sign() > 0 {
			reserveA.Set(big256.U0)
		} else {
			reserveA.Set(clip(reserveA, delta.DeltaOutErc))
		}
		reserveB.Add(reserveB, delta.DeltaInBinInternal)
	}

	state.Ticks[tick].ReserveA.Set(reserveA)
	state.Ticks[tick].ReserveB.Set(reserveB)
}
