package dexv2

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:embed abis/lens.json
var lensJSON []byte

// lens is FluidDexV2Lens, built from pkg/util/ephemeral/contracts/fluid/dex-v2.
var lens = ephemeral.MustLoad("FluidDexV2Lens", lensJSON)

// A tick lies within MIN_TICK and MAX_TICK, so these words cover a spacing-1 pool; the lens reads
// the pool's own spacing and skips the words it cannot populate.
const (
	lensWordLower int16 = MIN_TICK >> 8
	lensWordUpper int16 = MAX_TICK >> 8
)

// lensSnapshot is a pool's state and every initialized tick, read by the lens at one block.
type lensSnapshot struct {
	block        uint64
	tickSpacing  int
	sqrtPriceX96 *big.Int
	tick         int
	liquidity    *big.Int
	ticks        []Tick
}

// snapshot reads the pool at block (nil for the latest) through the lens, paging when the ticks do
// not fit one call, and checks the ticks against the pool's liquidity.
func (t *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	dexID, dexType := parseFluidDexV2PoolAddress(address)
	p, err := ephemeral.ReadTicks(ctx, t.ethrpcClient.GetETHClient(), t.config.LensBatch, lens, t.config.LensGas,
		lensWordLower, lensWordUpper, block,
		common.HexToAddress(t.config.Dex), new(big.Int).SetUint64(uint64(dexType)), common.HexToHash(dexID))
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	if err := p.Check(); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, p.Block, err)
	}
	s := lensSnapshot{block: p.Block, tickSpacing: int(p.TickSpacing.Int64()), sqrtPriceX96: p.SqrtPriceX96,
		tick: p.Tick, liquidity: p.Liquidity, ticks: make([]Tick, len(p.Ticks))}
	for i, tk := range p.Ticks {
		s.ticks[i] = Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	return &s, nil
}
