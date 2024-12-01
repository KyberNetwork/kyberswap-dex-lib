package integral

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/daoleno/uniswapv3-sdk/utils"
)

type TickManager struct {
	data map[int32]Tick
}

type TickFromRPC struct {
	LiquidityTotal *big.Int `json:"liquidityTotal"` // the total position liquidity that references this tick
	LiquidityDelta *big.Int `json:"liquidityDelta"` // amount of net liquidity added (subtracted) when tick is crossed left-right (right-left),
	PrevTick       *big.Int `json:"prevTick"`
	NextTick       *big.Int `json:"nextTick"`
	// fee growth per unit of liquidity on the _other_ side of this tick (relative to the current tick)
	// only has relative meaning, not absolute — the value depends on when the tick is initialized
	OuterFeeGrowth0Token *big.Int `json:"outerFeeGrowth0Token"`
	OuterFeeGrowth1Token *big.Int `json:"outerFeeGrowth1Token"`
}

func (t TickFromRPC) toTick() Tick {
	return Tick{
		liquidityDelta: t.LiquidityDelta,
		prevTick:       int32(t.PrevTick.Int64()),
		nextTick:       int32(t.NextTick.Int64()),
	}
}

type Tick struct {
	liquidityDelta *big.Int `json:"liquidityDelta"`
	prevTick       int32    `json:"prevTick"`
	nextTick       int32    `json:"nextTick"`
}

func NewTickManager(ticks map[int32]Tick) *TickManager {
	return &TickManager{
		data: ticks,
	}
}

func (t *TickManager) Get(tick int32) Tick {
	if tickData, ok := t.data[tick]; ok {
		return tickData
	}

	return Tick{
		liquidityDelta: bignumber.ZeroBI,
		prevTick:       utils.MinTick,
		nextTick:       utils.MaxTick,
	}
}

func (t *TickManager) cross(tick int32) (*big.Int, int32, int32) {
	data := t.Get(tick)

	return data.liquidityDelta, data.prevTick, data.nextTick
}
