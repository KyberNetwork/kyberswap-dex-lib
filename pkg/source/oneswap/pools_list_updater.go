package oneswap

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

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
	if len(metadataBytes) > 0 {
		if err := sonic.Unmarshal(metadataBytes, &metadata); err != nil {
			logger.WithFields(logger.Fields{
				"metadataBytes": metadataBytes,
				"error":         err,
			}).Errorf("failed to unmarshal metadataBytes")
			return nil, metadataBytes, err
		}
	}

	var lengthBI *big.Int
	if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    oneSwapFactoryABI,
		Target: d.config.FactoryAddress,
		Method: poolFactoryMethodAllPoolsLength,
		Params: nil,
	}, []interface{}{&lengthBI}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get all pool length")
		return nil, metadataBytes, err
	}

	totalPools := int(lengthBI.Int64())
	currentOffset := metadata.Offset
	batchSize := d.config.NewPoolLimit
	if currentOffset+batchSize > totalPools {
		batchSize = totalPools - currentOffset
		if batchSize <= 0 {
			return nil, metadataBytes, nil
		}
	}

	poolAddresses := make([]common.Address, totalPools)
	poolAddressRequest := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		poolAddressRequest.AddCall(&ethrpc.Call{
			ABI:    oneSwapFactoryABI,
			Target: d.config.FactoryAddress,
			Method: poolFactoryMethodAllPools,
			Params: []interface{}{big.NewInt(int64(currentOffset + i))},
		}, []interface{}{&poolAddresses[i]})
	}
	if _, err := poolAddressRequest.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool address")
		return nil, metadataBytes, err
	}

	pools, err := d.processBatch(ctx, poolAddresses)
	if err != nil {
		return nil, metadataBytes, err
	}

	numPools := len(pools)
	newMetadataBytes, err := sonic.Marshal(Metadata{
		Offset: currentOffset + numPools,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("got %v Oneswap pools", numPools)

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	var (
		precisionMultipliers = make([][]*big.Int, len(poolAddresses))
		poolTokens           = make([][]common.Address, len(poolAddresses))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		calls.AddCall(&ethrpc.Call{
			ABI:    oneSwapABI,
			Target: poolAddress.Hex(),
			Method: poolMethodGetTokenPrecisionMultipliers,
			Params: nil,
		}, []interface{}{&precisionMultipliers[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    oneSwapABI,
			Target: poolAddress.Hex(),
			Method: poolMethodGetPoolTokens,
			Params: nil,
		}, []interface{}{&poolTokens[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate to get pool data")
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))
	for i, poolAddress := range poolAddresses {
		var tokens = make([]*entity.PoolToken, 0, len(poolTokens[i]))
		var reserves = make([]string, 0, len(poolTokens[i])+1)
		var staticExtra StaticExtra

		for j := 0; j < len(poolTokens[i]); j++ {
			tokenAddress := strings.ToLower(poolTokens[i][j].Hex())
			tokenModel := entity.PoolToken{
				Address:   tokenAddress,
				Weight:    defaultWeight,
				Swappable: true,
			}
			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, zeroString)
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precisionMultipliers[i][j].String())
		}
		reserves = append(reserves, zeroString) // for totalSupply
		staticExtraBytes, err := sonic.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": poolAddresses,
				"error":       err,
			}).Errorf("failed to marshal static extra data")
			return nil, err
		}
		var newPool = entity.Pool{
			Address:     strings.ToLower(poolAddress.Hex()),
			Exchange:    d.config.DexID,
			Type:        DexTypeOneSwap,
			StaticExtra: string(staticExtraBytes),
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
