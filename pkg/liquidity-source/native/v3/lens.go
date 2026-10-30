package v3

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

// lens is V3Lens, built from pkg/util/ephemeral/contracts/uniswap/v3. A Native V3 pool
// keeps a Uniswap V3 pool's slot0, tickBitmap and ticks, which is all the lens reads.
var lens = ephemeral.MustLoad("V3Lens", lensJSON)

// snapshot is what getSnapshot returns, one field per output in the ABI's order.
type snapshot struct {
	TickSpacing  *big.Int
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Liquidity    *big.Int
	BlockNumber  *big.Int
	NextWordPos  int16
	Done         bool
	Ticks        []struct {
		Tick           *big.Int
		LiquidityNet   *big.Int
		LiquidityGross *big.Int
	}
}

// The widest word range there is, tick spacing 1's. The lens reads the pool's own spacing and
// skips the words it cannot populate, so this covers every pool.
const (
	lensWordLower int16 = -3466
	lensWordUpper int16 = 3465
)

// lensSnapshot is a pool's state and every initialized tick, read by the lens at one block.
type lensSnapshot struct {
	block     uint64
	tick      int
	liquidity *big.Int
	ticks     []Tick
}

// snapshot reads the pool at block (nil for the latest) through the lens, paging when the ticks
// do not fit one call, and checks the ticks against the pool's liquidity.
func (t *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	pool := common.HexToAddress(address)
	var s lensSnapshot

	pin, err := ephemeral.Pin(ctx, t.ethrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	read, ticks, err := ephemeral.ScanWords(ctx, lensWordLower, lensWordUpper, pin, ephemeral.WordsParallel,
		func(ctx context.Context, lower, upper int16, block *big.Int) (ephemeral.Page[int16], []Tick, error) {
			var out snapshot
			err := ephemeral.CallPage(ctx, t.config.LensBatch, t.ethrpcClient.GetETHClient(), lower, lensWordLower, lens, &out, t.config.LensGas, block,
				pool, lower, upper)
			if err != nil {
				return ephemeral.Page[int16]{}, nil, err
			}
			if s.liquidity == nil { // the first page
				s.tick = int(out.Tick.Int64())
				s.liquidity = out.Liquidity
			}
			var got []Tick
			for _, pt := range out.Ticks {
				got = append(got, Tick{
					Index:          int(pt.Tick.Int64()),
					LiquidityNet:   pt.LiquidityNet,
					LiquidityGross: pt.LiquidityGross,
				})
			}
			return ephemeral.Page[int16]{Block: out.BlockNumber.Uint64(), Next: out.NextWordPos, Done: out.Done}, got, nil
		})
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	s.block = read
	s.ticks = ticks

	checked := make([]ephemeral.Tick, len(s.ticks))
	for i, tk := range s.ticks {
		checked[i] = ephemeral.Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	if err := ephemeral.CheckTicks(checked, s.tick, s.liquidity); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, s.block, err)
	}

	return &s, nil
}
