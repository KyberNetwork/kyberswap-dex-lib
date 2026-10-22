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

// lens is FluidDexV2Lens, released by KyberNetwork/ephemeral-contracts.
var lens = ephemeral.MustLoad("FluidDexV2Lens", lensJSON)

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
	var s lensSnapshot

	pin, err := ephemeral.Pin(ctx, t.ethrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	read, err := ephemeral.Scan(ctx, lensWordLower, pin,
		func(ctx context.Context, lower int16, block *big.Int) (ephemeral.Page[int16], error) {
			var out snapshot
			err := ephemeral.Call(ctx, t.ethrpcClient.GetETHClient(), lens, &out, t.config.LensGas, block,
				common.HexToAddress(t.config.Dex), new(big.Int).SetUint64(uint64(dexType)), common.HexToHash(dexID),
				lower, lensWordUpper)
			if err != nil {
				return ephemeral.Page[int16]{}, err
			}
			if s.liquidity == nil { // the first page
				s.tickSpacing = int(out.TickSpacing.Int64())
				s.sqrtPriceX96 = out.SqrtPriceX96
				s.tick = int(out.Tick.Int64())
				s.liquidity = out.Liquidity
			}
			for _, pt := range out.Ticks {
				s.ticks = append(s.ticks, Tick{
					Index:          int(pt.Tick.Int64()),
					LiquidityNet:   pt.LiquidityNet,
					LiquidityGross: pt.LiquidityGross,
				})
			}
			return ephemeral.Page[int16]{Block: out.BlockNumber.Uint64(), Next: out.NextWordPos, Done: out.Done}, nil
		})
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	s.block = read

	checked := make([]ephemeral.Tick, len(s.ticks))
	for i, tk := range s.ticks {
		checked[i] = ephemeral.Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	if err := ephemeral.CheckTicks(checked, s.tick, s.liquidity); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, s.block, err)
	}

	return &s, nil
}
