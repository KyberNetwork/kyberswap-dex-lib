package bin

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts PancakeInfinityBinLens abi/lens.json
//go:embed abi/lens.json
var lensJSON []byte

// lens is PancakeInfinityBinLens, built from pkg/util/ephemeral/contracts/pancake/infinity/bin.
var lens = ephemeral.MustLoad("PancakeInfinityBinLens", lensJSON)

// lensSnapshot is a pool's active bin and every non-empty bin, read by the lens at one block.
type lensSnapshot struct {
	block    uint64
	activeID uint32
	bins     []Bin
}

// snapshot reads the pool at block (nil for the latest) through the lens, paging when the bins do
// not fit one call, and checks that they can be its set. A bin pool keeps no reserves of its own to
// sum the bins against: the Vault counts tokens per manager.
func (t *PoolTracker) snapshot(ctx context.Context, poolID string, block *big.Int) (*lensSnapshot, error) {
	p, err := ephemeral.ReadBins(ctx, t.ethrpcClient.GetETHClient(), t.config.LensBatch, lens, t.config.LensGas,
		block, common.HexToAddress(t.config.BinPoolManagerAddress), common.HexToHash(poolID))
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", poolID, err)
	}
	if err := p.Check(); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", poolID, p.Block, err)
	}
	s := lensSnapshot{block: p.Block, activeID: p.ActiveID, bins: make([]Bin, len(p.Bins))}
	for i, b := range p.Bins {
		s.bins[i] = Bin{ID: b.ID, ReserveX: uint256.MustFromBig(b.ReserveX), ReserveY: uint256.MustFromBig(b.ReserveY)}
	}
	return &s, nil
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
