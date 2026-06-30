package backup

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	dexT1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-t1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *dexT1.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterBackupFactoryCE0(dexT1.DexType, NewPoolTracker)

func NewPoolTracker(config *dexT1.Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	poolReserves, isSwapAndArbitragePaused, blockNumber, err := t.getPoolReserves(ctx, p.Address, overrides)
	if err != nil {
		return p, err
	}

	collateralReserves, debtReserves := poolReserves.CollateralReserves, poolReserves.DebtReserves
	extra := dexT1.PoolExtra{
		CollateralReserves:       collateralReserves,
		DebtReserves:             debtReserves,
		IsSwapAndArbitragePaused: isSwapAndArbitragePaused,
		DexLimits:                poolReserves.Limits,
		CenterPrice:              poolReserves.CenterPrice,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": dexT1.DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	p.SwapFee = float64(poolReserves.Fee.Int64()) / dexT1.FeePercentPrecision
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		dexT1.GetMaxReserves(
			p.Tokens[0].Decimals,
			poolReserves.Limits.WithdrawableToken0,
			poolReserves.Limits.BorrowableToken0,
			poolReserves.CollateralReserves.Token0RealReserves,
			poolReserves.DebtReserves.Token0RealReserves).String(),
		dexT1.GetMaxReserves(
			p.Tokens[1].Decimals,
			poolReserves.Limits.WithdrawableToken1,
			poolReserves.Limits.BorrowableToken1,
			poolReserves.CollateralReserves.Token1RealReserves,
			poolReserves.DebtReserves.Token1RealReserves).String(),
	}

	return p, nil
}

func (t *PoolTracker) getPoolReserves(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*dexT1.PoolWithReserves, bool, uint64, error) {
	poolReserves := &dexT1.PoolWithReserves{}
	dexVariables2 := new(big.Int)

	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)

	req.AddCall(&ethrpc.Call{
		ABI:    *dexT1.DexReservesResolverABI,
		Target: t.config.DexReservesResolver,
		Method: dexT1.DRRMethodGetPoolReservesAdjusted,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&poolReserves})

	req.AddCall(&ethrpc.Call{
		ABI:    *dexT1.StorageReadABI,
		Target: poolAddress,
		Method: dexT1.SRMethodReadFromStorage,
		Params: []any{common.HexToHash("0x1")},
	}, []any{&dexVariables2})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": dexT1.DexType,
			"error":   err,
		}).Error("Failed to get pool reserves")
		return nil, false, 0, err
	}

	isSwapAndArbitragePaused := dexVariables2.Bit(255) == 1

	return poolReserves, isSwapAndArbitragePaused, resp.BlockNumber.Uint64(), nil
}
