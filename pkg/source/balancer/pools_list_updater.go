package balancer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
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
	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolsListUpdater{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	// Add more types of pool here
	supportedPoolTypes := []PoolType{
		subgraphPoolTypeWeighted,
		subgraphPoolTypeStable,
		subgraphPoolTypeMetaStable,
	}

	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	// Add timestamp to the context so that each run iteration will have something different
	ctx = util.NewContextWithTimestamp(ctx)

	var (
		newMetadata = make(Metadata)
		pools       []entity.Pool
	)

	// We don't want to use multiple goroutines here for the sake of simplicity and also the number pools is small
	for _, poolType := range supportedPoolTypes {
		poolsByType, newPoolTypeMetadataBytes, err := d.getNewPoolsByType(ctx, poolType, metadata[string(poolType)])
		if err != nil {
			logger.WithFields(logger.Fields{
				"type":  poolType,
				"error": err,
			}).Errorf("failed to update new pools by type")
			return nil, metadataBytes, err
		}

		newMetadata[string(poolType)] = newPoolTypeMetadataBytes
		pools = append(pools, poolsByType...)
	}

	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	numPools := len(pools)

	if numPools > 0 {
		logger.Infof("got total of %v Balancer pools of %v types", numPools, len(supportedPoolTypes))
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) getNewPoolsByType(ctx context.Context, poolType PoolType, poolTypeMetadata PoolTypeMetadata) ([]entity.Pool, PoolTypeMetadata, error) {
	logger.WithFields(logger.Fields{
		"type": poolType,
	}).Info("start updating new pools ...")

	subgraphPools, err := d.getPoolsListByType(ctx, poolType, poolTypeMetadata.LastCreateTime, d.config.NewPoolLimit, 0)
	if err != nil {
		logger.WithFields(logger.Fields{
			"type":  poolType,
			"error": err,
		}).Errorf("failed to get list of Balancer from subgraph")
		return nil, poolTypeMetadata, err
	}

	logger.WithFields(logger.Fields{
		"type": poolType,
	}).Infof("got %v pools from subgraph", len(subgraphPools))

	getVaultsRequest := d.ethrpcClient.NewRequest()

	var vaultAddresses = make([]common.Address, len(subgraphPools))
	for i, subgraphPair := range subgraphPools {
		getVaultsRequest.AddCall(&ethrpc.Call{
			ABI:    balancerPoolABI,
			Target: subgraphPair.Address,
			Method: poolMethodGetVault,
			Params: nil,
		}, []interface{}{&vaultAddresses[i]})
	}

	if _, err = getVaultsRequest.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"type":  poolType,
			"error": err,
		}).Errorf("failed to aggregate for Balancer vaults")
		return nil, poolTypeMetadata, err
	}

	pools := make([]entity.Pool, 0, len(subgraphPools))

	for i, p := range subgraphPools {
		var tokens = make([]*entity.PoolToken, 0)
		var reserves = make([]string, 0)
		var staticField = StaticExtra{
			VaultAddress: strings.ToLower(vaultAddresses[i].Hex()),
			PoolId:       p.ID,
		}

		for _, item := range p.Tokens {
			weight, _ := strconv.ParseFloat(item.Weight, 64)
			poolToken := entity.PoolToken{
				Address:   item.Address,
				Weight:    uint(weight * 1e18),
				Swappable: true,
			}

			staticField.TokenDecimals = append(staticField.TokenDecimals, item.Decimals)
			if poolToken.Weight == 0 {
				poolToken.Weight = uint(1e18 / len(p.Tokens))
			}

			tokens = append(tokens, &poolToken)
			reserves = append(reserves, zeroString)
		}
		var swapFee, _ = strconv.ParseFloat(p.SwapFee, 64)

		staticBytes, _ := json.Marshal(staticField)
		var newPool = entity.Pool{
			Address:     p.Address,
			ReserveUsd:  zeroFloat64,
			SwapFee:     swapFee,
			Exchange:    d.config.DexID,
			Type:        string(dexTypeByPoolType[poolType]),
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticBytes),
		}

		pools = append(pools, newPool)
	}

	// Track the last pool's CreatedAtTimestamp
	lastCreateTime := poolTypeMetadata.LastCreateTime
	if len(subgraphPools) > 0 {
		lastSubgraphPool := subgraphPools[len(subgraphPools)-1]
		lastCreateTime = lastSubgraphPool.CreateTime
	}

	newPoolTypeMetadata := PoolTypeMetadata{
		LastCreateTime: lastCreateTime,
	}

	logger.Infof("got %v %v pools", len(pools), poolType)

	return pools, newPoolTypeMetadata, nil
}

func (d *PoolsListUpdater) getPoolsListByType(
	ctx context.Context,
	poolType PoolType,
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
				poolType: "%v",
				createTime_gte: %v,
				totalShares_gt: 0.01,
				swapEnabled: true
			},
			first: %v,
			skip: %v,
			orderBy: createTime,
			orderDirection: asc,
		) {
			id
			address
			poolType
			swapFee
			createTime
			tokens {
			  address
			  decimals	
			  weight
			}
		}
	}`, poolType, lastCreateTime, first, skip),
	)

	var response struct {
		Pairs []*SubgraphPool `json:"pools"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"type":  poolType,
			"error": err,
		}).Errorf("failed to query subgraph to get pools list")
		return nil, err
	}

	return response.Pairs, nil
}
