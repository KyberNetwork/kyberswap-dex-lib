package pools

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/ethereum/go-ethereum/common"
)

type Event int

const (
	EventSwapped Event = iota + 1
	EventPositionUpdated
	EventVirtualOrdersExecuted
	EventOrderUpdated
)

type (
	swappedEvent struct {
		tickAfter      int32
		sqrtRatioAfter *big.Int
		liquidityAfter *big.Int
	}
	positionUpdatedEvent struct {
		liquidityDelta *big.Int
		lower          int32
		upper          int32
	}
)

func parseSwappedEventIfMatching(data []byte, poolKey *PoolKey) (*swappedEvent, error) {
	poolId := data[20:52]

	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return nil, fmt.Errorf("computing expected pool id: %w", err)
	}
	if slices.Compare(poolId, expectedPoolId) != 0 {
		return nil, nil
	}

	var tickAfter int32
	{
		buf := bytes.NewReader(data[112:116])
		if err := binary.Read(buf, binary.BigEndian, &tickAfter); err != nil {
			return nil, fmt.Errorf("reading tick: %w", err)
		}
	}

	temp := new(big.Int)
	sqrtRatioAfter := math.FloatSqrtRatioToFixed(temp.SetBytes(data[100:112]))
	liquidityAfter := temp.SetBytes(data[84:100])

	return &swappedEvent{
		tickAfter,
		sqrtRatioAfter,
		liquidityAfter,
	}, nil
}

func parsePositionUpdatedEventIfMatching(data []byte, poolKey *PoolKey) (*positionUpdatedEvent, error) {
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

	if slices.Compare(expectedPoolId, poolId[:]) != 0 {
		return nil, nil
	}

	params, ok := values[2].(struct {
		Salt   [32]uint8 `json:"salt"`
		Bounds struct {
			Lower int32 `json:"lower"`
			Upper int32 `json:"upper"`
		} `json:"bounds"`
		LiquidityDelta *big.Int `json:"liquidityDelta"`
	})
	if !ok {
		return nil, errors.New("failed to parse params")
	}

	if params.LiquidityDelta.Sign() == 0 {
		return nil, nil
	}

	return &positionUpdatedEvent{
		liquidityDelta: params.LiquidityDelta,
		lower:          params.Bounds.Lower,
		upper:          params.Bounds.Upper,
	}, nil
}

func (p *BasePool) ApplyEvent(event Event, data []byte) error {
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
			p.Liquidity.Add(p.Liquidity, liquidityDelta)
		}
	}

	return nil
}

func (p *FullRangePool) ApplyEvent(event Event, data []byte) error {
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

		p.Liquidity.Add(p.Liquidity, event.liquidityDelta)
	}

	return nil
}

func (p *TwammPool) ApplyEvent(event Event, data []byte) error {
	switch event {
	case EventVirtualOrdersExecuted:
		expectedPoolId, err := p.GetKey().NumId()
		if err != nil {
			return fmt.Errorf("computing expected pool id: %w", err)
		}

		if slices.Compare(data[0:32], expectedPoolId) != 0 {
			return nil
		}

		// TODO Need a timestamp here to update the last execution time

		p.token0SaleRate.SetBytes(data[32:46])
		p.token1SaleRate.SetBytes(data[46:60])
	case EventOrderUpdated:
		values, err := abis.OrderUpdatedEvent.Inputs.Unpack(data)
		if err != nil {
			return fmt.Errorf("unpacking event data: %w", err)
		}

		orderKey, ok := values[2].(struct {
			SellToken common.Address `json:"sellToken"`
			BuyToken  common.Address `json:"buyToken"`
			Fee       uint64         `json:"fee"`
			StartTime *big.Int       `json:"startTime"`
			EndTime   *big.Int       `json:"endTime"`
		})
		if !ok {
			return errors.New("failed to parse orderKey")
		}

		var token0, token1 common.Address
		if orderKey.BuyToken.Cmp(orderKey.SellToken) == 1 {
			token0, token1 = orderKey.SellToken, orderKey.BuyToken
		} else {
			token0, token1 = orderKey.BuyToken, orderKey.SellToken
		}

		poolKey := p.GetKey()

		if poolKey.Token0.Cmp(token0) != 0 || poolKey.Token1.Cmp(token1) != 0 || poolKey.Config.Fee != orderKey.Fee {
			return nil
		}

		sellsToken1 := orderKey.SellToken.Cmp(token1) == 0

		var affectedSaleRate *big.Int
		if sellsToken1 {
			affectedSaleRate = p.token1SaleRate
		} else {
			affectedSaleRate = p.token0SaleRate
		}

		saleRateDelta, ok := values[3].(*big.Int)
		if !ok {
			return errors.New("failed to parse saleRateDelta")
		}

		startIdx := 0

		orderBoundaries := [2]struct {
			time          *big.Int
			saleRateDelta *big.Int
		}{
			{
				time:          orderKey.StartTime,
				saleRateDelta: saleRateDelta,
			},
			{
				time:          orderKey.EndTime,
				saleRateDelta: new(big.Int).Neg(saleRateDelta),
			},
		}

		for _, orderBoundary := range orderBoundaries {
			time := orderBoundary.time.Uint64()

			if time > p.lastExecutionTime {
				idx, found := slices.BinarySearchFunc(p.virtualOrderDeltas[startIdx:], time, func(srd TwammSaleRateDelta, time uint64) int {
					return cmp.Compare(srd.Time, time)
				})

				idx += startIdx

				if !found {
					p.virtualOrderDeltas = slices.Insert(
						p.virtualOrderDeltas,
						idx,
						TwammSaleRateDelta{
							Time:           time,
							SaleRateDelta0: new(big.Int),
							SaleRateDelta1: new(big.Int),
						},
					)
				}

				orderDelta := &p.virtualOrderDeltas[idx]
				var affectedSaleRateDelta *big.Int
				if sellsToken1 {
					affectedSaleRateDelta = orderDelta.SaleRateDelta1
				} else {
					affectedSaleRateDelta = orderDelta.SaleRateDelta0
				}
				affectedSaleRateDelta.Add(affectedSaleRateDelta, orderBoundary.saleRateDelta)

				startIdx = idx + 1
			} else {
				affectedSaleRate.Add(affectedSaleRate, orderBoundary.saleRateDelta)
			}
		}
	default:
		return p.FullRangePool.ApplyEvent(event, data)
	}

	return nil
}

func (p *BasePool) NewBlock()      {}
func (p *FullRangePool) NewBlock() {}

func (p *OraclePool) NewBlock() {
	p.swappedThisBlock = false
}
