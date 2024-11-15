package dexT1

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
)

type PoolTracker struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	poolReserves, blockNumber, err := t.getPoolReserves(ctx, p.Address)
	if err != nil {
		return p, err
	}

	extra := PoolExtra{
		CollateralReserves: poolReserves.CollateralReserves,
		DebtReserves:       poolReserves.DebtReserves,
		DexLimits:          poolReserves.Limits,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	p.SwapFee = float64(poolReserves.Fee.Int64()) / float64(FeePercentPrecision)
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		new(big.Int).Add(poolReserves.CollateralReserves.Token0RealReserves, poolReserves.DebtReserves.Token0RealReserves).String(),
		new(big.Int).Add(poolReserves.CollateralReserves.Token1RealReserves, poolReserves.DebtReserves.Token1RealReserves).String(),
	}

	return p, nil
}

func (t *PoolTracker) getPoolReserves(ctx context.Context, poolAddress string) (*PoolWithReserves, uint64, error) {
	pool := &PoolWithReserves{}

	req := t.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    dexReservesResolverABI,
		Target: t.config.DexReservesResolver,
		Method: DRRMethodGetPoolReservesAdjusted,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&pool})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get pool reserves")
		return nil, 0, err
	}

	return pool, resp.BlockNumber.Uint64(), nil
}
