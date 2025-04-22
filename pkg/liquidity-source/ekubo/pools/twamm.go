package pools

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"time"

	ekubo_math "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math/twamm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type TwammPoolSwapState struct {
	*FullRangePoolSwapState
	Token0SaleRate    *big.Int `json:"token0SaleRate"`
	Token1SaleRate    *big.Int `json:"token1SaleRate"`
	LastExecutionTime uint64   `json:"lastExecutionTime"`
}

type TwammPoolState struct {
	*FullRangePoolState
	Token0SaleRate     *big.Int             `json:"token0SaleRate"`
	Token1SaleRate     *big.Int             `json:"token1SaleRate"`
	LastExecutionTime  uint64               `json:"lastExecutionTime"`
	VirtualOrderDeltas []TwammSaleRateDelta `json:"virtualOrderDeltas"`
}

type TwammSaleRateDelta struct {
	Time           uint64   `json:"time"`
	SaleRateDelta0 *big.Int `json:"saleRateDelta0"`
	SaleRateDelta1 *big.Int `json:"saleRateDelta1"`
}

type TwammPool struct {
	*FullRangePool
	token0SaleRate     *big.Int
	token1SaleRate     *big.Int
	lastExecutionTime  uint64
	virtualOrderDeltas []TwammSaleRateDelta
}

func NewTwammPool(key *PoolKey, state *TwammPoolState) *TwammPool {
	return &TwammPool{
		FullRangePool:      NewFullRangePool(key, state.FullRangePoolState),
		token0SaleRate:     state.Token0SaleRate,
		token1SaleRate:     state.Token1SaleRate,
		lastExecutionTime:  state.LastExecutionTime,
		virtualOrderDeltas: state.VirtualOrderDeltas,
	}
}

func (p *TwammPool) GetState() any {
	return &TwammPoolState{
		FullRangePoolState: p.FullRangePoolState,
		Token0SaleRate:     p.token0SaleRate,
		Token1SaleRate:     p.token1SaleRate,
		LastExecutionTime:  p.lastExecutionTime,
		VirtualOrderDeltas: p.virtualOrderDeltas,
	}
}

func (p *TwammPool) SetSwapState(state any) {
	twammState := state.(*TwammPoolSwapState)

	p.FullRangePoolSwapState = twammState.FullRangePoolSwapState
	p.token0SaleRate = twammState.Token0SaleRate
	p.token1SaleRate = twammState.Token1SaleRate
	p.lastExecutionTime = twammState.LastExecutionTime
}

func (p *TwammPool) quoteWithTimestampFn(amount *big.Int, isToken1 bool, estimateTimestampFn func() uint64) (*quoting.Quote, error) {
	currentTime := max(estimateTimestampFn(), p.lastExecutionTime)

	nextSqrtRatio := p.SqrtRatio
	token0SaleRate, token1SaleRate := new(big.Int).Set(p.token0SaleRate), new(big.Int).Set(p.token1SaleRate)
	lastExecutionTime := p.lastExecutionTime

	virtualOrderDeltaTimesCrossed := int64(0)
	nextSaleRateDeltaIndex := slices.IndexFunc(p.virtualOrderDeltas, func(srd TwammSaleRateDelta) bool {
		return srd.Time > lastExecutionTime
	})
	if nextSaleRateDeltaIndex == -1 {
		nextSaleRateDeltaIndex = math.MaxInt
	}

	var fullRangePoolSwapStateOverride *FullRangePoolSwapState

	for lastExecutionTime != currentTime {
		var saleRateDelta *TwammSaleRateDelta
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

		timeElapsedBig := new(big.Int).SetUint64(timeElapsed)

		amount0 := new(big.Int).Rsh(new(big.Int).Mul(token0SaleRate, timeElapsedBig), 32)
		amount1 := new(big.Int).Rsh(new(big.Int).Mul(token1SaleRate, timeElapsedBig), 32)

		if amount0.Sign() == 1 && amount1.Sign() == 1 {
			currentSqrtRatio := nextSqrtRatio
			if currentSqrtRatio.Cmp(ekubo_math.MinSqrtRatio) == -1 {
				currentSqrtRatio = ekubo_math.MinSqrtRatio
			} else if currentSqrtRatio.Cmp(ekubo_math.MaxSqrtRatio) == 1 {
				currentSqrtRatio = ekubo_math.MaxSqrtRatio
			}

			nextSqrtRatio = twamm.CalculateNextSqrtRatio(
				currentSqrtRatio,
				p.Liquidity,
				token0SaleRate,
				token1SaleRate,
				uint32(timeElapsed),
				p.GetKey().Config.Fee,
			)

			var (
				amount   *big.Int
				isToken1 bool
			)
			if currentSqrtRatio.Cmp(nextSqrtRatio) == -1 {
				amount, isToken1 = amount1, true
			} else {
				amount, isToken1 = amount0, false
			}

			quote, err := p.
				FullRangePool.
				quoteWithLimitAndOverride(amount, isToken1, nextSqrtRatio, fullRangePoolSwapStateOverride)
			if err != nil {
				return nil, fmt.Errorf("virtual order full range pool quote: %w", err)
			}

			{
				temp := quote.SwapInfo.SwapStateAfter.(FullRangePoolSwapState)
				fullRangePoolSwapStateOverride = &temp
			}
		} else if amount0.Sign() == 1 || amount1.Sign() == 1 {
			var (
				amount   *big.Int
				isToken1 bool
			)
			if amount0.Sign() != 0 {
				amount, isToken1 = amount0, false
			} else {
				amount, isToken1 = amount1, true
			}

			quote, err := p.
				FullRangePool.
				quoteWithLimitAndOverride(amount, isToken1, nil, fullRangePoolSwapStateOverride)
			if err != nil {
				return nil, fmt.Errorf("virtual order full range pool quote: %w", err)
			}

			{
				temp := quote.SwapInfo.SwapStateAfter.(FullRangePoolSwapState)
				fullRangePoolSwapStateOverride = &temp
			}

			nextSqrtRatio = fullRangePoolSwapStateOverride.SqrtRatio
		}

		if saleRateDelta != nil {
			if saleRateDelta.Time == nextExecutionTime {
				token0SaleRate.Add(token0SaleRate, saleRateDelta.SaleRateDelta0)
				token1SaleRate.Add(token1SaleRate, saleRateDelta.SaleRateDelta1)

				nextSaleRateDeltaIndex++
				virtualOrderDeltaTimesCrossed++
			}
		}

		lastExecutionTime = nextExecutionTime
	}

	finalQuote, err := p.
		FullRangePool.
		quoteWithLimitAndOverride(amount, isToken1, nil, fullRangePoolSwapStateOverride)
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
		Gas:              finalQuote.Gas + virtualOrderDeltaTimesCrossed*quoting.GasCostOfOneVirtualOrderDelta + virtualOrdersExecuted*quoting.GasCostOfExecutingVirtualOrders,
	}, nil
}

const slotDuration = 12

func (p *TwammPool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteWithTimestampFn(amount, isToken1, func() uint64 {
		return uint64(time.Now().Unix()) + slotDuration
	})
}
