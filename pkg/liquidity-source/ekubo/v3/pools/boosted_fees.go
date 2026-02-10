package pools

import (
	"cmp"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type (
	BoostedFeesPoolSwapState struct {
		*ConcentratedPoolSwapState
		*TimedPoolSwapState
	}

	BoostedFeesPoolState struct {
		*ConcentratedPoolState
		*TimedPoolState
	}

	BoostedFeesPool struct {
		*ConcentratedPool
		donateRate0      *uint256.Int
		donateRate1      *uint256.Int
		lastDonateTime   uint64
		donateRateDeltas []TimeRateDelta
	}
)

func (p *BoostedFeesPool) GetState() any {
	return NewBoostedFeesPoolState(
		p.ConcentratedPoolState,
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

	quote, err := p.ConcentratedPool.Quote(amount, isToken1)
	if err != nil {
		return nil, fmt.Errorf("quoting concentrated pool: %w", err)
	}

	quote.Gas += quoting.ExtraBaseGasCostOfOneBoostedFeesSwap

	if currentTime != realLastDonateTime {
		quote.Gas += quoting.GasCostOfExecutingVirtualDonations
	}

	if feesAccumulated {
		quote.Gas += quoting.GasCostOfBoostedFeesFeeAccumulation
	}

	quote.Gas += approximateExtraDistinctTimeBitmapLookups(realLastDonateTime, currentTime)*quoting.GasCostOfOneColdSload +
		virtualDonateDeltaTimesCrossed*quoting.GasCostOfCrossingOneVirtualDonateDelta

	quote.SwapInfo.SwapStateAfter = NewBoostedFeesPoolSwapState(
		quote.SwapInfo.SwapStateAfter.(*ConcentratedPoolSwapState),
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
	cloned.ConcentratedPool = p.ConcentratedPool.CloneSwapStateOnly().(*ConcentratedPool)
	cloned.donateRate0 = p.donateRate0.Clone()
	cloned.donateRate1 = p.donateRate1.Clone()
	return &cloned
}

func (p *BoostedFeesPool) SetSwapState(state quoting.SwapState) {
	boostedFeesState := state.(*BoostedFeesPoolSwapState)
	p.ConcentratedPool.SetSwapState(boostedFeesState.ConcentratedPoolSwapState)
	p.lastDonateTime = boostedFeesState.LastExecutionTime
	p.donateRate0 = boostedFeesState.Token0Rate
	p.donateRate1 = boostedFeesState.Token1Rate
}

func (p *BoostedFeesPool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	return p.quoteWithTimestampFn(amount, isToken1, estimatedBlockTimestamp)
}

func (p *BoostedFeesPool) ApplyEvent(event Event, data []byte, blockTimestamp uint64) error {
	switch event {
	case EventFeesDonated:
		expectedPoolId, err := p.GetKey().NumId()
		if err != nil {
			return fmt.Errorf("computing expected pool id: %w", err)
		}

		if slices.Compare(data[0:32], expectedPoolId) != 0 {
			return nil
		}

		if blockTimestamp == 0 {
			return errors.New("block timestamp is zero")
		}

		p.lastDonateTime = blockTimestamp
		p.donateRate0.SetBytes(data[32:46])
		p.donateRate1.SetBytes(data[46:60])
	case EventPoolBoosted:
		expectedPoolId, err := p.GetKey().NumId()
		if err != nil {
			return fmt.Errorf("computing expected pool id: %w", err)
		}

		if slices.Compare(data[0:32], expectedPoolId) != 0 {
			return nil
		}

		values, err := abis.PoolBoostedEvent.Inputs.Unpack(data)
		if err != nil {
			return fmt.Errorf("unpacking event data: %w", err)
		}

		startTime, ok := values[1].(uint64)
		if !ok {
			return errors.New("failed to parse startTime")
		}

		endTime, ok := values[2].(uint64)
		if !ok {
			return errors.New("failed to parse endTime")
		}

		var rate0, rate1 *int256.Int
		{
			rate0Abi, ok := values[3].(*big.Int)
			if !ok {
				return errors.New("failed to parse rate0")
			}

			rate1Abi, ok := values[4].(*big.Int)
			if !ok {
				return errors.New("failed to parse rate1")
			}

			if rate0Abi.Sign() == 0 && rate1Abi.Sign() == 0 {
				return nil
			}

			rate0, rate1 = big256.SFromBig(rate0Abi), big256.SFromBig(rate1Abi)
		}

		startIdx := 0
		orderBoundaries := [2]TimeRateDelta{
			{
				Time:   startTime,
				Delta0: rate0,
				Delta1: rate1,
			},
			{
				Time:   endTime,
				Delta0: new(int256.Int).Neg(rate0),
				Delta1: new(int256.Int).Neg(rate1),
			},
		}

		for _, orderBoundary := range orderBoundaries {
			time := orderBoundary.Time

			if time > p.lastDonateTime {
				idx, found := slices.BinarySearchFunc(p.donateRateDeltas[startIdx:], time, func(srd TimeRateDelta, time uint64) int {
					return cmp.Compare(srd.Time, time)
				})

				idx += startIdx

				if !found {
					p.donateRateDeltas = slices.Insert(
						p.donateRateDeltas,
						idx,
						TimeRateDelta{
							Time:   time,
							Delta0: new(int256.Int),
							Delta1: new(int256.Int),
						},
					)
				}

				virtualDelta := &p.donateRateDeltas[idx]
				virtualDelta.Delta0.Add(virtualDelta.Delta0, orderBoundary.Delta0)
				virtualDelta.Delta1.Add(virtualDelta.Delta1, orderBoundary.Delta1)

				if virtualDelta.Delta0.IsZero() && virtualDelta.Delta1.IsZero() {
					p.donateRateDeltas = slices.Delete(p.donateRateDeltas, idx, idx+1)
				}

				startIdx = idx + 1
			} else {
				p.donateRate0.Add(p.donateRate0, (*uint256.Int)(rate0))
				p.donateRate1.Add(p.donateRate1, (*uint256.Int)(rate1))
			}
		}
	default:
		return p.ConcentratedPool.ApplyEvent(event, data, blockTimestamp)
	}

	return nil
}

func (p *BoostedFeesPool) NewBlock() {}

func NewBoostedFeesPoolSwapState(concentratedSwapState *ConcentratedPoolSwapState, timedSwapState *TimedPoolSwapState) *BoostedFeesPoolSwapState {
	return &BoostedFeesPoolSwapState{
		ConcentratedPoolSwapState: concentratedSwapState,
		TimedPoolSwapState:        timedSwapState,
	}
}

func NewBoostedFeesPoolState(concentratedState *ConcentratedPoolState, timedState *TimedPoolState) *BoostedFeesPoolState {
	return &BoostedFeesPoolState{
		ConcentratedPoolState: concentratedState,
		TimedPoolState:        timedState,
	}
}

func NewBoostedFeesPool(key *ConcentratedPoolKey, state *BoostedFeesPoolState) *BoostedFeesPool {
	return &BoostedFeesPool{
		ConcentratedPool: NewConcentratedPool(key, state.ConcentratedPoolState),
		donateRate0:      state.Token0Rate,
		donateRate1:      state.Token1Rate,
		lastDonateTime:   state.LastExecutionTime,
		donateRateDeltas: state.VirtualDeltas,
	}
}

func approximateExtraDistinctTimeBitmapLookups(startTime, endTime uint64) int64 {
	return int64((endTime >> 16) - (startTime >> 16))
}
