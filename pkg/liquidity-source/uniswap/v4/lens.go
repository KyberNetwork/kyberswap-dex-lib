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

//go:embed abi/lens.json
var lensJSON []byte

// lens is V4Lens, built from pkg/util/ephemeral/contracts/uniswap/v4.
var lens = ephemeral.MustLoad("V4Lens", lensJSON)

// snapshot is what getSnapshot returns, one field per output in the ABI's order.
type snapshot struct {
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

// The widest word range there is, tick spacing 1's. The lens skips the words the pool's spacing
// cannot populate, so this covers every pool.
const (
	lensWordLower int16 = -3466
	lensWordUpper int16 = 3465
)

// lensSnapshot is a pool's price, liquidity and every initialized tick, read by the lens at one
// block.
type lensSnapshot struct {
	block        uint64
	sqrtPriceX96 *big.Int
	tick         int
	liquidity    *big.Int
	ticks        []Tick
}

// snapshot reads the pool at block (nil for the latest) through the lens and StateView,
// paging when the ticks do not fit one call, and checks the ticks against the pool's liquidity.
func (t *PoolTracker) snapshot(ctx context.Context, p entity.Pool, block *big.Int) (*lensSnapshot, error) {
	// The tick spacing is part of the pool key the id hashes, which StateView does not return.
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: static extra: %w", p.Address, err)
	}
	stateView := common.HexToAddress(t.config.StateViewAddress)
	id := common.HexToHash(p.Address)
	spacing := big.NewInt(int64(staticExtra.TickSpacing))
	var s lensSnapshot

	pin, err := ephemeral.Pin(ctx, t.ethrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", p.Address, err)
	}
	read, ticks, err := ephemeral.ScanWords(ctx, lensWordLower, lensWordUpper, pin, ephemeral.WordsParallel,
		func(ctx context.Context, lower, upper int16, block *big.Int) (ephemeral.Page[int16], []Tick, error) {
			var out snapshot
			err := ephemeral.CallPage(ctx, t.config.LensBatch, t.ethrpcClient.GetETHClient(), lower, lensWordLower, lens, &out, t.config.LensGas, block,
				stateView, id, spacing, lower, upper)
			if err != nil {
				return ephemeral.Page[int16]{}, nil, err
			}
			if s.sqrtPriceX96 == nil {
				s.sqrtPriceX96 = out.SqrtPriceX96
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
		return nil, fmt.Errorf("lens snapshot of %s: %w", p.Address, err)
	}
	s.block = read
	s.ticks = ticks

	checked := make([]ephemeral.Tick, len(s.ticks))
	for i, tk := range s.ticks {
		checked[i] = ephemeral.Tick{Index: tk.Index, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	if err := ephemeral.CheckTicks(checked, s.tick, s.liquidity); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", p.Address, s.block, err)
	}

	return &s, nil
}
