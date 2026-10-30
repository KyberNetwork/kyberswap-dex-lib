package bin

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:embed abi/lens.json
var lensJSON []byte

// lens is PancakeInfinityBinLens, built from pkg/util/ephemeral/contracts/pancake/infinity/bin.
var lens = ephemeral.MustLoad("PancakeInfinityBinLens", lensJSON)

// snapshot is what getSnapshot returns, one field per output in the ABI's order.
type snapshot struct {
	ActiveId    *big.Int
	BlockNumber *big.Int
	NextId      *big.Int
	Done        bool
	Bins        []struct {
		Id       *big.Int
		ReserveX *big.Int
		ReserveY *big.Int
	}
}

// lensSnapshot is a pool's active bin and every non-empty bin, read by the lens at one block.
type lensSnapshot struct {
	block    uint64
	activeID uint32
	bins     []Bin
}

// errBinsInconsistent means a lens read is not a pool's bin set: out of order, or holding an
// empty bin. Unlike a Liquidity Book pair, a bin pool has no reserves of its own to sum the bins
// against: the Vault counts tokens per manager.
var errBinsInconsistent = errors.New("bins are not a pool's bin set")

// snapshot reads the pool at block (nil for the latest) through the lens, paging when the bins do
// not fit one call.
func (t *PoolTracker) snapshot(ctx context.Context, poolID string, block *big.Int) (*lensSnapshot, error) {
	manager := common.HexToAddress(t.config.BinPoolManagerAddress)
	id := common.HexToHash(poolID)
	s := lensSnapshot{bins: []Bin{}}
	first := true

	pin, err := ephemeral.Pin(ctx, t.ethrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", poolID, err)
	}
	read, err := ephemeral.Scan(ctx, uint32(0), pin,
		func(ctx context.Context, start uint32, block *big.Int) (ephemeral.Page[uint32], error) {
			var out snapshot
			err := ephemeral.CallPage(ctx, t.config.LensBatch, t.ethrpcClient.GetETHClient(), start, uint32(0), lens, &out, t.config.LensGas, block,
				manager, id, new(big.Int).SetUint64(uint64(start)))
			if err != nil {
				return ephemeral.Page[uint32]{}, err
			}
			if first {
				s.activeID = uint32(out.ActiveId.Uint64())
				first = false
			}
			for _, b := range out.Bins {
				s.bins = append(s.bins, Bin{
					ID:       uint32(b.Id.Uint64()),
					ReserveX: uint256.MustFromBig(b.ReserveX),
					ReserveY: uint256.MustFromBig(b.ReserveY),
				})
			}
			return ephemeral.Page[uint32]{Block: out.BlockNumber.Uint64(), Next: uint32(out.NextId.Uint64()),
				Done: out.Done}, nil
		})
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", poolID, err)
	}
	s.block = read

	if err := checkBins(s.bins); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", poolID, s.block, err)
	}
	return &s, nil
}

// checkBins reports whether bins can be a pool's set: ascending and none empty.
func checkBins(bins []Bin) error {
	for i, b := range bins {
		if i > 0 && b.ID <= bins[i-1].ID {
			return fmt.Errorf("%w: bin %d follows bin %d", errBinsInconsistent, b.ID, bins[i-1].ID)
		}
		if b.ReserveX.IsZero() && b.ReserveY.IsZero() {
			return fmt.Errorf("%w: bin %d is empty", errBinsInconsistent, b.ID)
		}
	}
	return nil
}

// withActiveBin returns bins with the active one in it, empty when it holds nothing: a swap starts
// in the active bin and steps on from it, and the simulator finds the next bin by the active one's
// place in the set, which an empty active bin, dropped like every empty bin, does not have.
func withActiveBin(bins []Bin, activeID uint32) []Bin {
	i, found := slices.BinarySearchFunc(bins, activeID, func(b Bin, id uint32) int {
		return int(int64(b.ID) - int64(id))
	})
	if found {
		return bins
	}
	return slices.Insert(slices.Clip(bins), i, Bin{ID: activeID, ReserveX: uint256.NewInt(0),
		ReserveY: uint256.NewInt(0)})
}

// hasState reports whether a tracker has given the pool its bins. A pool factory writes the bins as
// null, and a tracker as a list, empty for a pool holding no liquidity.
func hasState(p entity.Pool) bool {
	var extra struct {
		Bins json.RawMessage `json:"bins"`
	}
	return len(p.Extra) > 0 && json.Unmarshal([]byte(p.Extra), &extra) == nil && len(extra.Bins) > 0 &&
		string(extra.Bins) != "null"
}
