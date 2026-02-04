package pools

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math/twamm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

type (
	TwammPoolSwapState struct {
		*FullRangePoolSwapState
		Token0SaleRate    *uint256.Int `json:"token0SaleRate"`
		Token1SaleRate    *uint256.Int `json:"token1SaleRate"`
		LastExecutionTime uint64       `json:"lastExecutionTime"`
	}

	TwammPoolState struct {
		*FullRangePoolState
		*TimedPoolState
	}

	TwammPool struct {
		*FullRangePool
		token0SaleRate     *uint256.Int
		token1SaleRate     *uint256.Int
		lastExecutionTime  uint64
		virtualOrderDeltas []TimeRateDelta
	}

	TwammOrderKeyAbi = struct {
		Token0 common.Address `json:"token0"`
		Token1 common.Address `json:"token1"`
		Config [32]byte       `json:"config"`
	}

	TwammOrderKey struct {
		TwammOrderKeyAbi
	}
)

func (p *TwammPool) GetState() any {
	return NewTwammPoolState(
		p.FullRangePoolState,
		NewTimedPoolState(NewTimedPoolSwapState(p.token0SaleRate, p.token1SaleRate, p.lastExecutionTime), p.virtualOrderDeltas),
	)
}

func (p *TwammPool) CloneState() any {
	cloned := *p
	cloned.FullRangePool = p.FullRangePool.CloneState().(*FullRangePool)
	cloned.token0SaleRate = p.token0SaleRate.Clone()
	cloned.token1SaleRate = p.token1SaleRate.Clone()
	return &cloned
}

func (p *TwammPool) SetSwapState(state quoting.SwapState) {
	twammState := state.(*TwammPoolSwapState)

	p.FullRangePoolSwapState = twammState.FullRangePoolSwapState
	p.token0SaleRate = twammState.Token0SaleRate
	p.token1SaleRate = twammState.Token1SaleRate
	p.lastExecutionTime = twammState.LastExecutionTime
}

func (p *TwammPool) quoteWithTimestampFn(amount *uint256.Int, isToken1 bool,
	estimateTimestampFn func() uint64) (*quoting.Quote, error) {
	currentTime := max(estimateTimestampFn(), p.lastExecutionTime)

	nextSqrtRatio := p.SqrtRatio
	var token0SaleRate, token1SaleRate, tmp, tmp2 uint256.Int
	token0SaleRate.Set(p.token0SaleRate)
	token1SaleRate.Set(p.token1SaleRate)
	lastExecutionTime := p.lastExecutionTime

	var virtualOrderDeltaTimesCrossed int64
	nextSaleRateDeltaIndex := slices.IndexFunc(p.virtualOrderDeltas, func(srd TimeRateDelta) bool {
		return srd.Time > lastExecutionTime
	})
	if nextSaleRateDeltaIndex == -1 {
		nextSaleRateDeltaIndex = math.MaxInt
	}

	var fullRangePoolSwapStateOverride *FullRangePoolSwapState

	for lastExecutionTime != currentTime {
		var saleRateDelta *TimeRateDelta
		nextExecutionTime := currentTime

		if nextSaleRateDeltaIndex < len(p.virtualOrderDeltas) {
			saleRateDelta = &p.virtualOrderDeltas[nextSaleRateDeltaIndex]
			if nextExecutionTime > saleRateDelta.Time {
				nextExecutionTime = saleRateDelta.Time
			}
		}

		timeElapsed := nextExecutionTime - lastExecutionTime
		if timeElapsed > uint64(math.MaxUint32) {
			return nil, errors.New("too much time passed since last execution")
		}

		timeElapsedBig := tmp.SetUint64(timeElapsed)

		amount0 := tmp2.Rsh(tmp2.Mul(&token0SaleRate, timeElapsedBig), 32)
		amount1 := tmp.Rsh(tmp.Mul(&token1SaleRate, timeElapsedBig), 32)

		if amount0.Sign() > 0 && amount1.Sign() > 0 {
			currentSqrtRatio := nextSqrtRatio
			if currentSqrtRatio.Lt(ekubomath.MinSqrtRatio) {
				currentSqrtRatio = ekubomath.MinSqrtRatio
			} else if currentSqrtRatio.Gt(ekubomath.MaxSqrtRatio) {
				currentSqrtRatio = ekubomath.MaxSqrtRatio
			}

			nextSqrtRatio = twamm.CalculateNextSqrtRatio(
				currentSqrtRatio,
				p.Liquidity,
				&token0SaleRate,
				&token1SaleRate,
				uint32(timeElapsed),
				p.GetKey().Fee(),
			)

			isToken1 := currentSqrtRatio.Lt(nextSqrtRatio)
			amount := lo.Ternary(isToken1, amount1, amount0)
			quote, err := p.quoteWithLimitAndOverride(amount, isToken1, nextSqrtRatio, fullRangePoolSwapStateOverride)
			if err != nil {
				return nil, fmt.Errorf("virtual order full range pool quote: %w", err)
			}

			fullRangePoolSwapStateOverride = quote.SwapInfo.SwapStateAfter.(*FullRangePoolSwapState)
		} else if amount0.Sign() > 0 || amount1.Sign() > 0 {
			isToken1 := amount0.IsZero()
			amount := lo.Ternary(isToken1, amount1, amount0)
			quote, err := p.quoteWithLimitAndOverride(amount, isToken1, nil, fullRangePoolSwapStateOverride)
			if err != nil {
				return nil, fmt.Errorf("virtual order full range pool quote: %w", err)
			}

			fullRangePoolSwapStateOverride = quote.SwapInfo.SwapStateAfter.(*FullRangePoolSwapState)
			nextSqrtRatio = fullRangePoolSwapStateOverride.SqrtRatio
		}

		if saleRateDelta != nil && saleRateDelta.Time == nextExecutionTime {
			token0SaleRate.Add(&token0SaleRate, (*uint256.Int)(saleRateDelta.Delta0))
			token1SaleRate.Add(&token1SaleRate, (*uint256.Int)(saleRateDelta.Delta1))

			nextSaleRateDeltaIndex++
			virtualOrderDeltaTimesCrossed++
		}

		lastExecutionTime = nextExecutionTime
	}

	finalQuote, err := p.quoteWithLimitAndOverride(amount, isToken1, nil, fullRangePoolSwapStateOverride)
	if err != nil {
		return nil, fmt.Errorf("final full range pool quote: %w", err)
	}

	var virtualOrdersExecuted int64
	if currentTime > p.lastExecutionTime {
		virtualOrdersExecuted = 1
	}

	return &quoting.Quote{
		ConsumedAmount:   finalQuote.ConsumedAmount,
		CalculatedAmount: finalQuote.CalculatedAmount,
		FeesPaid:         finalQuote.FeesPaid,
		Gas:              finalQuote.Gas + virtualOrderDeltaTimesCrossed*quoting.GasVirtualOrderDelta + virtualOrdersExecuted*quoting.GasExecutingVirtualOrders,
		SwapInfo: quoting.SwapInfo{
			SkipAhead: 0,
			IsToken1:  isToken1,
			SwapStateAfter: NewTwammPoolSwapState(
				finalQuote.SwapInfo.SwapStateAfter.(*FullRangePoolSwapState),
				&token0SaleRate,
				&token1SaleRate,
				currentTime,
			),
			TickSpacingsCrossed: 0,
		},
	}, nil
}

func (p *TwammPool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteWithTimestampFn(amount, isToken1, estimatedBlockTimestamp)
}

func (k *TwammOrderKey) Fee() uint64 {
	return binary.BigEndian.Uint64(k.Config[:])
}

func (k *TwammOrderKey) SellsToken1() bool {
	return k.Config[8] != 0
}

func (k *TwammOrderKey) StartTime() uint64 {
	return binary.BigEndian.Uint64(k.Config[16:])
}

func (k *TwammOrderKey) EndTime() uint64 {
	return binary.BigEndian.Uint64(k.Config[24:])
}

func NewTwammPoolSwapState(fullRangeSwapState *FullRangePoolSwapState, token0SaleRate, token1SaleRate *uint256.Int, lastExecutionTime uint64) *TwammPoolSwapState {
	return &TwammPoolSwapState{
		FullRangePoolSwapState: fullRangeSwapState,
		Token0SaleRate:         token0SaleRate,
		Token1SaleRate:         token1SaleRate,
		LastExecutionTime:      lastExecutionTime,
	}
}

func NewTwammPoolState(fullRangeState *FullRangePoolState, timedState *TimedPoolState) *TwammPoolState {
	return &TwammPoolState{
		FullRangePoolState: fullRangeState,
		TimedPoolState:     timedState,
	}
}

func NewTwammPool(key *FullRangePoolKey, state *TwammPoolState) *TwammPool {
	return &TwammPool{
		FullRangePool:      NewFullRangePool(key, state.FullRangePoolState),
		token0SaleRate:     state.Token0Rate,
		token1SaleRate:     state.Token1Rate,
		lastExecutionTime:  state.LastExecutionTime,
		virtualOrderDeltas: state.VirtualDeltas,
	}
}
