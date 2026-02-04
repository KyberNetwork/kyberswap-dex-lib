package pools

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	EventSwapped Event = iota + 1
	EventPositionUpdated
	EventVirtualOrdersExecuted
	EventOrderUpdated
	EventFeesDonated
	EventPoolBoosted
)

type (
	Event int

	swappedEvent struct {
		sqrtRatioAfter *uint256.Int
		tickAfter      int32
		liquidityAfter *uint256.Int
	}
	positionUpdatedEvent struct {
		liquidityDelta *int256.Int
		lower          int32
		upper          int32
	}
)

func (p *BasePool) ApplyEvent(event Event, data []byte, _ uint64) error {
	switch event {
	case EventSwapped:
		event, err := parseSwappedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.ActiveTick = event.tickAfter
		p.SqrtRatio = event.sqrtRatioAfter
		p.Liquidity = event.liquidityAfter

		p.ActiveTickIndex = NearestInitializedTickIndex(p.SortedTicks, p.ActiveTick)
	case EventPositionUpdated:
		event, err := parsePositionUpdatedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		lower, upper, liquidityDelta := event.lower, event.upper, event.liquidityDelta

		p.UpdateTick(lower, liquidityDelta, false, false)
		p.UpdateTick(upper, liquidityDelta, true, false)

		p.ActiveTickIndex = NearestInitializedTickIndex(p.SortedTicks, p.ActiveTick)

		if p.ActiveTick >= lower && p.ActiveTick < upper {
			p.Liquidity.Add(p.Liquidity, (*uint256.Int)(liquidityDelta))
		}
	default:
	}
	return nil
}

func (p *StableswapPool) ApplyEvent(event Event, data []byte, _ uint64) error {
	switch event {
	case EventSwapped:
		event, err := parseSwappedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.SqrtRatio = event.sqrtRatioAfter
		p.Liquidity = event.liquidityAfter
	case EventPositionUpdated:
		event, err := parsePositionUpdatedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.Liquidity.Add(p.Liquidity, (*uint256.Int)(event.liquidityDelta))
	default:
	}

	return nil
}

func (p *FullRangePool) ApplyEvent(event Event, data []byte, _ uint64) error {
	switch event {
	case EventSwapped:
		event, err := parseSwappedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.SqrtRatio = event.sqrtRatioAfter
		p.Liquidity = event.liquidityAfter
	case EventPositionUpdated:
		event, err := parsePositionUpdatedEventIfMatching(data, p.GetKey())
		if err != nil || event == nil {
			return err
		}

		p.Liquidity.Add(p.Liquidity, (*uint256.Int)(event.liquidityDelta))
	default:
	}
	return nil
}

func (p *TwammPool) ApplyEvent(event Event, data []byte, blockTimestamp uint64) error {
	switch event {
	case EventVirtualOrdersExecuted:
		if blockTimestamp == 0 {
			return fmt.Errorf("block timestamp is zero")
		}

		expectedPoolId, err := p.GetKey().NumId()
		if err != nil {
			return fmt.Errorf("computing expected pool id: %w", err)
		}

		if slices.Compare(data[0:32], expectedPoolId) != 0 {
			return nil
		}

		p.lastExecutionTime = blockTimestamp
		p.token0SaleRate.SetBytes(data[32:46])
		p.token1SaleRate.SetBytes(data[46:60])
	case EventOrderUpdated:
		values, err := abis.OrderUpdatedEvent.Inputs.Unpack(data)
		if err != nil {
			return fmt.Errorf("unpacking event data: %w", err)
		}

		saleRateDelta, ok := values[3].(*big.Int)
		if !ok {
			return errors.New("failed to parse saleRateDelta")
		}

		if saleRateDelta.Sign() == 0 {
			return nil
		}

		orderKeyAbi, ok := values[2].(TwammOrderKeyAbi)
		if !ok {
			return errors.New("failed to parse orderKey")
		}
		orderKey := TwammOrderKey{TwammOrderKeyAbi: orderKeyAbi}

		poolKey := p.GetKey()
		if poolKey.Token0Address() != orderKey.Token0 || poolKey.Token1Address() != orderKey.Token1 || poolKey.Fee() != orderKey.Fee() {
			return nil
		}

		startIdx := 0
		sellsToken1 := orderKey.SellsToken1()
		affectedSaleRate := lo.Ternary(sellsToken1, p.token1SaleRate, p.token0SaleRate)
		uSaleRateDelta := big256.SFromBig(saleRateDelta)
		orderBoundaries := [2]struct {
			time          uint64
			saleRateDelta *int256.Int
		}{
			{
				time:          orderKey.StartTime(),
				saleRateDelta: uSaleRateDelta,
			},
			{
				time:          orderKey.EndTime(),
				saleRateDelta: new(int256.Int).Neg(uSaleRateDelta),
			},
		}

		for _, orderBoundary := range orderBoundaries {
			time := orderBoundary.time

			if time > p.lastExecutionTime {
				idx, found := slices.BinarySearchFunc(p.virtualOrderDeltas[startIdx:], time, func(srd TimeRateDelta, time uint64) int {
					return cmp.Compare(srd.Time, time)
				})

				idx += startIdx

				if !found {
					p.virtualOrderDeltas = slices.Insert(
						p.virtualOrderDeltas,
						idx,
						TimeRateDelta{
							Time:   time,
							Delta0: new(int256.Int),
							Delta1: new(int256.Int),
						},
					)
				}

				orderDelta := &p.virtualOrderDeltas[idx]
				affectedSaleRateDelta := lo.Ternary(sellsToken1, orderDelta.Delta1, orderDelta.Delta0)
				affectedSaleRateDelta.Add(affectedSaleRateDelta, orderBoundary.saleRateDelta)

				if orderDelta.Delta0.IsZero() && orderDelta.Delta1.IsZero() {
					p.virtualOrderDeltas = slices.Delete(p.virtualOrderDeltas, idx, idx+1)
				}

				startIdx = idx + 1
			} else {
				affectedSaleRate.Add(affectedSaleRate, (*uint256.Int)(orderBoundary.saleRateDelta))
			}
		}
	default:
		return p.FullRangePool.ApplyEvent(event, data, blockTimestamp)
	}

	return nil
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
		return p.BasePool.ApplyEvent(event, data, blockTimestamp)
	}

	return nil
}

func (p *BasePool) NewBlock()       {}
func (p *FullRangePool) NewBlock()  {}
func (p *StableswapPool) NewBlock() {}

func (p *OraclePool) NewBlock() {
	p.swappedThisBlock = false
}

func (p *MevCapturePool) NewBlock() {
	p.swappedThisBlock = false
	p.lastTick = p.ActiveTick
}

func parseSwappedEventIfMatching(data []byte, poolKey IPoolKey) (*swappedEvent, error) {
	if len(data) < 116 {
		return nil, nil
	}

	poolId := data[20:52]
	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return nil, fmt.Errorf("computing expected pool id: %w", err)
	}
	if !bytes.Equal(poolId, expectedPoolId) {
		return nil, nil
	}

	return &swappedEvent{
		sqrtRatioAfter: math.FloatSqrtRatioToFixed(new(uint256.Int).SetBytes(data[84:96])),
		tickAfter:      int32(binary.BigEndian.Uint32(data[96:100])),
		liquidityAfter: new(uint256.Int).SetBytes(data[100:116]),
	}, nil
}

func parsePositionUpdatedEventIfMatching(data []byte, poolKey IPoolKey) (*positionUpdatedEvent, error) {
	values, err := abis.PositionUpdatedEvent.Inputs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("unpacking event data: %w", err)
	}

	poolId, ok := values[1].([32]byte)
	if !ok {
		return nil, errors.New("failed to parse poolId")
	}

	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return nil, fmt.Errorf("computing expected pool id: %w", err)
	}

	if !bytes.Equal(expectedPoolId, poolId[:]) {
		return nil, nil
	}

	liquidityDelta, ok := values[3].(*big.Int)
	if !ok {
		return nil, errors.New("failed to parse liquidityDelta")
	}

	if liquidityDelta.Sign() == 0 {
		return nil, nil
	}

	params, ok := values[2].([32]byte)
	if !ok {
		return nil, errors.New("failed to parse positionId")
	}

	return &positionUpdatedEvent{
		liquidityDelta: int256.MustFromBig(liquidityDelta),
		lower:          int32(binary.BigEndian.Uint32(params[24:28])),
		upper:          int32(binary.BigEndian.Uint32(params[28:32])),
	}, nil
}
