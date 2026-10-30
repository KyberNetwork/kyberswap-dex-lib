package ephemeral

import (
	"context"
	"errors"
	"fmt"
	"math/big"
)

// BinSnapshot is what a bin lens's getSnapshot returns, one field per output in the ABI's order.
// LiquidityBookV21Lens and LiquidityBookV20Lens return them all;
// PancakeInfinityBinLens all but the reserves, which a bin pool does not keep: the Vault
// counts tokens per manager.
type BinSnapshot struct {
	ActiveId    *big.Int
	ReserveX    *big.Int
	ReserveY    *big.Int
	BlockNumber *big.Int
	NextId      *big.Int
	Done        bool
	Bins        []struct {
		Id       *big.Int
		ReserveX *big.Int
		ReserveY *big.Int
	}
}

// Bin is a non-empty bin as bin lenses return it.
type Bin struct {
	ID       uint32
	ReserveX *big.Int
	ReserveY *big.Int
}

// BinPool is a pool's active bin, its reserves when it keeps them, and every non-empty bin, read
// through a bin lens at one block.
type BinPool struct {
	Block    uint64
	ActiveID uint32
	ReserveX *big.Int // nil for a pool that keeps no reserves
	ReserveY *big.Int
	Bins     []Bin
}

// ErrBinsInconsistent means a bin set is not a pool's: out of order, holding an empty bin, or not
// adding up to the pool's reserves, so it lacks a bin or carries a wrong one.
var ErrBinsInconsistent = errors.New("ephemeral: bins are not the pool's")

// CheckBins reports whether bins, a pool's whole set read at one block, can be its set: ascending
// and none empty, and, for a pool that keeps reserves (reserveX not nil), summing to them exactly.
func CheckBins(bins []Bin, reserveX, reserveY *big.Int) error {
	sumX, sumY := new(big.Int), new(big.Int)
	for i, b := range bins {
		if i > 0 && b.ID <= bins[i-1].ID {
			return fmt.Errorf("%w: bin %d follows bin %d", ErrBinsInconsistent, b.ID, bins[i-1].ID)
		}
		if b.ReserveX.Sign() == 0 && b.ReserveY.Sign() == 0 {
			return fmt.Errorf("%w: bin %d is empty", ErrBinsInconsistent, b.ID)
		}
		sumX.Add(sumX, b.ReserveX)
		sumY.Add(sumY, b.ReserveY)
	}
	if reserveX != nil && (sumX.Cmp(reserveX) != 0 || sumY.Cmp(reserveY) != 0) {
		return fmt.Errorf("%w: %d bins hold (%s, %s), the pool reports (%s, %s)", ErrBinsInconsistent, len(bins),
			sumX, sumY, reserveX, reserveY)
	}
	return nil
}

// Check is CheckBins of the pool's bins against its reserves.
func (p *BinPool) Check() error {
	return CheckBins(p.Bins, p.ReserveX, p.ReserveY)
}

// ReadBins reads a pool through a bin lens at block (nil for the latest), paging from bin 0 with
// every page at one block, the first through the Batcher batch asks for (see CallPage). args are
// the lens's arguments before the bin to start at.
func ReadBins(ctx context.Context, c Node, batch *BatchConfig, contract *Contract, gas uint64, block *big.Int,
	args ...any) (*BinPool, error) {
	pin, err := Pin(ctx, c, block)
	if err != nil {
		return nil, err
	}
	p := BinPool{Bins: []Bin{}}
	first := true
	read, err := Scan(ctx, uint32(0), pin,
		func(ctx context.Context, start uint32, block *big.Int) (Page[uint32], error) {
			var out BinSnapshot
			if err := CallPage(ctx, batch, c, start, uint32(0), contract, &out, gas, block,
				append(args[:len(args):len(args)], new(big.Int).SetUint64(uint64(start)))...); err != nil {
				return Page[uint32]{}, err
			}
			if first {
				p.ActiveID, p.ReserveX, p.ReserveY = uint32(out.ActiveId.Uint64()), out.ReserveX, out.ReserveY
				first = false
			}
			for _, b := range out.Bins {
				p.Bins = append(p.Bins, Bin{ID: uint32(b.Id.Uint64()), ReserveX: b.ReserveX, ReserveY: b.ReserveY})
			}
			return Page[uint32]{Block: out.BlockNumber.Uint64(), Next: uint32(out.NextId.Uint64()), Done: out.Done},
				nil
		})
	if err != nil {
		return nil, err
	}
	p.Block = read
	return &p, nil
}
