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
	poolReserves, blockNumber, err := t.getPoolReserves(ctx, p.Address, overrides)
	if err != nil {
		return p, err
	}

	collateralReserves, debtReserves := poolReserves.CollateralReserves, poolReserves.DebtReserves
	extra := PoolExtra{
		CollateralReserves: collateralReserves,
		DebtReserves:       debtReserves,
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

	var reserves [2]big.Int
	reserves[0].Add(collateralReserves.Token0RealReserves, debtReserves.Token0RealReserves)
	reserves[1].Add(collateralReserves.Token1RealReserves, debtReserves.Token1RealReserves)
	for i, reserve := range reserves {
		if p.Tokens[i].Decimals > DexAmountsDecimals {
			reserve.Mul(&reserve, bignumber.TenPowInt(int8(p.Tokens[i].Decimals)-DexAmountsDecimals))
		} else if p.Tokens[i].Decimals < DexAmountsDecimals {
			reserve.Div(&reserve, bignumber.TenPowInt(DexAmountsDecimals-int8(p.Tokens[i].Decimals)))
		}
	}
	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}

	return p, nil
}

func (t *PoolTracker) getPoolReserves(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*PoolWithReserves, uint64, error) {
	pool := &PoolWithReserves{}

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
