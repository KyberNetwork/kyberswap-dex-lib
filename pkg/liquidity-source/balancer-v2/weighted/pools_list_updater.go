package weighted

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
	sharedUpdater *shared.PoolsListUpdater
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	graphqlClient := graphqlpkg.NewWithTimeout(config.SubgraphAPI, graphQLRequestTimeout)

	sharedUpdater := shared.NewPoolsListUpdater(&shared.Config{
		DexID:        config.DexID,
		SubgraphAPI:  config.SubgraphAPI,
		NewPoolLimit: config.NewPoolLimit,
		PoolType:     poolTypeWeighted,
	})

	return &PoolsListUpdater{
		config:        *config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
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

	pools, err := u.initPools(ctx, subgraphPools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":   u.config.DexID,
			"dexType": DexType,
		}).Error(err.Error())

		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, subgraphPools []*shared.SubgraphPool) ([]entity.Pool, error) {
	pools := make([]entity.Pool, len(subgraphPools))

	for i, subgraphPool := range subgraphPools {
		var (
			poolTokens     = make([]*entity.PoolToken, len(subgraphPool.Tokens))
			reserves       = make([]string, len(subgraphPool.Tokens))
			scalingFactors = make([]*big.Int, len(subgraphPool.Tokens))
		)

		for j, token := range subgraphPool.Tokens {

			w, err := strconv.ParseFloat(token.Weight, 64)
			if err != nil {
				return nil, err
			}
			weight := uint(w * 1e18)
			if weight == 0 {
				weight = uint(1e18 / len(subgraphPool.Tokens))
			}
			poolTokens[j] = &entity.PoolToken{
				Address:   token.Address,
				Weight:    weight,
				Swappable: true,
			}

			reserves[j] = "0"

			scalingFactors[j] = bignumber.TenPowInt(18 - uint8(token.Decimals))
			if subgraphPool.PoolTypeVersion.Int64() > poolTypeVer1 {
				scalingFactors[j] = new(big.Int).Mul(scalingFactors[j], bignumber.BONE)
			}
		}

		staticExtra := StaticExtra{
			PoolID:          subgraphPool.ID,
			PoolType:        subgraphPool.PoolType,
			PoolTypeVersion: int(subgraphPool.PoolTypeVersion.Int64()),
			ScalingFactors:  scalingFactors,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     subgraphPool.Address,
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Tokens:      poolTokens,
			Reserves:    reserves,
			StaticExtra: string(staticExtraBytes),
		}

	}

	return pools, nil
}
