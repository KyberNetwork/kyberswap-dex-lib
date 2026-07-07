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
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/abis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
		sqrtRatioAfter *uint256.Int
		liquidityAfter *uint256.Int
	}
	positionUpdatedEvent struct {
		liquidityDelta *int256.Int
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
	if !bytes.Equal(poolId, expectedPoolId) {
		return nil, nil
	}

	return &swappedEvent{
		tickAfter:      int32(binary.BigEndian.Uint32(data[112:116])),
		sqrtRatioAfter: math.FloatSqrtRatioToFixed(new(uint256.Int).SetBytes(data[100:112])),
		liquidityAfter: new(uint256.Int).SetBytes(data[84:100]),
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

	if !bytes.Equal(expectedPoolId, poolId[:]) {
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
		liquidityDelta: int256.MustFromBig(params.LiquidityDelta),
		lower:          params.Bounds.Lower,
		upper:          params.Bounds.Upper,
	}, nil
}

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
		if orderKey.BuyToken.Cmp(orderKey.SellToken) > 0 {
			token0, token1 = orderKey.SellToken, orderKey.BuyToken
		} else {
			token0, token1 = orderKey.BuyToken, orderKey.SellToken
		}

		poolKey := p.GetKey()
		if poolKey.Token0 != token0 || poolKey.Token1 != token1 || poolKey.Config.Fee != orderKey.Fee {
			return nil
		}

		saleRateDelta, ok := values[3].(*big.Int)
		if !ok {
			return errors.New("failed to parse saleRateDelta")
		}

		startIdx := 0
		sellsToken1 := orderKey.SellToken == token1
		affectedSaleRate := lo.Ternary(sellsToken1, p.token1SaleRate, p.token0SaleRate)
		uSaleRateDelta := big256.SFromBig(saleRateDelta)
		orderBoundaries := [2]struct {
			time          uint64
			saleRateDelta *int256.Int
		}{
			{
				time:          orderKey.StartTime.Uint64(),
				saleRateDelta: uSaleRateDelta,
			},
			{
				time:          orderKey.EndTime.Uint64(),
				saleRateDelta: new(int256.Int).Neg(uSaleRateDelta),
			},
		}

		for _, orderBoundary := range orderBoundaries {
			time := orderBoundary.time

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
							SaleRateDelta0: new(int256.Int),
							SaleRateDelta1: new(int256.Int),
						},
					)
				}

				orderDelta := &p.virtualOrderDeltas[idx]
				affectedSaleRateDelta := lo.Ternary(sellsToken1, orderDelta.SaleRateDelta1, orderDelta.SaleRateDelta0)
				affectedSaleRateDelta.Add(affectedSaleRateDelta, orderBoundary.saleRateDelta)

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

func (p *BasePool) NewBlock()      {}
func (p *FullRangePool) NewBlock() {}

func (p *OraclePool) NewBlock() {
	p.swappedThisBlock = false
}

func (p *MevResistPool) NewBlock() {
	p.swappedThisBlock = false
	p.lastTick = p.ActiveTick
}
