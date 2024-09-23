package wombat

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	graphqlClient := graphqlpkg.New(graphqlpkg.Config{
		Url:     cfg.SubgraphAPI,
		Header:  cfg.SubgraphHeaders,
		Timeout: graphQLRequestTimeout,
	})

	return &PoolsListUpdater{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	ctx = util.NewContextWithTimestamp(ctx)
	pools, lastCreatedTime, err := d.getNewPoolFromSubgraph(ctx, metadata.LastCreateTime)
	if err != nil {
		logger.WithFields(logger.Fields{
			"type":  DexTypeWombat,
			"error": err,
		}).Errorf("failed to get new pools")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(Metadata{LastCreateTime: lastCreatedTime})
	if err != nil {
		logger.WithFields(logger.Fields{
			"type":  DexTypeWombat,
			"error": err,
		}).Errorf("failed to marshal metadata")
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) getNewPoolFromSubgraph(ctx context.Context, lastCreateTime uint64) ([]entity.Pool, uint64, error) {
	logger.WithFields(logger.Fields{
		"type": DexTypeWombat,
	}).Info("start getting new pools...")

	subgraphPools, err := d.querySubgraph(ctx, lastCreateTime)
	if err != nil {
		logger.WithFields(logger.Fields{
			"type":  DexTypeWombat,
			"error": err,
		})
		return nil, lastCreateTime, err
	}

	logger.WithFields(logger.Fields{
		"type": DexTypeWombat,
	}).Infof("get %v pools from subgraph", len(subgraphPools))

	var pools = make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		if len(p.Assets) == 0 {
			continue
		}
		var reserves = make([]string, len(p.Assets))
		var tokens = make([]*entity.PoolToken, len(p.Assets))
		for j, asset := range p.Assets {
			tokens[j] = &entity.PoolToken{
				Address:   asset.UnderlyingToken.ID,
				Weight:    defaultTokenWeight,
				Decimals:  asset.UnderlyingToken.Decimals,
				Swappable: true,
			}
			reserves[j] = zeroString
		}

		poolType, err := d.classifyPoolType(ctx, p)
		if err != nil {
			return nil, lastCreateTime, err
		}
		if poolType == PoolTypeWombatCrossChain {
			continue
		}

		var newPool = entity.Pool{
			Address:   p.ID,
			Exchange:  d.config.DexID,
			Type:      poolType,
			Timestamp: time.Now().Unix(),
			Reserves:  reserves,
			Tokens:    tokens,
		}

		pools = append(pools, newPool)
	}

	// Track the last pool's CreatedAtTimestamp
	newLastCreateTime := lastCreateTime
	if len(subgraphPools) > 0 {
		lastSubgraphPool := subgraphPools[len(subgraphPools)-1]
		newLastCreateTime, _ = strconv.ParseUint(lastSubgraphPool.CreatedTimestamp, 10, 64)
	}

	logger.WithFields(logger.Fields{
		"type":     DexTypeWombat,
		"newPools": len(pools),
	}).Info("finish getting new pools")

	return pools, newLastCreateTime, nil
}

func (d *PoolsListUpdater) querySubgraph(
	ctx context.Context,
	lastCreateTime uint64,
) ([]*SubgraphPool, error) {
	req := graphql.NewRequest(fmt.Sprintf(`{
		pools(
			orderBy: createdTimestamp
			orderDirection: asc
			where: {createdTimestamp_gte: %v}
		  ) {
			id
			assets {
			  id
			  underlyingToken {
				decimals
				id
			  }
			}
			createdTimestamp
		  }
	}`, lastCreateTime),
	)

	var response struct {
		Pools []*SubgraphPool `json:"pools"`
	}
	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"type":  DexTypeWombat,
			"error": err,
		}).Errorf("failed to query subgraph to get pools")
		return nil, err
	}

	return response.Pools, nil
}

// classifyPoolType
// PoolTypeWombatLSD has relativePrice in assets
// PoolTypeWombatCrossChain has creditForTokensHaircut
// PoolTypeWombatMain do not has creditForTokensHaircut and relativePrice in assets
func (d *PoolsListUpdater) classifyPoolType(ctx context.Context, p *SubgraphPool) (string, error) {
	var relativePrice, creditForTokensHaircut *big.Int

	if len(p.Assets) <= 0 {
		return "", fmt.Errorf("asset is not found")
	}
	assetAddress := p.Assets[0].ID

	calls := d.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(false)
	calls.AddCall(&ethrpc.Call{
		ABI:    DynamicAssetABI,
		Target: assetAddress,
		Method: assetMethodGetRelativePrice,
		Params: nil,
	}, []interface{}{&relativePrice})
	calls.AddCall(&ethrpc.Call{
		ABI:    CrossChainPoolABI,
		Target: p.ID,
		Method: poolMethodCreditForTokensHaircut,
		Params: nil,
	}, []interface{}{&creditForTokensHaircut})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"type": DexTypeWombat,
			"err":  err,
		}).Errorf("failed to try aggregate call")

		return "", err
	}

	if relativePrice != nil {
		return PoolTypeWombatLSD, nil
	}
	// createForTokensHaircut to detect PoolTypeWombatCrossChain.
	// But following wombat team, PoolTypeWombatCrossChain can swap on the same chain as PoolTypeWombatMain
	if creditForTokensHaircut != nil {
		return PoolTypeWombatMain, nil
	}

	return PoolTypeWombatMain, nil
}
