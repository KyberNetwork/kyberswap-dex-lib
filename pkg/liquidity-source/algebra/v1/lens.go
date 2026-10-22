package algebrav1

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:embed abis/lens.json
var lensJSON []byte

// lens is AlgebraV1Lens, released by KyberNetwork/ephemeral-contracts.
var lens = ephemeral.MustLoad("AlgebraV1Lens", lensJSON)

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

// The widest word range there is, that of a table indexed by tick. The lens narrows it to the
// words the pool's spacing can populate when the table is indexed by tick over the spacing.
const (
	lensWordLower int16 = -3466
	lensWordUpper int16 = 3465
)

// tickTable is how an exchange's pools index their tick table: by tick over the spacing, as
// Uniswap V3 does, or by tick. Algebra V1 deployments do either, and no pool says which.
type tickTable = int32

const (
	tickTableUnknown tickTable = iota
	tickTableCompressed
	tickTableRaw
)

// lensSnapshot is a pool's state and every initialized tick, read by the lens at one block.
type lensSnapshot struct {
	block       uint64
	tickSpacing int
	tick        int
	liquidity   *big.Int
	ticks       []tickspkg.Tick
}

// readLens reads a pool whole through the lens at block (nil for the latest), its tick table taken
// as indexed by tick when raw, paging when the ticks do not fit one call.
type readLens func(ctx context.Context, address string, block *big.Int, raw bool) (*lensSnapshot, error)

// snapshot reads the pool at block (nil for the latest) through the lens, and checks the ticks
// against the pool's liquidity.
//
// The exchange's table indexing is learnt from its pools and kept once a read finds ticks: a table
// read the wrong way either reverts TickTableMismatch, a set bit reading as a tick without
// liquidity, or, read by tick over the spacing when it is by tick, comes back without ticks, all of
// them outside the words such a read covers. A pool of spacing 1 reads alike either way and says
// nothing.
func (t *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	s, err := t.readTable(ctx, address, block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}

	checked := make([]ephemeral.Tick, len(s.ticks))
	for i, tk := range s.ticks {
		checked[i] = ephemeral.Tick{Index: tk.TickIdx, LiquidityNet: tk.LiquidityNet, LiquidityGross: tk.LiquidityGross}
	}
	if err := ephemeral.CheckTicks(checked, s.tick, s.liquidity); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, s.block, err)
	}
	return s, nil
}

// readTable reads the pool the way its exchange indexes the tick table, finding that out when it
// is not known yet.
func (t *PoolTracker) readTable(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	known := t.tickTable.Load()
	s, err := t.readLens(ctx, address, block, known == tickTableRaw)
	switch {
	case ephemeral.IsContractError(err, "TickTableMismatch"):
		// A pool of another kind than the rest of its exchange is read the other way without
		// unlearning the exchange's.
		if s, err = t.readLens(ctx, address, block, known != tickTableRaw); err == nil && known == tickTableUnknown &&
			len(s.ticks) > 0 {
			t.tickTable.CompareAndSwap(tickTableUnknown, tickTableRaw)
		}
		return s, err
	case err != nil:
		return nil, err
	case known == tickTableCompressed && s.tickSpacing > 1 && len(s.ticks) == 0:
		// A pool indexed by tick read over the spacing comes back without ticks rather than
		// reverting when they all lie past the words that read covers: taken as it is, the pool
		// would keep no ticks, or fail the check against its liquidity on every read.
		raw, err := t.readLens(ctx, address, new(big.Int).SetUint64(s.block), true)
		switch {
		case ephemeral.IsContractError(err, "TickTableMismatch"):
			return s, nil
		case err != nil:
			return nil, err
		}
		return raw, nil
	case known != tickTableUnknown:
		return s, nil
	case s.tickSpacing <= 1:
		// Indexed by tick over a spacing of 1 is indexed by tick: the read tells nothing.
		return s, nil
	case len(s.ticks) > 0:
		t.tickTable.CompareAndSwap(tickTableUnknown, tickTableCompressed)
		return s, nil
	}

	raw, err := t.readLens(ctx, address, new(big.Int).SetUint64(s.block), true)
	if err != nil {
		return nil, err
	}
	if len(raw.ticks) > 0 {
		t.tickTable.CompareAndSwap(tickTableUnknown, tickTableRaw)
	}
	return raw, nil
}

// readLensPages is readLens over the node: the pages of one read, at the block the first one read.
func (t *PoolTracker) readLensPages(ctx context.Context, address string, block *big.Int, raw bool) (*lensSnapshot,
	error) {
	pool := common.HexToAddress(address)
	var s lensSnapshot

	pin, err := ephemeral.Pin(ctx, t.EthrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, err
	}
	read, ticks, err := ephemeral.ScanWords(ctx, lensWordLower, lensWordUpper, pin, ephemeral.WordsParallel,
		func(ctx context.Context, lower, upper int16, block *big.Int) (ephemeral.Page[int16], []tickspkg.Tick, error) {
			var out snapshot
			err := ephemeral.CallPage(ctx, t.config.LensBatch, t.EthrpcClient.GetETHClient(), lower, lensWordLower, lens, &out, t.config.LensGas, block,
				pool, raw, lower, upper)
			if err != nil {
				return ephemeral.Page[int16]{}, nil, err
			}
			if s.liquidity == nil { // the first page
				s.tickSpacing = int(out.TickSpacing.Int64())
				s.tick = int(out.Tick.Int64())
				s.liquidity = out.Liquidity
			}
			var got []tickspkg.Tick
			for _, pt := range out.Ticks {
				got = append(got, tickspkg.Tick{
					TickIdx:        int(pt.Tick.Int64()),
					LiquidityGross: pt.LiquidityGross,
					LiquidityNet:   pt.LiquidityNet,
				})
			}
			return ephemeral.Page[int16]{Block: out.BlockNumber.Uint64(), Next: out.NextWordPos, Done: out.Done}, got, nil
		})
	if err != nil {
		return nil, err
	}
	s.block = read
	s.ticks = ticks
	return &s, nil
}
