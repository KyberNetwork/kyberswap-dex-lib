package uniswapv3

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = poollist.RegisterFactoryCEG(DexTypeUniswapV3, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp *big.Int, first, skip int) ([]SubgraphPool, error) {
	allowSubgraphError := d.config.IsAllowSubgraphError()

	req := graphqlpkg.NewRequest(getPoolsListQuery(allowSubgraphError, lastCreatedAtTimestamp, first, skip))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		// Workaround at the moment to live with the error subgraph on Arbitrum
		if allowSubgraphError && len(response.Pools) > 0 {
			return response.Pools, nil
		}

		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to query subgraph")
		return nil, err
	}

	return response.Pools, nil
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastCreatedAtTimestamp: integer.Zero(),
	}
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, graphFirstLimit, 0)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get pools list from subgraph")
		return nil, metadataBytes, err
	}

	numSubgraphPools := len(subgraphPools)

	logger.Infof("got %v subgraph pools from %s subgraph", numSubgraphPools, d.config.DexID)

	tickSpacings, _ := FetchTickSpacings(
		ctx,
		lo.Map(subgraphPools, func(item SubgraphPool, _ int) string { return item.ID }),
		d.ethrpcClient,
		uniswapV3PoolABI,
		methodTickSpacing,
	)

	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		tokens := make([]*entity.PoolToken, 0, 2)
		reserves := make([]string, 0, 2)

		extraField := Extra{
			TickSpacing: tickSpacings[p.ID],
		}
		staticField := StaticExtra{
			PoolId: p.ID,
		}

		if p.Token0.Address != "" {
			token0Decimals, err := strconv.Atoi(p.Token0.Decimals)

			if err != nil {
				token0Decimals = defaultTokenDecimals
			}

			tokenModel := entity.PoolToken{
				Address:   p.Token0.Address,
				Symbol:    p.Token0.Symbol,
				Decimals:  uint8(token0Decimals),
				Swappable: true,
			}

			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, "0")
		}

		if p.Token1.Address != "" {
			token1Decimals, err := strconv.Atoi(p.Token1.Decimals)

			if err != nil {
				token1Decimals = defaultTokenDecimals
			}

			tokenModel := entity.PoolToken{
				Address:   p.Token1.Address,
				Symbol:    p.Token1.Symbol,
				Decimals:  uint8(token1Decimals),
				Swappable: true,
			}

			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, "0")
		}

		var swapFee, _ = strconv.ParseFloat(p.FeeTier, 64)

		createdAtTimestamp, err := kutils.Atoi[int64](p.CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v", p.CreatedAtTimestamp, p.ID)
		}

		extraBytes, _ := json.Marshal(extraField)
		staticBytes, _ := json.Marshal(staticField)
		var newPool = entity.Pool{
			Address:      p.ID,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      swapFee,
			Exchange:     d.config.DexID,
			Type:         DexTypeUniswapV3,
			Timestamp:    createdAtTimestamp,
			Reserves:     reserves,
			Tokens:       tokens,
			Extra:        string(extraBytes),
			StaticExtra:  string(staticBytes),
		}

		pools = append(pools, newPool)
	}

	// Track the last pool's CreatedAtTimestamp
	var lastCreatedAtTimestamp = metadata.LastCreatedAtTimestamp
	if len(subgraphPools) > 0 {
		lastSubgraphPoolIndex := len(subgraphPools) - 1
		ts, ok := new(big.Int).SetString(subgraphPools[lastSubgraphPoolIndex].CreatedAtTimestamp, 10)
		if !ok {
			return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v",
				subgraphPools[lastSubgraphPoolIndex].CreatedAtTimestamp, subgraphPools[lastSubgraphPoolIndex].ID)
		}

		lastCreatedAtTimestamp = ts
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastCreatedAtTimestamp: lastCreatedAtTimestamp,
	})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("got %v %s pools", len(pools), d.config.DexID)

	return pools, newMetadataBytes, nil
}
