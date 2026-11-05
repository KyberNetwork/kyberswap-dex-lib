package uniswapv3

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts V3Lens abis/lens.json
//go:embed abis/lens.json
var lensJSON []byte

// lens is V3Lens, built from pkg/util/ephemeral/contracts/uniswap/v3.
var lens = ephemeral.MustLoad("V3Lens", lensJSON)

// lensSnapshot is a pool's state and every initialized tick, read by the lens at one block.
type lensSnapshot struct {
	block     uint64
	tick      int
	liquidity *big.Int
	ticks     []Tick
}

// ReadLens reads a pool at block (nil for the latest) through V3Lens, paging when the ticks do not
// fit one call, and checks the ticks against the pool's liquidity. A pool that keeps a Uniswap V3
// pool's slot0, tickBitmap and ticks, all the lens reads, can be read with it.
func ReadLens(ctx context.Context, node ephemeral.Node, cfg ephemeral.LensConfig, address string,
	block *big.Int) (*ephemeral.TickPool, error) {
	p, err := ephemeral.ReadTicks(ctx, node, lens, cfg.LensGas, ephemeral.WordLower,
		ephemeral.WordUpper, block, common.HexToAddress(address))
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	if err := p.Check(); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, p.Block, err)
	}
	return p, nil
}

// snapshot is ReadLens of the pool with the tracker's node and config.
func (t *Tracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	p, err := ReadLens(ctx, t.ethrpcClient.GetETHClient(), t.config.LensConfig, address, block)
	if err != nil {
		return nil, err
	}
	return newLensSnapshot(p), nil
}

// newLensSnapshot is p with its ticks in this package's Tick.
func newLensSnapshot(p *ephemeral.TickPool) *lensSnapshot {
	s := lensSnapshot{block: p.Block, tick: p.Tick, liquidity: p.Liquidity, ticks: make([]Tick, len(p.Ticks))}
	for i, tk := range p.Ticks {
		s.ticks[i] = Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	return &s
}
