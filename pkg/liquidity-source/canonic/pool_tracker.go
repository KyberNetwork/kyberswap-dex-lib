package canonic

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getPoolStateAtBlock(ctx, p, nil)
}

func (t *PoolTracker) getPoolStateAtBlock(
	ctx context.Context,
	p entity.Pool,
	blockNumber *big.Int,
) (entity.Pool, error) {
	logger.Infof("getting pool state for %v", p.Address)

	var (
		midPriceResult struct {
			Price     *big.Int
			Precision *big.Int
			UpdatedAt *big.Int
		}
		takerFeeResult    *big.Int
		marketStateResult *big.Int
		baseScaleResult   *big.Int
		quoteScaleResult  *big.Int
		depthResult       struct {
			AskRungBps        []uint16
			AskVolumesInBase  []*big.Int
			BidRungBps        []uint16
			BidVolumesInQuote []*big.Int
		}
	)

	req := t.ethrpcClient.R().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "getMidPrice",
	}, []any{&midPriceResult}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "takerFee",
		}, []any{&takerFeeResult}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "marketState",
		}, []any{&marketStateResult}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "baseScale",
		}, []any{&baseScaleResult}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "quoteScale",
		}, []any{&quoteScaleResult}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "getDepth",
			Params: []any{uint16(64)},
		}, []any{&depthResult})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	askVols := make([]*uint256.Int, len(depthResult.AskVolumesInBase))
	for i, v := range depthResult.AskVolumesInBase {
		askVols[i] = uint256.MustFromBig(v)
	}
	bidVols := make([]*uint256.Int, len(depthResult.BidVolumesInQuote))
	for i, v := range depthResult.BidVolumesInQuote {
		bidVols[i] = uint256.MustFromBig(v)
	}

	extra := Extra{
		MidPrice:   uint256.MustFromBig(midPriceResult.Price),
		MidPrec:    uint256.MustFromBig(midPriceResult.Precision),
		TakerFee:   uint256.MustFromBig(takerFeeResult),
		BaseScale:  uint256.MustFromBig(baseScaleResult),
		QuoteScale: uint256.MustFromBig(quoteScaleResult),
		AskBps:     depthResult.AskRungBps,
		AskVols:    askVols,
		BidBps:     depthResult.BidRungBps,
		BidVols:    bidVols,
		Active:     marketStateResult.Int64() == marketStateActive,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	totalAskBase := new(big.Int)
	for _, v := range depthResult.AskVolumesInBase {
		totalAskBase.Add(totalAskBase, v)
	}
	totalBidQuote := new(big.Int)
	for _, v := range depthResult.BidVolumesInQuote {
		totalBidQuote.Add(totalBidQuote, v)
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		totalAskBase.String(),
		totalBidQuote.String(),
	}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.Infof("finished pool state for %v (asks=%d bids=%d)", p.Address, len(extra.AskBps), len(extra.BidBps))

	return p, nil
}
