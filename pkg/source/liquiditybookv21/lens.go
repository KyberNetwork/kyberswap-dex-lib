package liquiditybookv21

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:embed abis/lens.json
var lensJSON []byte

// lens is LiquidityBookV21Lens, built from pkg/util/ephemeral/contracts/liquiditybook/v21.
// It reads v2.2 pairs too: they answer the same four views.
var lens = ephemeral.MustLoad("LiquidityBookV21Lens", lensJSON)

// snapshot is what getSnapshot returns, one field per output in the ABI's order.
type snapshot struct {
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

// lensSnapshot is a pair's active bin, reserves and every non-empty bin, read by the lens at one
// block.
type lensSnapshot struct {
	block              uint64
	activeID           uint32
	reserveX, reserveY *big.Int
	bins               []Bin
}

// errBinsInconsistent means a pair's bins do not add up to its reserves, so the set lacks a bin or
// carries a wrong one.
var errBinsInconsistent = errors.New("bins do not add up to the pair's reserves")

// snapshot reads the pair at block (nil for the latest) through the lens, paging when the bins do
// not fit one call, and checks the bins against the pair's reserves.
func (t *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	pair := common.HexToAddress(address)
	s := lensSnapshot{bins: []Bin{}}

	pin, err := ephemeral.Pin(ctx, t.ethrpcClient.GetETHClient(), block)
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	read, err := ephemeral.Scan(ctx, uint32(0), pin,
		func(ctx context.Context, start uint32, block *big.Int) (ephemeral.Page[uint32], error) {
			var out snapshot
			err := ephemeral.CallPage(ctx, t.cfg.LensBatch, t.ethrpcClient.GetETHClient(), start, uint32(0), lens, &out, t.cfg.LensGas, block,
				pair, new(big.Int).SetUint64(uint64(start)))
			if err != nil {
				return ephemeral.Page[uint32]{}, err
			}
			if s.reserveX == nil { // the first page
				s.activeID = uint32(out.ActiveId.Uint64())
				s.reserveX, s.reserveY = out.ReserveX, out.ReserveY
			}
			for _, b := range out.Bins {
				s.bins = append(s.bins, Bin{ID: uint32(b.Id.Uint64()), ReserveX: b.ReserveX, ReserveY: b.ReserveY})
			}
			return ephemeral.Page[uint32]{Block: out.BlockNumber.Uint64(), Next: uint32(out.NextId.Uint64()),
				Done: out.Done}, nil
		})
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	s.block = read

	if err := checkBins(s.bins, s.reserveX, s.reserveY); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, s.block, err)
	}

	return &s, nil
}

// checkBins reports whether bins, a pair's whole set read at one block, agree with its reserves:
// ascending, none empty, and summing to them exactly. getReserves leaves out the protocol fees,
// which no bin holds either, so a bin missing or wrong shows as a difference.
func checkBins(bins []Bin, reserveX, reserveY *big.Int) error {
	sumX, sumY := new(big.Int), new(big.Int)
	for i, b := range bins {
		if i > 0 && b.ID <= bins[i-1].ID {
			return fmt.Errorf("%w: bin %d follows bin %d", errBinsInconsistent, b.ID, bins[i-1].ID)
		}
		if b.ReserveX.Sign() == 0 && b.ReserveY.Sign() == 0 {
			return fmt.Errorf("%w: bin %d is empty", errBinsInconsistent, b.ID)
		}
		sumX.Add(sumX, b.ReserveX)
		sumY.Add(sumY, b.ReserveY)
	}
	if sumX.Cmp(reserveX) != 0 || sumY.Cmp(reserveY) != 0 {
		return fmt.Errorf("%w: %d bins hold (%s, %s), the pair reports (%s, %s)", errBinsInconsistent, len(bins),
			sumX, sumY, reserveX, reserveY)
	}
	return nil
}

// hasState reports whether a tracker has given the pair its bins. A pair factory leaves the Extra
// empty, and a pair holding no liquidity still has its bins key, so the key itself tells the two
// apart.
func hasState(p entity.Pool) bool {
	var extra struct {
		Bins json.RawMessage `json:"bins"`
	}
	return len(p.Extra) > 0 && json.Unmarshal([]byte(p.Extra), &extra) == nil && len(extra.Bins) > 0
}
