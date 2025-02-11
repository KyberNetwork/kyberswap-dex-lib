package ethervista

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
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	GetReservesResult struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}
func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	rpcStateData, blockNumber, err := d.getRPCState(ctx, p.Address, overrides)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": blockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      entity.PoolReserves{rpcStateData.Reserve0.String(), rpcStateData.Reserve1.String()},
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return d.updatePool(p, rpcStateData, blockNumber)
}

func (d *PoolTracker) updatePool(pool entity.Pool, rpcStateData RPCStateData, blockNumber *big.Int) (entity.Pool, error) {
	extra := Extra{
		RouterAddress:         rpcStateData.RouterAddress,
		BuyTotalFee:           rpcStateData.BuyTotalFee,
		SellTotalFee:          rpcStateData.SellTotalFee,
		USDCToETHBuyTotalFee:  rpcStateData.USDCToETHBuyTotalFee,
		USDCToETHSellTotalFee: rpcStateData.USDCToETHSellTotalFee,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Reserves = entity.PoolReserves{
		rpcStateData.Reserve0.String(),
		rpcStateData.Reserve1.String(),
	}
	pool.Extra = string(extraBytes)
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getRPCState(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (RPCStateData, *big.Int, error) {
	var (
		getReservesResult GetReservesResult
		buyTotalFee       uint8
		sellTotalFee      uint8
		routerAddress     common.Address
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		rpcRequest.SetOverrides(overrides)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodGetReserves,
		Params: nil,
	}, []interface{}{&getReservesResult})
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodBuyTotalFee,
		Params: nil,
	}, []interface{}{&buyTotalFee})
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairMethodSellTotalFee,
		Params: nil,
	}, []interface{}{&sellTotalFee})
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodRouter,
		Params: nil,
	}, []interface{}{&routerAddress})

	resp, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		return RPCStateData{}, nil, err
	}

	var (
		usdcToETHBuyTotalFee  *big.Int
		usdcToETHSellTotalFee *big.Int
	)

	rpcRequest = d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		rpcRequest.SetOverrides(overrides)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: routerAddress.Hex(),
		Method: routerMethodUSDCToEth,
		Params: []interface{}{big.NewInt(int64(buyTotalFee))},
	}, []interface{}{&usdcToETHBuyTotalFee})
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: routerAddress.Hex(),
		Method: routerMethodUSDCToEth,
		Params: []interface{}{big.NewInt(int64(sellTotalFee))},
	}, []interface{}{&usdcToETHSellTotalFee})

	_, err = rpcRequest.TryBlockAndAggregate()
	if err != nil {
		return RPCStateData{}, nil, err
	}

	return RPCStateData{
		Reserve0:              getReservesResult.Reserve0,
		Reserve1:              getReservesResult.Reserve1,
		BuyTotalFee:           uint(buyTotalFee),
		SellTotalFee:          uint(sellTotalFee),
		USDCToETHBuyTotalFee:  usdcToETHBuyTotalFee,
		USDCToETHSellTotalFee: usdcToETHSellTotalFee,
		RouterAddress:         routerAddress.Hex(),
	}, resp.BlockNumber, nil
}
