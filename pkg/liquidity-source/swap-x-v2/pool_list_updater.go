package swapxv2

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

	poolFactoryData, err := u.getPoolFactoryData(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getPoolFactoryData failed")

		return nil, metadataBytes, err
	}

	if poolFactoryData.IsPaused {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Info("factory is paused")
		return nil, metadataBytes, nil
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	batchSize := getBatchSize(int(poolFactoryData.AllPairsLength.Int64()), u.config.NewPoolLimit, offset)

	poolAddresses, err := u.listPoolAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPoolAddresses failed")

		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, poolAddresses, poolFactoryData)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(pools),
				"offset":      offset,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

// getPoolFactoryData gets number of pairs from the factory contracts
func (u *PoolsListUpdater) getPoolFactoryData(ctx context.Context) (velodromev2.PoolFactoryData, error) {
	pairFactoryData := velodromev2.PoolFactoryData{}

	getAllPairsLengthRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodIsPaused,
		Params: nil,
	}, []interface{}{&pairFactoryData.IsPaused})

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodAllPairsLength,
		Params: nil,
	}, []interface{}{&pairFactoryData.AllPairsLength})

	if _, err := getAllPairsLengthRequest.TryBlockAndAggregate(); err != nil {
		return velodromev2.PoolFactoryData{}, err
	}

	return pairFactoryData, nil
}

// getOffset gets index of the last pair that is fetched
func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

// listPoolAddresses lists address of pairs from offset
func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	listPoolAddressesResult := make([]common.Address, batchSize)

	listPoolAddressesRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < batchSize; i++ {
		index := big.NewInt(int64(offset + i))

		listPoolAddressesRequest.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodAllPairs,
			Params: []interface{}{index},
		}, []interface{}{&listPoolAddressesResult[i]})
	}

	resp, err := listPoolAddressesRequest.TryAggregate()
	if err != nil {
		return nil, err
	}

	var poolAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if !isSuccess {
			continue
		}

		poolAddresses = append(poolAddresses, listPoolAddressesResult[i])
	}

	return poolAddresses, nil
}

// initPools fetches token data and initializes pools
func (u *PoolsListUpdater) initPools(
	ctx context.Context,
	poolAddresses []common.Address,
	poolFactoryData velodromev2.PoolFactoryData,
) ([]entity.Pool, error) {
	metadataList, stableFee, volatileFee, blockNumber, err := u.listPoolData(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))
	for i, poolAddress := range poolAddresses {
		var fee = volatileFee
		if metadataList[i].St {
			fee = stableFee
		}

		extra, err := u.newExtra(poolFactoryData.IsPaused, fee)
		if err != nil {
			logger.
				WithFields(logger.Fields{"pool_address": poolAddress, "dex_id": u.config.DexID, "err": err}).
				Error("newExtra failed")
			continue
		}

		staticExtra, err := u.newStaticExtra(metadataList[i])
		if err != nil {
			logger.
				WithFields(logger.Fields{"pool_address": poolAddress, "dex_id": u.config.DexID, "err": err}).
				Error("newStaticExtra failed")
			continue
		}

		newPool := entity.Pool{
			Address:     strings.ToLower(poolAddress.Hex()),
			Exchange:    u.config.DexID,
			Type:        DexType,
			BlockNumber: blockNumber.Uint64(),
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{metadataList[i].R0.String(), metadataList[i].R1.String()},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(metadataList[i].T0.String()),
					Swappable: true,
				},
				{
					Address:   strings.ToLower(metadataList[i].T1.String()),
					Swappable: true,
				},
			},
			Extra:       string(extra),
			StaticExtra: string(staticExtra),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

// listPairTokens receives list of pair addresses and returns their token0 and token1
func (u *PoolsListUpdater) listPoolData(
	ctx context.Context,
	poolAddresses []common.Address,
) ([]velodromev2.PoolMetadata, *big.Int, *big.Int, *big.Int, error) {
	var (
		stableFee, volatileFee *big.Int

		poolMetadataList = make([]velodromev2.PoolMetadata, len(poolAddresses))
	)

	listPoolMetadataRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	listPoolMetadataRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodStableFee,
		Params: []interface{}{},
	}, []interface{}{&stableFee})
	listPoolMetadataRequest.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodVolatileFee,
		Params: []interface{}{},
	}, []interface{}{&volatileFee})

	for i, poolAddress := range poolAddresses {
		listPoolMetadataRequest.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodMetadata,
			Params: nil,
		}, []interface{}{&poolMetadataList[i]})
	}

	resp, err := listPoolMetadataRequest.Aggregate()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return poolMetadataList, stableFee, volatileFee, resp.BlockNumber, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func (u *PoolsListUpdater) newExtra(isPaused bool, fee *big.Int) ([]byte, error) {
	extra := velodromev2.PoolExtra{
		IsPaused: isPaused,
		Fee:      fee.Uint64(),
	}

	return json.Marshal(extra)
}

func (u *PoolsListUpdater) newStaticExtra(poolMetadata velodromev2.PoolMetadata) ([]byte, error) {
	decimal0, overflow := uint256.FromBig(poolMetadata.Dec0)
	if overflow {
		return nil, errors.New("dec0 overflow")
	}

	decimal1, overflow := uint256.FromBig(poolMetadata.Dec1)
	if overflow {
		return nil, errors.New("dec1 overflow")
	}

	staticExtra := velodromev2.PoolStaticExtra{
		FeePrecision: u.config.FeePrecision,
		Decimal0:     decimal0,
		Decimal1:     decimal1,
		Stable:       poolMetadata.St,
	}

	return json.Marshal(staticExtra)
}

// getBatchSize
// @params length number of pairs (factory tracked)
// @params limit number of pairs to be fetched in one run
// @params offset index of the last pair has been fetched
// @returns batchSize
func getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		return max(length-offset, 0)
	}

	return limit
}
