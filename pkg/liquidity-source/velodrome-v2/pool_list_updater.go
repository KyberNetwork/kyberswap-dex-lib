package velodromev2

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
func (u *PoolsListUpdater) getPoolFactoryData(ctx context.Context) (PoolFactoryData, error) {
	pairFactoryData := PoolFactoryData{}

	getAllPairsLengthRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    poolFactoryABI,
		Target: u.config.FactoryAddress,
		Method: poolFactoryMethodIsPaused,
		Params: nil,
	}, []interface{}{&pairFactoryData.IsPaused})

	getAllPairsLengthRequest.AddCall(&ethrpc.Call{
		ABI:    poolFactoryABI,
		Target: u.config.FactoryAddress,
		Method: poolFactoryMethodAllPoolsLength,
		Params: nil,
	}, []interface{}{&pairFactoryData.AllPairsLength})

	if _, err := getAllPairsLengthRequest.TryBlockAndAggregate(); err != nil {
		return PoolFactoryData{}, err
	}

	return pairFactoryData, nil
}

// getOffset gets index of the last pair that is fetched
func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := sonic.Unmarshal(metadataBytes, &metadata); err != nil {
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
			ABI:    poolFactoryABI,
			Target: u.config.FactoryAddress,
			Method: poolFactoryMethodAllPools,
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
	poolFactoryData PoolFactoryData,
) ([]entity.Pool, error) {
	metadataList, feeList, blockNumber, err := u.listPoolData(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))
	for i, poolAddress := range poolAddresses {
		extra, err := u.newExtra(poolFactoryData.IsPaused, feeList[i])
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
			BlockNumber: blockNumber,
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
) ([]PoolMetadata, []*big.Int, uint64, error) {
	poolMetadataList := make([]PoolMetadata, len(poolAddresses))

	listPoolMetadataRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

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
		return nil, nil, 0, err
	}

	feeList := make([]*big.Int, len(poolAddresses))

	listPoolFeeRequest := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)

	for i, poolAddress := range poolAddresses {
		listPoolFeeRequest.AddCall(&ethrpc.Call{
			ABI:    poolFactoryABI,
			Target: u.config.FactoryAddress,
			Method: poolFactoryMethodGetFee,
			Params: []interface{}{poolAddress, poolMetadataList[i].St},
		}, []interface{}{&feeList[i]})
	}

	resp, err = listPoolFeeRequest.Aggregate()
	if err != nil {
		return nil, nil, 0, err
	}

	return poolMetadataList, feeList, resp.BlockNumber.Uint64(), nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	metadataBytes, err := sonic.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func (u *PoolsListUpdater) newExtra(isPaused bool, fee *big.Int) ([]byte, error) {
	extra := PoolExtra{
		IsPaused: isPaused,
		Fee:      fee.Uint64(),
	}

	return sonic.Marshal(extra)
}

func (u *PoolsListUpdater) newStaticExtra(poolMetadata PoolMetadata) ([]byte, error) {
	decimal0, overflow := uint256.FromBig(poolMetadata.Dec0)
	if overflow {
		return nil, errors.New("dec0 overflow")
	}

	decimal1, overflow := uint256.FromBig(poolMetadata.Dec1)
	if overflow {
		return nil, errors.New("dec1 overflow")
	}

	staticExtra := PoolStaticExtra{
		FeePrecision: u.config.FeePrecision,
		Decimal0:     decimal0,
		Decimal1:     decimal1,
		Stable:       poolMetadata.St,
	}

	return sonic.Marshal(staticExtra)
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
		return length - offset
	}

	return limit
}
