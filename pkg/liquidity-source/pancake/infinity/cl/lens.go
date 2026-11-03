package cl

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts PancakeInfinityCLLens abi/lens.json
//go:embed abi/lens.json
var lensJSON []byte

// lens is PancakeInfinityCLLens, built from pkg/util/ephemeral/contracts/pancake/infinity/cl.
var lens = ephemeral.MustLoad("PancakeInfinityCLLens", lensJSON)

// lensSnapshot is a pool's price, liquidity, tick spacing and every initialized tick, read by the
// lens at one block.
type lensSnapshot struct {
	block       uint64
	tickSpacing uint64
	tick        int
	liquidity   *big.Int
	ticks       []Tick
}

// snapshot reads the pool at block (nil for the latest) through the lens and CLPoolManager,
// paging when the ticks do not fit one call, and checks the ticks against the pool's liquidity.
func (t *PoolTracker) snapshot(ctx context.Context, poolID string, block *big.Int) (*lensSnapshot, error) {
	p, err := ephemeral.ReadTicks(ctx, t.ethrpcClient.GetETHClient(), t.config.LensBatch, lens, t.config.LensGas,
		ephemeral.WordLower, ephemeral.WordUpper, block,
		common.HexToAddress(t.config.CLPoolManagerAddress), common.HexToHash(poolID))
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", poolID, err)
	}
	if err := p.Check(); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", poolID, p.Block, err)
	}
	s := lensSnapshot{block: p.Block, tickSpacing: p.TickSpacing.Uint64(),
		tick: p.Tick, liquidity: p.Liquidity, ticks: make([]Tick, len(p.Ticks))}
	for i, tk := range p.Ticks {
		s.ticks[i] = Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	return &s, nil
}
