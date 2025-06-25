package aavev3

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	ILogDecoder interface {
		Decode(logs []types.Log) (ReserveData, *big.Int, error)
	}

	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logDecoder   ILogDecoder
	}

	GetReserveDataResult struct {
		Configuration               uint256.Int
		LiquidityIndex              uint256.Int
		VariableBorrowIndex         uint256.Int
		CurrentLiquidityRate        uint256.Int
		CurrentVariableBorrowRate   uint256.Int
		CurrentStableBorrowRate     uint256.Int
		LastUpdateTimestamp         uint64
		ATokenAddress               [20]byte
		StableDebtTokenAddress      [20]byte
		VariableDebtTokenAddress    [20]byte
		InterestRateStrategyAddress [20]byte
		Id                          uint8
	}

	GetATokenBalanceResult struct {
		Balance *big.Int
	}

	GetVariableDebtBalanceResult struct {
		Balance *big.Int
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	tracker := &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   NewLogDecoder(),
	}
	return tracker, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	reserveData, blockNumber, err := d.getReserveData(ctx, p.Address, params.Logs)
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

	// Get AToken and VariableDebtToken balances
	aTokenBalance, variableDebtBalance, err := d.getTokenBalances(ctx, p.Address, reserveData)
	if err != nil {
		return p, err
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      []string{aTokenBalance.String(), variableDebtBalance.String()},
				"old_block_number": p.BlockNumber,
				"new_block_number": blockNumber,
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	return d.updatePool(p, reserveData, aTokenBalance, variableDebtBalance, blockNumber)
}

func (d *PoolTracker) getReserveData(ctx context.Context, poolAddress string, logs []types.Log) (ReserveData, *big.Int, error) {
	reserveData, blockNumber, err := d.getReserveDataFromLogs(logs)
	if err != nil {
		return d.getReserveDataFromRPCNode(ctx, poolAddress)
	}

	// if reserveData.Configuration.IsZero() {
	// 	return d.getReserveDataFromRPCNode(ctx, poolAddress)
	// }

	return reserveData, blockNumber, nil
}

func (d *PoolTracker) getTokenBalances(ctx context.Context, assetAddress string, reserveData ReserveData) (*big.Int, *big.Int, error) {
	var (
		aTokenBalance       GetATokenBalanceResult
		variableDebtBalance GetVariableDebtBalanceResult
	)

	getBalancesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Get AToken total supply (represents total supplied liquidity)
	getBalancesRequest.AddCall(&ethrpc.Call{
		ABI: atokenABI,
		// Target: common.BytesToAddress(reserveData.ATokenAddress[:]).Hex(),
		Method: atokenMethodTotalSupply,
		Params: nil,
	}, []interface{}{&aTokenBalance.Balance})

	// Get VariableDebtToken total supply (represents total borrowed liquidity)
	getBalancesRequest.AddCall(&ethrpc.Call{
		ABI: variableDebtTokenABI,
		// Target: common.BytesToAddress(reserveData.VariableDebtTokenAddress[:]).Hex(),
		Method: variableDebtTokenMethodTotalSupply,
		Params: nil,
	}, []interface{}{&variableDebtBalance.Balance})

	_, err := getBalancesRequest.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return aTokenBalance.Balance, variableDebtBalance.Balance, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, reserveData ReserveData, aTokenBalance, variableDebtBalance, blockNumber *big.Int) (entity.Pool, error) {
	extra := Extra{
		// PoolAddress:              d.config.PoolAddress,
		// 	ATokenAddress:            common.BytesToAddress(reserveData.ATokenAddress[:]).Hex(),
		// 			VariableDebtTokenAddress: common.BytesToAddress(reserveData.VariableDebtTokenAddress[:]).Hex(),
		// 			StableDebtTokenAddress:   common.BytesToAddress(reserveData.StableDebtTokenAddress[:]).Hex(),
		// LiquidityIndex:            reserveData.LiquidityIndex.ToBig(),
		// VariableBorrowIndex:       reserveData.VariableBorrowIndex.ToBig(),
		// CurrentLiquidityRate:      reserveData.CurrentLiquidityRate.ToBig(),
		// CurrentVariableBorrowRate: reserveData.CurrentVariableBorrowRate.ToBig(),
		// CurrentStableBorrowRate:   reserveData.CurrentStableBorrowRate.ToBig(),
		// LastUpdateTimestamp: uint64(reserveData.LastUpdateTimestamp),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Reserves = entity.PoolReserves{
		aTokenBalance.String(),
		variableDebtBalance.String(),
	}
	pool.Extra = string(extraBytes)
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	return pool, nil
}

func (d *PoolTracker) getReserveDataFromRPCNode(ctx context.Context, assetAddress string) (ReserveData, *big.Int, error) {
	var getReserveDataResult GetReserveDataResult

	getReserveDataRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	getReserveDataRequest.AddCall(&ethrpc.Call{
		ABI:    aaveV3PoolABI,
		Target: d.config.PoolAddress,
		Method: poolMethodGetReserveData,
		Params: []interface{}{assetAddress},
	}, []interface{}{&getReserveDataResult})

	resp, err := getReserveDataRequest.TryBlockAndAggregate()
	if err != nil {
		return ReserveData{}, nil, err
	}

	return ReserveData{
		// Configuration:               getReserveDataResult.Configuration,
		// LiquidityIndex:              getReserveDataResult.LiquidityIndex,
		// VariableBorrowIndex:         getReserveDataResult.VariableBorrowIndex,
		// CurrentLiquidityRate:        getReserveDataResult.CurrentLiquidityRate,
		// CurrentVariableBorrowRate:   getReserveDataResult.CurrentVariableBorrowRate,
		// CurrentStableBorrowRate:     getReserveDataResult.CurrentStableBorrowRate,
		// LastUpdateTimestamp:         getReserveDataResult.LastUpdateTimestamp,
		// ATokenAddress:               getReserveDataResult.ATokenAddress,
		// StableDebtTokenAddress:      getReserveDataResult.StableDebtTokenAddress,
		// VariableDebtTokenAddress:    getReserveDataResult.VariableDebtTokenAddress,
		// InterestRateStrategyAddress: getReserveDataResult.InterestRateStrategyAddress,
		// Id:                          getReserveDataResult.Id,
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) getReserveDataFromLogs(logs []types.Log) (ReserveData, *big.Int, error) {
	if len(logs) == 0 {
		return ReserveData{}, nil, nil
	}

	return d.logDecoder.Decode(logs)
}
