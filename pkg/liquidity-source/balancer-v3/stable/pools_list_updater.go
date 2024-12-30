package stable

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	sharedUpdater := shared.NewPoolsListUpdater(&shared.Config{
		DexID:           config.DexID,
		SubgraphAPI:     config.SubgraphAPI,
		SubgraphHeaders: config.SubgraphHeaders,
		NewPoolLimit:    config.NewPoolLimit,
	})

	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		sharedUpdater: sharedUpdater,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexId":   u.config.DexID,
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	subgraphPools, newMetadataBytes, err := u.sharedUpdater.GetNewPools(ctx, metadataBytes)
	if err != nil {
		return nil, nil, err
	}

	vaults, poolVersions, err := u.getPoolInfos(ctx, subgraphPools)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(subgraphPools, vaults, poolVersions)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getPoolInfos(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]string, []int, error) {
	var (
		vaultAddresses = make([]common.Address, len(subgraphPools))
		vaults         = make([]string, len(subgraphPools))
		poolInfos      = make([]string, len(subgraphPools))
		poolVersions   = make([]int, len(subgraphPools))
	)

	req := u.ethrpcClient.R().SetContext(ctx)
	for idx, subgraphPool := range subgraphPools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: subgraphPool.Address,
			Method: shared.PoolMethodGetVault,
		}, []interface{}{&vaultAddresses[idx]})
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: subgraphPool.Address,
			Method: shared.PoolMethodVersion,
		}, []interface{}{&poolInfos[idx]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Errorf("failed to getPoolInfos: %v", err)
		return nil, nil, err
	}

	for idx, addr := range vaultAddresses {
		var poolInfo shared.PoolInfo
		err := json.Unmarshal([]byte(poolInfos[idx]), &poolInfo)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexId":   u.config.DexID,
				"dexType": DexType,
			}).Warnf("invalid pool version data, fallback to %v", err)

			poolInfo.Version = shared.PoolVersion1 // temporary
		}

		poolVersions[idx] = poolInfo.Version
		vaults[idx] = strings.ToLower(addr.Hex())
	}

	return vaults, poolVersions, nil
}

func (u *PoolsListUpdater) initPools(subgraphPools []*shared.SubgraphPool, vaults []string, poolVersions []int) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))
	for idx := range subgraphPools {
		pool, err := u.initPool(subgraphPools[idx], vaults[idx], poolVersions[idx])
		if err != nil {
			return nil, err
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) initPool(subgraphPool *shared.SubgraphPool, vault string, poolVersion int) (entity.Pool, error) {
	var (
		poolTokens     = make([]*entity.PoolToken, len(subgraphPool.Tokens))
		reserves       = make([]string, len(subgraphPool.Tokens))
		scalingFactors = make([]*uint256.Int, len(subgraphPool.Tokens))
		err            error
	)

	for j, token := range subgraphPool.Tokens {
		scalingFactors[j], err = uint256.FromDecimal(token.ScalingFactor)
		if err != nil {
			return entity.Pool{}, err
		}

		poolTokens[j] = &entity.PoolToken{
			Address:   token.Address,
			Weight:    1,
			Swappable: true,
		}

		reserves[j] = "0"
	}

	staticExtraBytes, err := json.Marshal(&StaticExtra{
		PoolType:    PoolType,
		PoolVersion: poolVersion,
		Vault:       vault,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(&Extra{
		DecimalScalingFactors: scalingFactors,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     subgraphPool.Address,
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Tokens:      poolTokens,
		Reserves:    reserves,
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
	}, nil
}
