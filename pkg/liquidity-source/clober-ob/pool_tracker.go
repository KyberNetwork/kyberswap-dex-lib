package cloberob

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"pool":  p.Address,
		"dexId": t.config.DexId,
	})
	l.Info("start getting new state")

	var (
		highest           cloberlib.Tick
		liquidity         []Liquidity
		maxExpectedOutput struct {
			TakenQuoteAmount *big.Int
			SpentBaseAmount  *big.Int
		}
	)

	bookId, _ := new(big.Int).SetString(p.Address, 10)

	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    bookManagerABI,
		Target: t.config.BookManager.String(),
		Method: bookManagerMethodGetHighest,
		Params: []any{bookId},
	}, []any{&highest}).AddCall(&ethrpc.Call{
		ABI:    bookManagerABI,
		Target: t.config.BookViewer.String(),
		Method: bookViewerMethodGetExpectedOutput,
		Params: []any{bookId, bignumber.ZeroBI, bignumber.MAX_INT_128, common.Hash{}},
	}, []any{&maxExpectedOutput}).AddCall(&ethrpc.Call{
		ABI:    bookViewerABI,
		Target: t.config.BookViewer.String(),
		Method: bookViewerMethodGetLiquidity,
		Params: []any{bookId, cloberlib.MaxTick, maxTickLimit},
	}, []any{&maxExpectedOutput}).Aggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to aggregate RPC requests")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		Highest: highest,
		Depths:  liquidity,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{"0", maxExpectedOutput.TakenQuoteAmount.String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	l.Info("finish updating state of pool")

	return p, nil
}
