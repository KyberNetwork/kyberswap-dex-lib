package velodromev1

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
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		feeTracker   IFeeTracker
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
		feeTracker:   NewGenericFeeTracker(ethrpcClient, cfg.FeeTracker),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

	pairFactoryData, err := u.getPairFactoryData(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getPairFactoryData failed")

		return nil, metadataBytes, err
	}

	if pairFactoryData.IsPaused {
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

	batchSize := u.getBatchSize(int(pairFactoryData.AllPairsLength.Int64()), u.config.NewPoolLimit, offset)

	pairAddresses, err := u.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPairAddresses failed")

		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, pairAddresses, pairFactoryData)
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

// getPairFactoryData gets number of pairs from the factory contracts
func (u *PoolsListUpdater) getPairFactoryData(ctx context.Context) (PairFactoryData, error) {
	pairFactoryData := PairFactoryData{}
	_, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    pairFactoryABI,
		Target: u.config.FactoryAddress,
		Method: pairFactoryMethodIsPaused,
	}, []any{&pairFactoryData.IsPaused}).AddCall(&ethrpc.Call{
		ABI:    pairFactoryABI,
		Target: u.config.FactoryAddress,
		Method: pairFactoryMethodAllPairsLength,
	}, []any{&pairFactoryData.AllPairsLength}).TryBlockAndAggregate()
	return pairFactoryData, err
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

// listPairAddresses lists address of pairs from offset
func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	listPairAddressesResult := make([]common.Address, batchSize)

	listPairAddressesRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < batchSize; i++ {
		index := big.NewInt(int64(offset + i))

		listPairAddressesRequest.AddCall(&ethrpc.Call{
			ABI:    pairFactoryABI,
			Target: u.config.FactoryAddress,
			Method: pairFactoryMethodAllPairs,
			Params: []any{index},
		}, []any{&listPairAddressesResult[i]})
	}

	resp, err := listPairAddressesRequest.TryAggregate()
	if err != nil {
		return nil, err
	}

	var pairAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if !isSuccess {
			continue
		}

		pairAddresses = append(pairAddresses, listPairAddressesResult[i])
	}

	return pairAddresses, nil
}

// initPools fetches token data and initializes pools
func (u *PoolsListUpdater) initPools(
	ctx context.Context,
	pairAddresses []common.Address,
	pairFactoryData PairFactoryData,
) ([]entity.Pool, error) {
	metadataList, blockNumber, err := u.listMetadata(ctx, pairAddresses)
	if err != nil {
		return nil, err
	}

	if u.feeTracker != nil {
		req := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
		for i, pairAddress := range pairAddresses {
			u.feeTracker.AddGetFeeCall(req, u.config.FactoryAddress, pairAddress.Hex(), metadataList[i].St,
				&metadataList[i].Fee)
		}
		if _, err = req.Aggregate(); err != nil {
			return nil, err
		}
	} else {
		for i := range metadataList {
			metadataList[i].Fee = u.config.Fee
		}
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))
	for i, pairAddress := range pairAddresses {
		staticExtra, err := u.newStaticExtra(metadataList[i])
		if err != nil {
			logger.
				WithFields(logger.Fields{"pair_address": pairAddress, "dex_id": u.config.DexID, "err": err}).
				Error("newStaticExtra failed")
			continue
		}

		extra, err := u.newExtra(metadataList[i], pairFactoryData)
		if err != nil {
			logger.
				WithFields(logger.Fields{"pair_address": pairAddress, "dex_id": u.config.DexID, "err": err}).
				Error("newExtra failed")
			continue
		}

		newPool := entity.Pool{
			Address:     strings.ToLower(pairAddress.Hex()),
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

// listMetadata retrieves pool metadata and block number for given pair addresses
func (u *PoolsListUpdater) listMetadata(ctx context.Context, pairAddresses []common.Address) ([]PairMetadata, *big.Int,
	error) {
	metadataList := make([]PairMetadata, len(pairAddresses))

	listMetadataRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, pairAddress := range pairAddresses {
		listMetadataRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: pairMethodMetadata,
		}, []any{&metadataList[i]})
	}

	resp, err := listMetadataRequest.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return metadataList, resp.BlockNumber, nil
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

func (u *PoolsListUpdater) newStaticExtra(pairMetadata PairMetadata) ([]byte, error) {
	decimal0, overflow := uint256.FromBig(pairMetadata.Dec0)
	if overflow {
		return nil, errors.New("dec0 overflow")
	}

	decimal1, overflow := uint256.FromBig(pairMetadata.Dec1)
	if overflow {
		return nil, errors.New("dec1 overflow")
	}

	staticExtra := PoolStaticExtra{
		FeePrecision: u.config.FeePrecision,
		Decimal0:     decimal0,
		Decimal1:     decimal1,
		Stable:       pairMetadata.St,
	}

	return json.Marshal(staticExtra)
}

func (u *PoolsListUpdater) newExtra(pairMetadata PairMetadata, factoryData PairFactoryData) ([]byte, error) {
	return json.Marshal(PoolExtra{
		IsPaused: factoryData.IsPaused,
		Fee:      pairMetadata.Fee,
	})
}

// getBatchSize
// @params length number of pairs (factory tracked)
// @params limit number of pairs to be fetched in one run
// @params offset index of the last pair has been fetched
// @returns batchSize
func (u *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		if offset > length {
			logger.WithFields(logger.Fields{
				"dex":    u.config.DexID,
				"offset": offset,
				"length": length,
			}).Warn("[getBatchSize] offset is greater than length")
		}
		return max(length-offset, 0)
	}

	return limit
}
