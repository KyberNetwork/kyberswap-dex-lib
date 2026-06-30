package dexT1

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Failed to get pool reserves")
		return p, err
	}

	return buildPoolState(p, d, resp.BlockNumber)
}

type rpcData struct {
	poolReserves  *PoolWithReserves
	dexVariables2 *big.Int
}

func newRPCData() *rpcData {
	return &rpcData{
		poolReserves:  &PoolWithReserves{},
		dexVariables2: new(big.Int),
	}
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress, resolverAddress string, d *rpcData) {
	addFn(&ethrpc.Call{
		ABI:    dexReservesResolverABI,
		Target: resolverAddress,
		Method: DRRMethodGetPoolReservesAdjusted,
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&d.poolReserves})
	addFn(&ethrpc.Call{
		ABI:    storageReadABI,
		Target: poolAddress,
		Method: SRMethodReadFromStorage,
		Params: []any{common.HexToHash("0x1")},
	}, []any{&d.dexVariables2})
}

func buildPoolState(p entity.Pool, d *rpcData, blockNumber *big.Int) (entity.Pool, error) {
	isSwapAndArbitragePaused := d.dexVariables2.Bit(255) == 1

	poolReserves := d.poolReserves
	extra := PoolExtra{
		CollateralReserves:       poolReserves.CollateralReserves,
		DebtReserves:             poolReserves.DebtReserves,
		IsSwapAndArbitragePaused: isSwapAndArbitragePaused,
		DexLimits:                poolReserves.Limits,
		CenterPrice:              poolReserves.CenterPrice,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
		return p, err
	}

	p.SwapFee = float64(poolReserves.Fee.Int64()) / FeePercentPrecision
	p.Extra = string(extraBytes)
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		GetMaxReserves(
			p.Tokens[0].Decimals,
			poolReserves.Limits.WithdrawableToken0,
			poolReserves.Limits.BorrowableToken0,
			poolReserves.CollateralReserves.Token0RealReserves,
			poolReserves.DebtReserves.Token0RealReserves).String(),
		GetMaxReserves(
			p.Tokens[1].Decimals,
			poolReserves.Limits.WithdrawableToken1,
			poolReserves.Limits.BorrowableToken1,
			poolReserves.CollateralReserves.Token1RealReserves,
			poolReserves.DebtReserves.Token1RealReserves).String(),
	}

	return p, nil
}

func GetMaxReserves(
	decimals uint8,
	withdrawableLimit TokenLimit,
	borrowableLimit TokenLimit,
	realColReserves *big.Int,
	realDebtReserves *big.Int,
) *big.Int {
	maxLimitReserves := new(big.Int).Set(borrowableLimit.ExpandsTo)
	if borrowableLimit.ExpandsTo.Cmp(withdrawableLimit.ExpandsTo) != 0 {
		maxLimitReserves.Add(maxLimitReserves, withdrawableLimit.ExpandsTo)
	}

	maxRealReserves := new(big.Int).Add(realColReserves, realDebtReserves)
	if decimals > DexAmountsDecimals {
		maxRealReserves.Mul(maxRealReserves, bignumber.TenPowInt(int8(decimals)-DexAmountsDecimals))
	} else if decimals < DexAmountsDecimals {
		maxRealReserves.Div(maxRealReserves, bignumber.TenPowInt(DexAmountsDecimals-int8(decimals)))
	}

	if maxRealReserves.Cmp(maxLimitReserves) < 0 {
		return maxRealReserves
	}
	return maxLimitReserves
}
