package canonic

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dex_id":       t.config.DexId,
		"pool_address": p.Address,
	}).Info("started getting new pool state")

	p, err := t.getPoolState(ctx, p, nil)
	if err != nil {
		return p, err
	}

	logger.WithFields(logger.Fields{
		"dex_id":       t.config.DexId,
		"pool_address": p.Address,
	}).Info("finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) getPoolState(ctx context.Context, p entity.Pool, blockNumber *big.Int) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	maobAddr := p.Address

	var (
		midPriceRes struct {
			Price     *big.Int
			Precision *big.Int
			UpdatedAt *big.Int
		}
		takerFeeBI   *big.Int
		feeDenom     *big.Int
		minQuoteTkr  *big.Int
		mktStateBI   *big.Int
		stateExpiry  *big.Int
		rungDenom    *big.Int
		priceSigfigs *big.Int
		depth        struct {
			AskRungs   []uint16
			AskVolumes []*big.Int
			BidRungs   []uint16
			BidVolumes []*big.Int
		}
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodGetMidPrice,
	}, []any{&midPriceRes})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodTakerFee,
	}, []any{&takerFeeBI})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodFeeDenom,
	}, []any{&feeDenom})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodMinQuoteTaker,
	}, []any{&minQuoteTkr})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodMarketState,
	}, []any{&mktStateBI})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodStateExpiresAt,
	}, []any{&stateExpiry})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodRungDenom,
	}, []any{&rungDenom})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodPriceSigfigs,
	}, []any{&priceSigfigs})
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: maobAddr,
		Method: maobMethodGetDepth,
		Params: []any{defaultRungCount},
	}, []any{&depth})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	for i, r := range resp.Result {
		if !r {
			return p, fmt.Errorf("canonic tracker: call %d failed for %s", i, maobAddr)
		}
	}

	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	askVolStrs := bignumber.ToStrings(depth.AskVolumes)
	bidVolStrs := bignumber.ToStrings(depth.BidVolumes)

	extra := Extra{
		MidPrice:       midPriceRes.Price.String(),
		MidPrecision:   midPriceRes.Precision.String(),
		OracleUpdAt:    midPriceRes.UpdatedAt.Uint64(),
		TakerFee:       uint32(takerFeeBI.Uint64()),
		FeeDenom:       feeDenom.String(),
		MinQuoteTaker:  minQuoteTkr.String(),
		MarketState:    uint8(mktStateBI.Uint64()),
		StateExpiresAt: stateExpiry.Uint64(),
		RungDenom:      rungDenom.String(),
		PriceSigfigs:   priceSigfigs.String(),
		AskRungs:       depth.AskRungs,
		AskVolumes:     askVolStrs,
		BidRungs:       depth.BidRungs,
		BidVolumes:     bidVolStrs,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	totalBase := bignumber.Sum(depth.AskVolumes)
	totalQuote := bignumber.Sum(depth.BidVolumes)

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{totalBase.String(), totalQuote.String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
