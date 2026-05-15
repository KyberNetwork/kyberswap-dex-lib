package nadswap

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
	logDecoder   ILogDecoder
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, c *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: c,
		logDecoder:   NewLogDecoder(),
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	start := time.Now()
	rd, blockNum, err := t.getReserves(ctx, p.Address, &params)
	if err != nil {
		return p, err
	}
	if blockNum != nil && p.BlockNumber > blockNum.Uint64() {
		logger.WithFields(logger.Fields{
			"pool_id": p.Address, "pool_block": p.BlockNumber, "data_block": blockNum.Uint64(),
		}).Info("skip update: data block older than pool block")
		return p, nil
	}

	extra := Extra(rd)
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = []string{rd.Reserve0.Dec(), rd.Reserve1.Dec()}
	p.Timestamp = time.Now().Unix()
	if blockNum != nil {
		p.BlockNumber = blockNum.Uint64()
	}

	logger.WithFields(logger.Fields{
		"pool_id": p.Address, "duration_ms": time.Since(start).Milliseconds(),
	}).Info("Finished getting new pool state")
	return p, nil
}

func (t *PoolTracker) getReserves(
	ctx context.Context, poolAddr string, params *pool.GetNewPoolStateParams,
) (ReserveData, *big.Int, error) {
	if params != nil && len(params.Logs) > 0 {
		rd, blockNum, err := t.logDecoder.Decode(params.Logs, params.BlockHeaders)
		if err == nil && !rd.IsZero() {
			return rd, blockNum, nil
		}
	}
	return t.getReservesFromRPC(ctx, poolAddr)
}

func (t *PoolTracker) getReservesFromRPC(ctx context.Context, poolAddr string) (ReserveData, *big.Int, error) {
	var result reservesRPCResult
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI: pairABI, Target: poolAddr, Method: pairMethodGetReserves,
	}, []any{&result})
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, nil, err
	}
	u0, _ := uint256.FromBig(result.Reserve0)
	u1, _ := uint256.FromBig(result.Reserve1)
	return ReserveData{Reserve0: u0, Reserve1: u1, BlockTimestampLast: result.BlockTimestampLast}, resp.BlockNumber, nil
}
