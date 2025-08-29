package velodromev1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	logDecoder   uniswapv2.ILogDecoder
	feeTracker   IFeeTracker
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   uniswapv2.NewLogDecoder(),
		feeTracker:   NewGenericFeeTracker(ethrpcClient, config.FeeTracker),
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":     p.Address,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pool state")
	}()

	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var isPaused bool
	req := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    pairFactoryABI,
		Target: d.config.FactoryAddress,
		Method: pairFactoryMethodIsPaused,
	}, []any{&isPaused})

	reserveData := d.getReserves(req, p.Address, &params)

	fee := d.config.Fee
	if d.feeTracker != nil {
		req = d.feeTracker.AddGetFeeCall(req, d.config.FactoryAddress, p.Address, staticExtra.Stable, &fee)
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	return d.updatePool(p, *reserveData, isPaused, fee, resp.BlockNumber)
}

func (d *PoolTracker) getReserves(req *ethrpc.Request, poolAddress string, params *pool.GetNewPoolStateParams) *ReserveData {
	reserveData, blockNumber, err := d.getReservesFromLogs(params)
	if err != nil || reserveData.IsZero() {
		result, _ := d.getReservesFromRPCNode(req, poolAddress)
		return result
	}
	req.SetBlockNumber(blockNumber)
	return &reserveData
}

func (d *PoolTracker) updatePool(
	p entity.Pool,
	reserveData ReserveData,
	isPaused bool,
	fee uint64,
	blockNumber *big.Int) (entity.Pool, error) {
	if p.BlockNumber > blockNumber.Uint64() {
		return p, nil
	}

	poolExtra := PoolExtra{
		IsPaused: isPaused,
		Fee:      fee,
	}
	poolExtraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	p.Extra = string(poolExtraBytes)
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = int64(reserveData.BlockTimestampLast)

	return p, nil
}

func (d *PoolTracker) getReservesFromRPCNode(req *ethrpc.Request, poolAddress string) (*ReserveData,
	*ethrpc.Request) {
	var getReservesResult ReserveData
	return &getReservesResult, req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
	}, []any{&getReservesResult})
}

func (d *PoolTracker) getReservesFromLogs(params *pool.GetNewPoolStateParams) (ReserveData, *big.Int, error) {
	if len(params.Logs) == 0 || d.logDecoder == nil {
		return ReserveData{}, nil, nil
	}
	return d.logDecoder.Decode(params.Logs, params.BlockHeaders)
}
