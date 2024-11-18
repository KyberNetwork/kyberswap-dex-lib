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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	extra := PoolExtra{
		CollateralReserves:       collateralReserves,
		DebtReserves:             debtReserves,
		IsSwapAndArbitragePaused: isSwapAndArbitragePaused,
		DexLimits:                poolReserves.Limits,
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
		getMaxReserves(
			p.Tokens[0].Decimals,
			poolReserves.Limits.WithdrawableToken0,
			poolReserves.Limits.BorrowableToken0,
			poolReserves.CollateralReserves.Token0RealReserves,
			poolReserves.DebtReserves.Token0RealReserves).String(),
		getMaxReserves(
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
) (*PoolWithReserves, bool, uint64, error) {
	pool := &PoolWithReserves{}

	dexVariables2 := bignumber.ZeroBI

	req := t.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    dexReservesResolverABI,
		Target: t.config.DexReservesResolver,
		Method: DRRMethodGetPoolReservesAdjusted,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&pool})

	req.AddCall(&ethrpc.Call{
		ABI:    storageReadABI,
		Target: poolAddress,
		Method: SRMethodReadFromStorage,
		Params: []interface{}{common.HexToHash("0x1")}, // slot 1
	}, []interface{}{&dexVariables2})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get pool reserves")

		return nil, false, 0, err
	}

	isSwapAndArbitragePaused := dexVariables2.Rsh(dexVariables2, 255).Cmp(bignumber.One) == 0

	return pool, isSwapAndArbitragePaused, resp.BlockNumber.Uint64(), nil
}

func getMaxReserves(
	decimals uint8,
	withdrawableLimit TokenLimit,
	borrowableLimit TokenLimit,
	realColReserves *big.Int,
	realDebtReserves *big.Int,
) *big.Int {
	// max available reserves: the smaller possible value between real reserves and the expandTo limits
	// the expandTo limits include liquidity layer balances, utilization limits, withdrawable and borrowable limits

	// if expandTo for borrowable and withdrawable match, that means they are a hard limit like liquidity layer balance
	// or utilization limit. In that case expandTo can not be summed up. Otherwise it's the case of expanding withdrawal
	// and borrow limits, for which we must sum up the max available reserve amount.
	maxLimitReserves := new(big.Int).Add(borrowableLimit.ExpandsTo, withdrawableLimit.ExpandsTo)
	if borrowableLimit.ExpandsTo.Cmp(withdrawableLimit.ExpandsTo) == 0 {
		maxLimitReserves.Set(borrowableLimit.ExpandsTo)
	}

	maxRealReserves := new(big.Int).Add(realColReserves, realDebtReserves)
	if decimals > DexAmountsDecimals {
		maxRealReserves.Mul(maxRealReserves, bignumber.TenPowInt(int8(decimals)-DexAmountsDecimals))
	} else if decimals < DexAmountsDecimals {
		maxRealReserves.Div(maxRealReserves, bignumber.TenPowInt(DexAmountsDecimals-int8(decimals)))
	}

	var maxReserve = maxLimitReserves
	if maxRealReserves.Cmp(maxLimitReserves) < 0 {
		maxReserve = maxRealReserves
	}

	return maxReserve
}
