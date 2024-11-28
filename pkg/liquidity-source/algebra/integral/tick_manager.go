package integral

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type TickManager struct {
	data map[int32]Tick
}

type Tick struct {
	liquidityTotal *big.Int // the total position liquidity that references this tick
	liquidityDelta *big.Int // amount of net liquidity added (subtracted) when tick is crossed left-right (right-left),
	prevTick       int32
	nextTick       int32
	// fee growth per unit of liquidity on the _other_ side of this tick (relative to the current tick)
	// only has relative meaning, not absolute — the value depends on when the tick is initialized
	outerFeeGrowth0Token *big.Int
	outerFeeGrowth1Token *big.Int
}

func NewTickManager() *TickManager {
	return &TickManager{
		data: make(map[int32]Tick),
	}
}

func (t *TickManager) Get(tick int32) Tick {
	if tickData, ok := t.data[tick]; ok {
		return tickData
	}

	return Tick{
		liquidityTotal:       bignumber.ZeroBI,
		liquidityDelta:       bignumber.ZeroBI,
		prevTick:             0,
		nextTick:             0,
		outerFeeGrowth0Token: bignumber.ZeroBI,
		outerFeeGrowth1Token: bignumber.ZeroBI,
	}
}

func (t *TickManager) cross(tick int32, feeGrowth0, feeGrowth1 *big.Int) (*big.Int, int32, int32) {
	data := t.Get(tick)

	data.outerFeeGrowth1Token = new(big.Int).Sub(feeGrowth1, data.outerFeeGrowth1Token)
	data.outerFeeGrowth0Token = new(big.Int).Sub(feeGrowth0, data.outerFeeGrowth0Token)

	return data.liquidityDelta, data.prevTick, data.nextTick
}
