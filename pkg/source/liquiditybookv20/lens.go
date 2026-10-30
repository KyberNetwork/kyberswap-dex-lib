package liquiditybookv20

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

//go:embed abis/lens.json
var lensJSON []byte

// lens is LiquidityBookV20Lens, built from pkg/util/ephemeral/contracts/liquiditybook/v20.
var lens = ephemeral.MustLoad("LiquidityBookV20Lens", lensJSON)

// lensSnapshot is a pair's active bin, reserves and every non-empty bin, read by the lens at one
// block.
type lensSnapshot struct {
	block              uint64
	activeID           uint32
	reserveX, reserveY *big.Int
	bins               []Bin
}

// snapshot reads the pair at block (nil for the latest) through the lens, paging when the bins do
// not fit one call, and checks the bins against the pair's reserves. The swap
// fees a v2.0 pair keeps apart from its bins are in neither getReservesAndId nor any bin, so a bin
// missing or wrong shows as a difference.
func (t *PoolTracker) snapshot(ctx context.Context, address string, block *big.Int) (*lensSnapshot, error) {
	p, err := ephemeral.ReadBins(ctx, t.ethrpcClient.GetETHClient(), t.cfg.LensBatch, lens, t.cfg.LensGas, block,
		common.HexToAddress(address))
	if err != nil {
		return nil, fmt.Errorf("lens snapshot of %s: %w", address, err)
	}
	if err := p.Check(); err != nil {
		return nil, fmt.Errorf("lens snapshot of %s at block %d: %w", address, p.Block, err)
	}
	s := lensSnapshot{block: p.Block, activeID: p.ActiveID, reserveX: p.ReserveX, reserveY: p.ReserveY,
		bins: make([]Bin, len(p.Bins))}
	for i, b := range p.Bins {
		s.bins[i] = Bin(b)
	}
	return &s, nil
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
