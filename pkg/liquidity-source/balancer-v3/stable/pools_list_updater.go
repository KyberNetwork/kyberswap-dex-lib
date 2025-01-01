package stable

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

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

	if u.config.Factories == nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error("factories config is empty")

		return nil, nil, errors.New("PoolTypeByFactory config is empty")
	}

	subgraphPools, newMetadataBytes, err := u.sharedUpdater.GetNewPools(ctx, metadataBytes)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(subgraphPools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPools(subgraphPools []*shared.SubgraphPool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, subgraphPool := range subgraphPools {
		poolType, found := u.config.Factories[subgraphPool.Factory]
		if !found && subgraphPool.Factory != "" {
			logger.WithFields(logger.Fields{
				"dexId":   u.config.DexID,
				"dexType": DexType,
			}).Warnf("detected a new factory that hasn't been configured : %s", subgraphPool.Factory)
			continue
		}

		if strings.EqualFold(PoolType, poolType) {
			staticExtraBytes, err := json.Marshal(&StaticExtra{
				Vault: subgraphPool.Vault.ID,
			})
			if err != nil {
				return nil, err
			}

			var (
				poolTokens = make([]*entity.PoolToken, len(subgraphPool.Tokens))
				reserves   = make([]string, len(subgraphPool.Tokens))
			)
			for i, token := range subgraphPool.Tokens {
				poolTokens[i] = &entity.PoolToken{
					Address:   token.Address,
					Weight:    1,
					Swappable: true,
				}
				reserves[i] = "0"
			}

			pools = append(pools, entity.Pool{
				Address:     subgraphPool.Address,
				Exchange:    u.config.DexID,
				Type:        DexType,
				Timestamp:   time.Now().Unix(),
				Tokens:      poolTokens,
				Reserves:    reserves,
				StaticExtra: string(staticExtraBytes),
			})
		}
	}

	return pools, nil
}
