package hiddenocean

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Info("Start getting new pools")

	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			logger.WithFields(logger.Fields{
				"dex_id": u.config.DexID,
				"error":  err,
			}).Warn("failed to unmarshal metadata, starting from offset 0")
		}
	}

	// Get total pool count from registry
	var poolCount *big.Int
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: u.config.RegistryAddress,
		Method: registryMethodPoolCount,
	}, []any{&poolCount}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"dex_id": u.config.DexID,
			"error":  err,
		}).Error("failed to get pool count from registry")
		return nil, metadataBytes, err
	}

	totalPools := int(poolCount.Int64())
	offset := metadata.Offset
	batchSize := u.getBatchSize(totalPools, offset)

	if batchSize == 0 {
		return nil, metadataBytes, nil
	}

	// Fetch pool info for each index in the batch
	poolInfoList := make([]RegistryPoolInfo, batchSize)
	getPoolReq := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := range batchSize {
		getPoolReq.AddCall(&ethrpc.Call{
			ABI:    registryABI,
			Target: u.config.RegistryAddress,
			Method: registryMethodGetPool,
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&poolInfoList[i]})
	}

	resp, err := getPoolReq.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex_id": u.config.DexID,
			"error":  err,
		}).Error("failed to fetch pool info from registry")
		return nil, metadataBytes, err
	}

	// Build entity.Pool list from successful calls
	var pools []entity.Pool
	for i, success := range resp.Result {
		if !success {
			continue
		}

		info := poolInfoList[i]
		poolAddr := hexutil.Encode(info.Pool[:])
		token0 := hexutil.Encode(info.Token0[:])
		token1 := hexutil.Encode(info.Token1[:])

		pools = append(pools, entity.Pool{
			Address:   poolAddr,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
		})
	}

	// Update metadata with new offset
	newOffset := offset + batchSize
	newMetadataBytes, err := json.Marshal(Metadata{Offset: newOffset})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dex_id":      u.config.DexID,
		"pools_len":   len(pools),
		"offset":      offset,
		"total_pools": totalPools,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getBatchSize(totalPools int, offset int) int {
	remaining := totalPools - offset
	if remaining <= 0 {
		return 0
	}

	limit := u.config.NewPoolLimit
	if limit <= 0 {
		limit = defaultNewPoolLimit
	}

	if remaining < limit {
		return remaining
	}

	return limit
}
