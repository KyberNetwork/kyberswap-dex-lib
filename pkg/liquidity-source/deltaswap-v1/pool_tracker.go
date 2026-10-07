package deltaswapv1

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type dsFeeInfo struct {
	DsFee          uint8
	DsFeeThreshold uint8
}

type tradeLiquidityEMAResult struct {
	TradeLiquidityEMA     *uint256.Int
	LastTradeLiquiditySum *uint256.Int
	LastTradeBlockNumber  uint32
}

type liquidityEMAResult struct {
	LiquidityEMA             *uint256.Int
	LastLiquidityBlockNumber uint32
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
		dsFeeInfoData         dsFeeInfo
		reservesResult        uniswapv2.ReserveData
		tradeLiquidityEMAData tradeLiquidityEMAResult
		liquidityEMAData      liquidityEMAResult
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1FactoryABI,
		Target: t.config.FactoryAddress,
		Method: factoryMethodDsFeeInfo,
	}, []any{&dsFeeInfoData})
	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []any{&reservesResult})
	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1PairABI,
		Target: p.Address,
		Method: factoryMethodGetTradeLiquidityEMAParams,
	}, []any{&tradeLiquidityEMAData})
	calls.AddCall(&ethrpc.Call{
		ABI:    deltaSwapV1PairABI,
		Target: p.Address,
		Method: factoryMethodGetLiquidityEMA,
	}, []any{&liquidityEMAData})

	resp, err := calls.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		DsFee:                    dsFeeInfoData.DsFee,
		DsFeeThreshold:           dsFeeInfoData.DsFeeThreshold,
		LiquidityEMA:             liquidityEMAData.LiquidityEMA,
		LastLiquidityBlockNumber: uint64(liquidityEMAData.LastLiquidityBlockNumber),
		TradeLiquidityEMA:        tradeLiquidityEMAData.TradeLiquidityEMA,
		LastTradeLiquiditySum:    tradeLiquidityEMAData.LastTradeLiquiditySum,
		LastTradeBlockNumber:     uint64(tradeLiquidityEMAData.LastTradeBlockNumber),
	})

	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reservesResult.Reserve0.String(), reservesResult.Reserve1.String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
