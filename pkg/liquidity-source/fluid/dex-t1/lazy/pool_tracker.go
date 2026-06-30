package lazy

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

var _ = pooltrack.RegisterFactoryCE0(dexT1.DexType, NewPoolTracker)

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
	d := newRPCData()
	req := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides)
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) }, p.Address, t.config.DexReservesResolver, d)

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": dexT1.DexType, "error": err}).Error("Failed to get pool reserves")
		return p, err
	}

	return buildPoolState(p, d, resp.BlockNumber)
}

type rpcData struct {
	poolReserves  *dexT1.PoolWithReserves
	dexVariables2 *big.Int
}

func newRPCData() *rpcData {
	return &rpcData{
		poolReserves:  &dexT1.PoolWithReserves{},
		dexVariables2: new(big.Int),
	}
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress, resolverAddress string, d *rpcData) {
	addFn(&ethrpc.Call{
		ABI:    *dexT1.DexReservesResolverABI,
		Target: resolverAddress,
		Method: dexT1.DRRMethodGetPoolReservesAdjusted,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&d.poolReserves})
	addFn(&ethrpc.Call{
		ABI:    *dexT1.StorageReadABI,
		Target: poolAddress,
		Method: dexT1.SRMethodReadFromStorage,
		Params: []any{common.HexToHash("0x1")},
	}, []any{&d.dexVariables2})
}

func buildPoolState(p entity.Pool, d *rpcData, blockNumber *big.Int) (entity.Pool, error) {
	isSwapAndArbitragePaused := d.dexVariables2.Bit(255) == 1

	poolReserves := d.poolReserves
	extra := dexT1.PoolExtra{
		CollateralReserves:       poolReserves.CollateralReserves,
		DebtReserves:             poolReserves.DebtReserves,
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
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
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
