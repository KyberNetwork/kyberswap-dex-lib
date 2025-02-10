package deltaswapv1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type NomiStableReserve struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
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
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		dsFeeInfoTuple          [2]interface{} // dsFee, dsFeeThreshold uint8
		reservesResult          uniswapv2.GetReservesResult
		tradeLiquidityEMAParams [3]interface{} // tradeLiquidityEMA, lastTradeLiquiditySum uint112, lastTradeBlockNumber uint32
		liquidityEMA            [2]interface{} // liquidityEMA uint112, lastLiquidityBlockNumber uint32
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1FactoryABI,
		Target: t.config.FactoryAddress,
		Method: factoryMethodDsFeeInfo,
	}, []interface{}{&dsFeeInfoTuple})
	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []interface{}{&reservesResult})
	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1PairABI,
		Target: p.Address,
		Method: factoryMethodGetTradeLiquidityEMAParams,
	}, []interface{}{&tradeLiquidityEMAParams})
	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1PairABI,
		Target: p.Address,
		Method: factoryMethodGetLiquidityEMA,
	}, []interface{}{&liquidityEMA})

	resp, err := calls.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		DsFee:                    dsFeeInfoTuple[0].(uint8),
		DsFeeThreshold:           dsFeeInfoTuple[1].(uint8),
		LiquidityEMA:             uint256.MustFromBig(liquidityEMA[0].(*big.Int)),
		LastLiquidityBlockNumber: uint64(liquidityEMA[1].(uint32)),
		TradeLiquidityEMA:        uint256.MustFromBig(tradeLiquidityEMAParams[0].(*big.Int)),
		LastTradeLiquiditySum:    uint256.MustFromBig(tradeLiquidityEMAParams[1].(*big.Int)),
		LastTradeBlockNumber:     uint64(tradeLiquidityEMAParams[2].(uint32)),
	})

	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	p.Reserves = entity.PoolReserves{reservesResult.Reserve0.String(), reservesResult.Reserve1.String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
