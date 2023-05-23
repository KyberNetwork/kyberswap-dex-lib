package curve

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolsListUpdater, error) {
	if err := initConfig(cfg, ethrpcClient); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("[Curve] failed to init poolsListUpdater")
		return nil, err
	}

	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to unmarshal metadataBytes")
			return nil, nil, err
		}
	}

	var (
		poolTypeMap           = make(map[string][]PoolAndRegistries)
		registryOrFactoryList = []struct {
			ABI     abi.ABI
			Address string
			Offset  int
		}{
			{mainRegistryABI, d.config.MainRegistryAddress, metadata.MainRegistryOffset},
			{metaPoolFactoryABI, d.config.MetaPoolsFactoryAddress, metadata.MetaFactoryOffset},
			{cryptoRegistryABI, d.config.CryptoPoolsRegistryAddress, metadata.CryptoRegistryOffset},
			{cryptoFactoryABI, d.config.CryptoPoolsFactoryAddress, metadata.CryptoFactoryOffset},
		}
	)

	var newPoolLimitLeft = d.config.NewPoolLimit
	for i := 0; i < len(registryOrFactoryList); i++ {
		poolAddresses, poolTypes, nextOffset, err := d.getNewPoolAddressesFromRegistryOrFactory(
			ctx,
			registryOrFactoryList[i].ABI,
			registryOrFactoryList[i].Address,
			registryOrFactoryList[i].Offset,
			newPoolLimitLeft,
		)
		if err != nil {
			logger.WithFields(logger.Fields{
				"address": registryOrFactoryList[i].Address,
				"offset":  registryOrFactoryList[i].Offset,
				"error":   err,
			}).Errorf("failed to get new pool addresses from the registry or factory")
			return nil, nil, err
		}
		newPoolLimitLeft = newPoolLimitLeft - (nextOffset - registryOrFactoryList[i].Offset)

		for j := 0; j < len(poolAddresses); j++ {
			poolTypeMap[poolTypes[j]] = append(poolTypeMap[poolTypes[j]], PoolAndRegistries{
				PoolAddress:              poolAddresses[j],
				RegistryOrFactoryABI:     &registryOrFactoryList[i].ABI,
				RegistryOrFactoryAddress: &registryOrFactoryList[i].Address,
			})
		}

		registryOrFactoryList[i].Offset = nextOffset
	}

	var pools []entity.Pool
	for poolType, poolAndRegistries := range poolTypeMap {
		var newPools []entity.Pool
		var err error
		switch poolType {
		case poolTypeBase:
			newPools, err = d.getNewPoolsTypeBase(ctx, poolAndRegistries)
		case poolTypePlainOracle:
			newPools, err = d.getNewPoolsTypePlainOracle(ctx, poolAndRegistries)
		case poolTypeMeta:
			newPools, err = d.getNewPoolsTypeMeta(ctx, poolAndRegistries)
		case poolTypeAave:
			newPools, err = d.getNewPoolsTypeAave(ctx, poolAndRegistries)
		case poolTypeCompound:
			newPools, err = d.getNewPoolsTypeCompound(ctx, poolAndRegistries)
		case poolTypeTwo:
			newPools, err = d.getNewPoolsTypeTwo(ctx, poolAndRegistries)
		case poolTypeTricrypto:
			newPools, err = d.getNewPoolsTypeTricrypto(ctx, poolAndRegistries)
		default:
			continue
		}
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolType": poolType,
				"error":    err,
			}).Errorf("failed to get new pools of type")
			return nil, nil, err
		}
		pools = append(pools, newPools...)
		logger.Infof("got total of %v Curve pools of %v types from registry and factory", len(newPools), poolType)
	}

	if !d.hasInitialized {
		newPools, err := d.initPool()
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to init new pool from file")
			return nil, nil, err
		}
		pools = append(pools, newPools...)
		d.hasInitialized = true
	}

	newMetaDataBytes, err := json.Marshal(Metadata{
		MainRegistryOffset:   registryOrFactoryList[0].Offset,
		MetaFactoryOffset:    registryOrFactoryList[1].Offset,
		CryptoRegistryOffset: registryOrFactoryList[2].Offset,
		CryptoFactoryOffset:  registryOrFactoryList[3].Offset,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to marshaling metadata")
		return nil, nil, err
	}

	return pools, newMetaDataBytes, nil
}

func (d *PoolsListUpdater) initPool() ([]entity.Pool, error) {
	newPoolBytes, ok := bytesByPath[d.config.PoolPath]
	if !ok {
		logger.WithFields(logger.Fields{
			"poolPath": d.config.PoolPath,
		}).Errorf("not found the pool path bytes data")
		return nil, errors.New("not found the pool path bytes data")
	}

	var poolItems []PoolItem
	if err := json.Unmarshal(newPoolBytes, &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal new pool bytes data")
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolItems))
	for i, poolItem := range poolItems {
		if len(poolItem.LpToken) == 0 {
			logger.WithFields(logger.Fields{
				"poolID": poolItem.ID,
			}).Errorf("can not find lpToken from pool item")
			return nil, errors.New("can not find lpToken from pool item")
		}

		var staticExtraBytes []byte
		switch poolItem.Type {
		case poolTypeBase:
			var staticExtra = PoolBaseStaticExtra{
				LpToken:    poolItem.LpToken,
				APrecision: poolItem.APrecision,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
				staticExtra.Rates = append(staticExtra.Rates, poolItem.Tokens[j].Rate)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)

		case poolTypePlainOracle:
			var staticExtra = PoolPlainOracleStaticExtra{
				LpToken:    poolItem.LpToken,
				APrecision: poolItem.APrecision,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)

		case poolTypeAave:
			var staticExtra = PoolAaveStaticExtra{
				LpToken:          poolItem.LpToken,
				UnderlyingTokens: poolItem.UnderlyingTokens,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)

		case poolTypeCompound:
			var staticExtra = PoolCompoundStaticExtra{
				LpToken:          poolItem.LpToken,
				UnderlyingTokens: poolItem.UnderlyingTokens,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)

		case poolTypeMeta:
			var staticExtra = PoolMetaStaticExtra{
				LpToken:          poolItem.LpToken,
				BasePool:         poolItem.BasePool,
				RateMultiplier:   poolItem.RateMultiplier,
				APrecision:       poolItem.APrecision,
				UnderlyingTokens: poolItem.UnderlyingTokens,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
				staticExtra.Rates = append(staticExtra.Rates, poolItem.Tokens[j].Rate)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)

		case poolTypeTwo:
			var staticExtra = PoolTwoStaticExtra{
				LpToken: poolItem.LpToken,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)

		case poolTypeTricrypto:
			var staticExtra = PoolTricryptoStaticExtra{
				LpToken: poolItem.LpToken,
			}
			for j := range poolItem.Tokens {
				staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			}
			staticExtraBytes, _ = json.Marshal(staticExtra)
		}

		var reserves = make(entity.PoolReserves, len(poolItem.Tokens))
		var tokens = make([]*entity.PoolToken, len(poolItem.Tokens))
		for j := 0; j < len(poolItem.Tokens); j++ {
			reserves[j] = zeroString
			if poolItem.Type == poolTypeAave {
				tokens[j] = &entity.PoolToken{
					Address:   poolItem.Tokens[j].Address,
					Weight:    defaultWeight,
					Swappable: false,
				}
			} else {
				tokens[j] = &entity.PoolToken{
					Address:   poolItem.Tokens[j].Address,
					Weight:    defaultWeight,
					Swappable: true,
				}
			}
		}

		var newPool = entity.Pool{
			Address:     poolItem.ID,
			Exchange:    DexTypeCurve,
			Type:        poolItem.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			StaticExtra: string(staticExtraBytes),
		}

		pools[i] = newPool
	}

	return pools, nil
}

func (d *PoolsListUpdater) getNewPoolAddressesFromRegistryOrFactory(
	ctx context.Context,
	registryOrFactoryABI abi.ABI,
	registryOrFactoryAddress string,
	currentOffset int,
	newPoolLimit int,
) ([]common.Address, []string, int, error) {
	poolAddresses, newOffset, err := d.getPoolAddresses(
		ctx, registryOrFactoryABI, registryOrFactoryAddress, currentOffset, newPoolLimit)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": registryOrFactoryAddress,
			"error":   err,
		}).Errorf("failed to get pool addresses")
		return nil, nil, currentOffset, err
	}

	poolTypes, err := d.classifyPoolTypes(ctx, registryOrFactoryABI, registryOrFactoryAddress, poolAddresses)
	if err != nil {
		return nil, nil, currentOffset, err
	}

	return poolAddresses, poolTypes, newOffset, nil
}

func (d *PoolsListUpdater) getPoolAddresses(
	ctx context.Context,
	registryOrFactoryABI abi.ABI,
	registryOrFactoryAddress string,
	currentOffset int,
	newPoolLimit int,
) ([]common.Address, int, error) {
	var lengthBI *big.Int
	if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    registryOrFactoryABI,
		Target: registryOrFactoryAddress,
		Method: registryOrFactoryMethodPoolCount,
		Params: nil,
	}, []interface{}{&lengthBI}).Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pool count")
		return nil, currentOffset, err
	}

	totalLength := int(lengthBI.Int64())
	batchSize := newPoolLimit
	if currentOffset+batchSize > totalLength {
		batchSize = totalLength - currentOffset
		if batchSize <= 0 {
			return nil, currentOffset, nil
		}
	}

	poolAddresses := make([]common.Address, batchSize)
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    registryOrFactoryABI,
			Target: registryOrFactoryAddress,
			Method: registryOrFactoryMethodPoolList,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&poolAddresses[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate call to get pool addresses")
		return nil, currentOffset, err
	}

	return poolAddresses, currentOffset + batchSize, nil
}
