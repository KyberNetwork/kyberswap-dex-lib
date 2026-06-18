package uniswapv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logDecoder   ILogDecoder
		feeTracker   IFeeTracker
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	tracker := &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   NewLogDecoder(),
	}
	if feeTrackerCfg := config.FeeTracker; feeTrackerCfg != nil {
		tracker.feeTracker = NewGenericFeeTracker(feeTrackerCfg)
	}
	return tracker, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	// Reserves come from logs when available; that already gives the block, so drop stale updates
	// before doing any RPC.
	logsReserve, blockNumber, errLogs := d.getReservesFromLogs(&params)
	fromLogs := errLogs == nil && !logsReserve.IsZero()
	if fromLogs && p.BlockNumber > blockNumber.Uint64() {
		logger.WithFields(logger.Fields{
			"pool_id":           p.Address,
			"pool_block_number": p.BlockNumber,
			"data_block_number": blockNumber.Uint64(),
		}).Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	reserveData := logsReserve
	if fromLogs {
		req.SetBlockNumber(blockNumber)
	} else {
		d.addReservesCall(req, p, &reserveData)
	}

	var previousExtra Extra
	_ = json.Unmarshal([]byte(p.Extra), &previousExtra)

	fee := d.config.Fee
	if d.feeTracker != nil {
		fee = previousExtra.Fee
		d.feeTracker.AddFeeCall(req, d.config.FactoryAddress, p.Address, &fee)
	}

	taxTracker, taxInfo := newTokenTaxTracker(d.config.FactoryAddress, p, previousExtra)
	if taxTracker != nil {
		taxTracker.AddCalls(req)
	}

	var resp *ethrpc.Response
	if !fromLogs {
		var err error
		resp, err = req.TryBlockAndAggregate()
		if err != nil {
			return p, err
		}
		blockNumber = resp.BlockNumber
		if p.BlockNumber > blockNumber.Uint64() {
			return p, nil
		}
	} else if d.feeTracker != nil || taxTracker != nil {
		var err error
		resp, err = req.TryAggregate()
		if err != nil {
			return p, err
		}
	}

	if taxTracker != nil {
		taxInfo = taxTracker.Resolve(resp)
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      reserveData,
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return d.updatePool(p, reserveData, fee, taxInfo, blockNumber)
}

// addReservesCall appends a getReserves call to req, filling out after the aggregate.
func (d *PoolTracker) addReservesCall(req *ethrpc.Request, p entity.Pool, out *ReserveData) {
	req.AddCall(&ethrpc.Call{
		ABI:    uniswapV2PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []any{out})
}

func (d *PoolTracker) updatePool(p entity.Pool, reserveData ReserveData, fee uint64,
	taxInfo tokentax.TaxInfo, blockNumber *big.Int) (entity.Pool, error) {
	extra := Extra{
		Fee:          fee,
		FeePrecision: d.config.FeePrecision,
	}
	if taxInfo.Checked {
		extra.TaxInfo = &taxInfo
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber.Uint64()

	// Keep pool listing timestamp if reserves unchanged since pool creation
	p.Timestamp = max(p.Timestamp, int64(reserveData.BlockTimestampLast))

	return p, nil
}

func (d *PoolTracker) getReservesFromLogs(params *pool.GetNewPoolStateParams) (ReserveData, *big.Int, error) {
	if len(params.Logs) == 0 {
		return ReserveData{}, nil, nil
	}

	return d.logDecoder.Decode(params.Logs, params.BlockHeaders)
}
