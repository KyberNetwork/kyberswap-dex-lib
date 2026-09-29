package flowstatec1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE(DexType, NewPoolTracker)

func NewPoolTracker(ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{ethrpcClient: ethrpcClient}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"poolAddress": p.Address})

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	inventoryToken := p.Tokens[1].Address

	// quoteBuyFromPool's fillableAmount is min(requested amount, actual depth), so a
	// probe below expected depth only yields the unit rate, not the real cap. Depth
	// comes from the inventory token's balance of the pool instead (verified on-chain
	// to equal quoteBuyFromPool's fillableAmount when probing at/above full depth).
	var (
		quoteOut  struct{ Quote Quote }
		balanceOf *big.Int
	)
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    marketABI,
		Target: staticExtra.Market,
		Method: "quoteBuyFromPool",
		Params: []any{
			common.HexToAddress(staticExtra.Pool),
			common.HexToAddress(staticExtra.QuoteAsset),
			extra.ProbeAmount.ToBig(),
		},
	}, []any{&quoteOut})
	req.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: inventoryToken,
		Method: "balanceOf",
		Params: []any{common.HexToAddress(staticExtra.Pool)},
	}, []any{&balanceOf})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to track flowstate-c1 pool state")
		return p, err
	}

	quote := quoteOut.Quote
	extra.Available = quote.Available
	extra.ProbeQuoteCost = uint256.MustFromBig(quote.QuoteAmount)
	extra.FillableAmount = uint256.MustFromBig(balanceOf)
	extra.FeeBps = quote.FeeBps

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{"0", extra.FillableAmount.String()}
	p.Timestamp = time.Now().Unix()

	if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	l.Info("successfully tracked flowstate-c1 pool state")

	return p, nil
}
