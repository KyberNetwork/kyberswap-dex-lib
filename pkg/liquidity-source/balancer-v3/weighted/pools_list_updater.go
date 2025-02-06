package weighted

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	sharedUpdater *shared.PoolsListUpdater
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	sharedUpdater := shared.NewPoolsListUpdater(&shared.Config{
		DexID:           config.DexID,
		SubgraphAPI:     config.SubgraphAPI,
		SubgraphHeaders: config.SubgraphHeaders,
		NewPoolLimit:    config.NewPoolLimit,
		Factory:         config.Factory,
	}, graphqlClient)

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

	if u.config.Factory == "" {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error("factory config is empty")

		return nil, nil, errors.New("factory config is empty")
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
		staticExtraBytes, err := json.Marshal(&StaticExtra{
			Vault:             subgraphPool.Vault.ID,
			DefaultHook:       u.config.DefaultHook,
			IsPoolInitialized: subgraphPool.IsInitialized,
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

	return pools, nil
}
