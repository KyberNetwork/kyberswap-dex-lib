package maverickv2

import (
	"fmt"
	"math/big"
	"strings"

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
	if tokenAIn {
		scaleAmount, err = scaleFromAmount(amountIn, p.decimals[0])
	} else {
		scaleAmount, err = scaleFromAmount(amountIn, p.decimals[1])
	}
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	newState := p.state.Clone()
	_, amountOut, binCrossed, err := swap(newState, scaleAmount, tokenAIn, false, false)
	if err != nil {
		return nil, fmt.Errorf("can not get amount out, err: %v", err)
	}

	var scaleAmountOut *uint256.Int
	if tokenAIn {
		scaleAmountOut, err = ScaleToAmount(amountOut, p.decimals[1])
	} else {
		scaleAmountOut, err = ScaleToAmount(amountOut, p.decimals[0])
	}
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
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
			activeTick: newState.ActiveTick,
			bins:       newState.Bins,
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
	tokenAIn := strings.EqualFold(tokenIn, p.Pool.Info.Tokens[0])

	var scaleAmount *uint256.Int
	var err error
	if tokenAIn {
		scaleAmount, err = ScaleToAmount(amountOut, p.decimals[1])
	} else {
		scaleAmount, err = ScaleToAmount(amountOut, p.decimals[0])
	}
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	newState := p.state.Clone()
	amountIn, _, binCrossed, err := swap(newState, scaleAmount, tokenAIn, true, false)
	if err != nil {
		return nil, fmt.Errorf("swap failed, err: %v", err)
	}

	var scaleAmountIn *uint256.Int
	if tokenAIn {
		scaleAmountIn, err = scaleFromAmount(amountIn, p.decimals[0])
	} else {
		scaleAmountIn, err = scaleFromAmount(amountIn, p.decimals[1])
	}
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: scaleAmountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: GasSwap + GasCrossBin*int64(binCrossed),
		SwapInfo: maverickSwapInfo{
			activeTick: newState.ActiveTick,
			bins:       newState.Bins,
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

	p.state.Bins = newState.bins
	p.state.ActiveTick = newState.activeTick
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

type maverickSwapInfo struct {
	activeTick int32
	bins       map[uint32]Bin
}

// Helper functions for swap implementation
func swap(state *MaverickPoolState, amount *uint256.Int, tokenAIn bool, exactOutput bool, bypassLimit bool) (*uint256.Int, *uint256.Int, uint32, error) {
	// Implementation based on maverick-v2-pool-math.ts estimateSwap function

	delta := &Delta{
		DeltaInBinInternal: new(uint256.Int),
		DeltaInErc:         new(uint256.Int),
		DeltaOutErc:        new(uint256.Int),
		Excess:             new(uint256.Int).Set(amount),
		TokenAIn:           tokenAIn,
		ExactOutput:        exactOutput,
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
		return nil, nil, 0, fmt.Errorf("beyond swap limit")
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

		// Break if we've reached the maximum iterations to avoid infinite loops
		if binCrossed > MaxSwapCalcIter {
			break
		}
	}

	return delta.DeltaInErc, delta.DeltaOutErc, binCrossed, nil
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
	}

	activeTick := state.ActiveTick

	// Check if we've reached the tick limit
	if (activeTick > tickLimit && delta.TokenAIn) || (activeTick < tickLimit && !delta.TokenAIn) {
		state.ActiveTick += boolToInt32(!delta.TokenAIn) - boolToInt32(delta.TokenAIn)
		return delta, true, nil
	}

	// Find next tick with liquidity
	crossedBin := false
	ticksSearched := 0
	for {
		tickData, ok := getTickData(state, activeTick)

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
			return delta, true, nil
		}

		// Safety check to avoid infinite loops
		if ticksSearched > 1000 {
			return nil, false, fmt.Errorf("too many ticks searched without finding liquidity")
		}
	}

	state.ActiveTick = activeTick

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
	self.DeltaInBinInternal = new(uint256.Int).Add(self.DeltaInBinInternal, delta.DeltaInBinInternal)
	self.DeltaInErc = new(uint256.Int).Add(self.DeltaInErc, delta.DeltaInErc)
	self.DeltaOutErc = new(uint256.Int).Add(self.DeltaOutErc, delta.DeltaOutErc)
	self.Excess = new(uint256.Int).Set(delta.Excess)
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
}

type Extra struct {
	FeeAIn           uint32             `json:"feeAIn"`
	FeeBIn           uint32             `json:"feeBIn"`
	ProtocolFeeRatio uint8              `json:"protocolFeeRatio"`
	Bins             map[uint32]Bin     `json:"bins"`
	BinPositions     map[int32][]uint32 `json:"binPositions"`
	BinMap           map[int32]uint32   `json:"binMap"`
	ActiveTick       int32              `json:"activeTick"`
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
