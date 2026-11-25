package v3

import (
	"context"
	"math/big"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// lensSnapshot is a pool's state and every initialized tick, read by the lens at one block.
type lensSnapshot struct {
	block     uint64
	tick      int
	liquidity *big.Int
	ticks     []Tick
}

// snapshot reads the pool through uniswap-v3's V3Lens: a Native V3 pool keeps a Uniswap V3 pool's
// slot0, tickBitmap and ticks, which is all the lens reads.
func (t *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	p, err := uniswapv3.ReadLens(ctx, t.ethrpcClient.GetETHClient(), t.config.LensConfig, address, block)
	if err != nil {
		return nil, err
	}
	s := lensSnapshot{block: p.Block, tick: p.Tick, liquidity: p.Liquidity, ticks: make([]Tick, len(p.Ticks))}
	for i, tk := range p.Ticks {
		s.ticks[i] = Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	return &s, nil
}
