package pools

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

type (
	BoostedFeesPoolSwapState struct {
		*BasePoolSwapState
		*TimedPoolSwapState
	}

	BoostedFeesPoolState struct {
		*BasePoolState
		*TimedPoolState
	}

	BoostedFeesPool struct {
		*BasePool
		donateRate0      *uint256.Int
		donateRate1      *uint256.Int
		lastDonateTime   uint64
		donateRateDeltas []TimeRateDelta
	}
)

func (p *BoostedFeesPool) GetState() any {
	return NewBoostedFeesPoolState(
		p.BasePoolState,
		NewTimedPoolState(NewTimedPoolSwapState(p.donateRate0, p.donateRate1, p.lastDonateTime), p.donateRateDeltas),
	)
}

func (p *BoostedFeesPool) quoteWithTimestampFn(amount *uint256.Int, isToken1 bool,
	estimateTimestampFn func() uint64) (*quoting.Quote, error) {
	currentTime := max(estimateTimestampFn(), p.lastDonateTime)

	donateRate0 := p.donateRate0.Clone()
	donateRate1 := p.donateRate1.Clone()

	var virtualDonateDeltaTimesCrossed int64
	var feesAccumulated bool
	var realLastDonateTime uint64

	if uint32(currentTime) != uint32(p.lastDonateTime) {
		realLastDonateTime = realLastTime(currentTime, uint32(p.lastDonateTime))
		time := realLastDonateTime
		tmp := new(uint256.Int)

		for _, delta := range p.donateRateDeltas {
			if delta.Time <= realLastDonateTime {
				continue
			}

			if delta.Time > currentTime {
				break
			}

			timeDiff := uint256.NewInt(delta.Time - time)
			feesAccumulated = feesAccumulated || !tmp.Mul(donateRate0, timeDiff).Rsh(tmp, 32).IsZero() || !tmp.Mul(donateRate1, timeDiff).Rsh(tmp, 32).IsZero()

			donateRate0.Add(donateRate0, (*uint256.Int)(delta.Delta0))
			donateRate1.Add(donateRate1, (*uint256.Int)(delta.Delta1))

			time = delta.Time
			virtualDonateDeltaTimesCrossed += 1
		}
	} else {
		realLastDonateTime = currentTime
	}

	quote, err := p.BasePool.Quote(amount, isToken1)
	if err != nil {
		return nil, fmt.Errorf("quoting concentrated pool: %w", err)
	}

	// TODO
	quote.Gas += 0

	quote.SwapInfo.SwapStateAfter = NewBoostedFeesPoolSwapState(
		quote.SwapInfo.SwapStateAfter.(*BasePoolSwapState),
		NewTimedPoolSwapState(
			donateRate0,
			donateRate1,
			currentTime,
		),
	)

	return quote, nil
}

func (p *BoostedFeesPool) CloneSwapStateOnly() Pool {
	cloned := *p
	cloned.BasePool = p.BasePool.CloneSwapStateOnly().(*BasePool)
	cloned.donateRate0 = p.donateRate0.Clone()
	cloned.donateRate1 = p.donateRate1.Clone()
	return &cloned
}

func (p *BoostedFeesPool) SetSwapState(state quoting.SwapState) {
	boostedFeesState := state.(*BoostedFeesPoolSwapState)
	p.BasePool.SetSwapState(boostedFeesState.BasePoolSwapState)
	p.lastDonateTime = boostedFeesState.LastExecutionTime
	p.donateRate0 = boostedFeesState.Token0Rate
	p.donateRate1 = boostedFeesState.Token1Rate
}

func (p *BoostedFeesPool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteWithTimestampFn(amount, isToken1, estimatedBlockTimestamp)
}

func NewBoostedFeesPoolSwapState(baseSwapState *BasePoolSwapState, timedSwapState *TimedPoolSwapState) *BoostedFeesPoolSwapState {
	return &BoostedFeesPoolSwapState{
		BasePoolSwapState:  baseSwapState,
		TimedPoolSwapState: timedSwapState,
	}
}

func NewBoostedFeesPoolState(baseState *BasePoolState, timedState *TimedPoolState) *BoostedFeesPoolState {
	return &BoostedFeesPoolState{
		BasePoolState:  baseState,
		TimedPoolState: timedState,
	}
}

func NewBoostedFeesPool(key *ConcentratedPoolKey, state *BoostedFeesPoolState) *BoostedFeesPool {
	return &BoostedFeesPool{
		BasePool:         NewBasePool(key, state.BasePoolState),
		donateRate0:      state.Token0Rate,
		donateRate1:      state.Token1Rate,
		lastDonateTime:   state.LastExecutionTime,
		donateRateDeltas: state.VirtualDeltas,
	}
}
