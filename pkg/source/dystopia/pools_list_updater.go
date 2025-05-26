package dystopia

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexTypeDystopia, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to unmarshal metadata")

			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	var lengthBI *big.Int
	if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: poolFactoryMethodAllPairsLength,
		Params: nil,
	}, []interface{}{&lengthBI}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get number of pools from factory")

		return nil, metadataBytes, err
	}
	totalNumberOfPools := int(lengthBI.Int64())

	batchSize := d.config.NewPoolLimit
	currentOffset := metadata.Offset
	if currentOffset+batchSize > totalNumberOfPools {
		batchSize = totalNumberOfPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	getPoolAddressRequest := d.ethrpcClient.NewRequest().SetContext(ctx)
	var poolAddresses = make([]common.Address, batchSize)
	for i := 0; i < batchSize; i++ {
		getPoolAddressRequest.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: d.config.FactoryAddress,
			Method: poolFactoryMethodAllPairs,
			Params: []interface{}{big.NewInt(int64(currentOffset + i))},
		}, []interface{}{&poolAddresses[i]})
	}
	if _, err := getPoolAddressRequest.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool address")

		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, poolAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to process update new pool")

		return nil, metadataBytes, err
	}

	if len(pools) > 0 {
		logger.WithFields(logger.Fields{
			"dexID":                     d.config.DexID,
			"batchSize":                 batchSize,
			"totalNumberOfUpdatedPools": currentOffset + len(pools),
			"totalNumberOfPools":        totalNumberOfPools,
		}).Infof("scan DystopiaFactory")
	}

	nextOffset := currentOffset + len(pools)
	newMetadataBytes, err := json.Marshal(Metadata{
		Offset: nextOffset,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to json marshal metadata")

		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil

}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		limit        = len(poolAddresses)
		poolMetadata = make([]DystopiaMetadata, limit)
		pools        = make([]entity.Pool, 0, limit)
	)

	calls := d.ethrpcClient.NewRequest()
	calls.SetContext(ctx)

	for i := 0; i < limit; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodMetadata,
			Params: nil,
		}, []interface{}{&poolMetadata[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate to get tokens from pool")

		return nil, err
	}

	for i, pAddr := range poolAddresses {
		poolAddress := strings.ToLower(pAddr.Hex())
		token0Address := strings.ToLower(poolMetadata[i].T0.Hex())
		token1Address := strings.ToLower(poolMetadata[i].T1.Hex())

		var token0 = entity.PoolToken{
			Address:   token0Address,
			Decimals:  uint8(len(poolMetadata[i].Dec0.String()) - 1),
			Swappable: true,
		}

		var token1 = entity.PoolToken{
			Address:   token1Address,
			Decimals:  uint8(len(poolMetadata[i].Dec1.String()) - 1),
			Swappable: true,
		}

		staticExtra := StaticExtra{
			Stable: poolMetadata[i].St,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error":       err,
				"poolAddress": pAddr,
			}).Errorf("failed to json marshal staticExtra")

			return nil, err
		}

		newPool := entity.Pool{
			Address:     poolAddress,
			SwapFee:     d.config.SwapFee,
			Exchange:    d.config.DexID,
			Type:        DexTypeDystopia,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{reserveZero, reserveZero},
			Tokens:      []*entity.PoolToken{&token0, &token1},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
