package uniswapv4

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts V4Lens abi/lens.json
//go:embed abi/lens.json
var lensJSON []byte

// lens is V4Lens, built from pkg/util/ephemeral/contracts/uniswap/v4.
var lens = ephemeral.MustLoad("V4Lens", lensJSON)

// lensSnapshot is a pool's price, liquidity and every initialized tick, read by the lens at one
// block.
type lensSnapshot struct {
	block     uint64
	tick      int
	liquidity *big.Int
	ticks     []Tick
}

// snapshot reads the pool at block (nil for the latest) through the lens and StateView,
// paging when the ticks do not fit one call, and checks the ticks against the pool's liquidity.
func (t *PoolTracker) snapshot(ctx context.Context, p entity.Pool, block *big.Int) (*lensSnapshot, error) {
	// The tick spacing is part of the pool key the id hashes, which StateView does not return.
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: static extra: %w", p.Address, err)
	}
	pool, err := ephemeral.ReadTicks(ctx, t.ethrpcClient.GetETHClient(), t.config.LensBatch, lens, t.config.LensGas,
		ephemeral.WordLower, ephemeral.WordUpper, block,
		common.HexToAddress(t.config.StateViewAddress), common.HexToHash(p.Address),
		big.NewInt(int64(staticExtra.TickSpacing)))
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", p.Address, err)
	}
	if err := pool.Check(); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", p.Address, pool.Block, err)
	}
	s := lensSnapshot{block: pool.Block, tick: pool.Tick, liquidity: pool.Liquidity,
		ticks: make([]Tick, len(pool.Ticks))}
	for i, tk := range pool.Ticks {
		s.ticks[i] = Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	return &s, nil
}
