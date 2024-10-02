package dexT1

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	if poolReserves.CollateralReserves.Token0RealReserves == nil ||
		poolReserves.CollateralReserves.Token1RealReserves == nil ||
		poolReserves.CollateralReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) != 0 ||
		poolReserves.CollateralReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) != 0 ||
		poolReserves.DebtReserves.Token0RealReserves == nil ||
		poolReserves.DebtReserves.Token1RealReserves == nil ||
		poolReserves.DebtReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) != 0 ||
		poolReserves.DebtReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) != 0 {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error reserves are nil / 0")
		return p, errors.New("pool reserves are nil / 0")
	}

	extra := PoolExtra{
		CollateralReserves: poolReserves.CollateralReserves,
		DebtReserves:       poolReserves.DebtReserves,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	p.SwapFee = float64(poolReserves.Fee.Int64()) / 10000
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
		Method: DRRMethodGetPoolReserves,
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
