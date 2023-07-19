package maverick

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"
	"math/big"
	"strconv"
	"time"
)

type PoolListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolListUpdater{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
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
			"type":  DexTypeMaverickV1,
			"error": err,
		}).Errorf("failed to get new pools")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(Metadata{LastCreateTime: lastCreatedTime})
	if err != nil {
		logger.WithFields(logger.Fields{
			"type":  DexTypeMaverickV1,
			"error": err,
		}).Errorf("failed to marshal metadata")
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolListUpdater) getNewPoolFromSubgraph(ctx context.Context, lastCreateTime *big.Int) ([]entity.Pool, *big.Int, error) {
	logger.WithFields(logger.Fields{
		"type": DexTypeMaverickV1,
	}).Info("start getting new pools...")

	subgraphPools, err := d.querySubgraph(ctx, lastCreateTime, d.config.NewPoolLimit, 0)
	if err != nil {
		logger.WithFields(logger.Fields{
			"type":  poolABI,
			"error": err,
		})
		return nil, lastCreateTime, err
	}

	logger.WithFields(logger.Fields{
		"type": DexTypeMaverickV1,
	}).Infof("get %v pools from subgraph", len(subgraphPools))

	var pools = make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		var tokens = []*entity.PoolToken{
			{
				Address:  p.TokenA.ID,
				Decimals: p.TokenA.Decimals,
			},
			{
				Address:  p.TokenB.ID,
				Decimals: p.TokenB.Decimals,
			},
		}
		var reserves = []string{zeroString, zeroString}

		tickSpacing := bignumber.NewBig10(p.TickSpacing)
		var staticExtra = StaticExtra{
			TickSpacing: tickSpacing,
		}

		staticBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"type":  DexTypeMaverickV1,
				"error": err,
			}).Errorf("failed to marshal static extra")
			return nil, lastCreateTime, err
		}

		swapFee, _ := strconv.ParseFloat(p.Fee, 64)

		var newPool = entity.Pool{
			Address:     p.ID,
			SwapFee:     swapFee,
			Exchange:    d.config.DexID,
			Type:        DexTypeMaverickV1,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticBytes),
		}

		pools = append(pools, newPool)
	}

	// Track the last pool's CreatedAtTimestamp
	newLastCreateTime := lastCreateTime
	if len(subgraphPools) > 0 {
		lastSubgraphPool := subgraphPools[len(subgraphPools)-1]
		newLastCreateTime = bignumber.NewBig10(lastSubgraphPool.Timestamp)
	}

	logger.WithFields(logger.Fields{
		"type":     DexTypeMaverickV1,
		"newPools": len(pools),
	}).Info("finish getting new pools")

	return pools, newLastCreateTime, nil
}

func (d *PoolListUpdater) querySubgraph(
	ctx context.Context,
	lastCreateTime *big.Int,
	first int,
	skip int,
) ([]*SubgraphPool, error) {
	if lastCreateTime == nil {
		lastCreateTime = zeroBI
	}

	req := graphql.NewRequest(fmt.Sprintf(`{
		pools(
			where : {
				timestamp_gte: %v,
			},
			first: %v,
			skip: %v,
			orderBy: timestamp,
			orderDirection: asc,
		) {
			id
			tickSpacing
			fee
			protocolFeeRatio
			timestamp
			tokenA {
			  id 
			  decimals	
			}
			tokenB {
			  id 
			  decimals	
			}
		}
	}`, lastCreateTime, first, skip),
	)

	var response struct {
		Pools []*SubgraphPool `json:"pools"`
	}
	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"type":  DexTypeMaverickV1,
			"error": err,
		}).Errorf("failed to query subgraph to get pools")
		return nil, err
	}

	return response.Pools, nil
}
