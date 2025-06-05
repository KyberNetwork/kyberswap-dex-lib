package v3

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

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
			"first": first,
			"skip":  skip,
		}).Errorf("failed to query subgraph")
		return nil, fmt.Errorf("failed to query subgraph: %w", err)
	}

	return response.Pools, nil
}

func (d *PoolsListUpdater) processPool(p SubgraphPool, staticData StaticData) entity.Pool {
	decimals0, _ := kutils.Atou[uint8](p.LpToken0.Decimals)
	decimals1, _ := kutils.Atou[uint8](p.LpToken1.Decimals)

	tokens := make([]*entity.PoolToken, 0, 4)

	// Add underlying tokens if exists
	for i, underlyingToken := range staticData.UnderlyingTokens {
		if strings.EqualFold(underlyingToken, valueobject.ZeroAddress) {
			continue
		}

		tokens = append(tokens, &entity.PoolToken{
			Address:   underlyingToken,
			Decimals:  lo.Ternary(i == 0, decimals0, decimals1),
			Swappable: true,
		})
	}

	tokens = append(tokens,
		&entity.PoolToken{
			Address:   p.LpToken0.Address,
			Decimals:  decimals0,
			Symbol:    p.LpToken0.Symbol,
			Swappable: true,
		},
		&entity.PoolToken{
			Address:   p.LpToken1.Address,
			Decimals:  decimals1,
			Symbol:    p.LpToken1.Symbol,
			Swappable: true,
		},
	)

	reserves := make(entity.PoolReserves, len(tokens))
	for i := range reserves {
		reserves[i] = "0"
	}

	swapFee, err := strconv.ParseFloat(p.FeeTier, 64)
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.ID,
			"error": err,
		}).Warn("invalid fee tier, using 0")
		swapFee = 0
	}

	createdAtTimestamp, err := kutils.Atoi[int64](p.CreatedAtTimestamp)
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.ID,
			"error": err,
		}).Warn("invalid timestamp, using 0")
		createdAtTimestamp = 0
	}

	staticBytes, err := json.Marshal(StaticExtra{
		TickSpacing:        staticData.TickSpacing,
		NeedScanUnderlying: true,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.ID,
			"error": err,
		}).Error("failed to marshal static extra data")
	}

	return entity.Pool{
		Address:     strings.ToLower(p.ID),
		SwapFee:     swapFee,
		Exchange:    d.config.DexID,
		Type:        DexType,
		Timestamp:   createdAtTimestamp,
		Reserves:    reserves,
		Tokens:      tokens,
		StaticExtra: string(staticBytes),
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{
		LastCreatedAtTimestamp: integer.Zero(),
	}
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, graphFirstLimit, 0)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error":         err,
			"lastTimestamp": metadata.LastCreatedAtTimestamp.String(),
		}).Errorf("failed to get pools list from subgraph")
		return nil, metadataBytes, fmt.Errorf("failed to get pools list: %w", err)
	}

	numSubgraphPools := len(subgraphPools)
	logger.Infof("got %v subgraph pools from %s subgraph", numSubgraphPools, d.config.DexID)

	poolDatas, err := d.FetchStaticData(ctx, subgraphPools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error":     err,
			"poolCount": numSubgraphPools,
		}).Errorf("failed to fetch rpc data")
		return nil, metadataBytes, fmt.Errorf("failed to fetch static data: %w", err)
	}

	pools := lo.Map(subgraphPools, func(p SubgraphPool, _ int) entity.Pool {
		return d.processPool(p, poolDatas[p.ID])
	})

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
		return nil, metadataBytes, fmt.Errorf("failed to marshal new metadata: %w", err)
	}

	logger.Infof("got %v %s pools", len(pools), d.config.DexID)

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) FetchStaticData(
	ctx context.Context,
	pools []SubgraphPool,
) (map[string]StaticData, error) {
	result := make(map[string]StaticData, len(pools))

	for i := 0; i < len(pools); i += rpcChunkSize {
		endIndex := min(i+rpcChunkSize, len(pools))
		chunk := pools[i:endIndex]

		tickSpacings := make([]*big.Int, len(chunk))
		underlyingTokens0 := make([]common.Address, len(chunk))
		underlyingTokens1 := make([]common.Address, len(chunk))

		req := d.ethrpcClient.NewRequest().SetContext(ctx)
		for j, pool := range chunk {
			req.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: pool.ID,
				Method: poolMethodTickSpacing,
				Params: nil,
			}, []any{&tickSpacings[j]})

			req.AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: pool.LpToken0.Address,
				Method: lpTokenMethodUnderlying,
				Params: nil,
			}, []any{&underlyingTokens0[j]})

			req.AddCall(&ethrpc.Call{
				ABI:    lpTokenABI,
				Target: pool.LpToken1.Address,
				Method: lpTokenMethodUnderlying,
				Params: nil,
			}, []any{&underlyingTokens1[j]})
		}

		_, err := req.TryAggregate()
		if err != nil {
			logger.WithFields(logger.Fields{
				"error":      err,
				"startIndex": i,
				"endIndex":   endIndex,
			}).Error("[FetchStaticData] failed to process Aggregate")
			return nil, fmt.Errorf("failed to aggregate RPC calls: %w", err)
		}

		for j := range chunk {
			poolAddress := pools[i+j].ID
			result[poolAddress] = StaticData{
				TickSpacing: tickSpacings[j].Uint64(),
				UnderlyingTokens: []string{
					strings.ToLower(underlyingTokens0[j].Hex()),
					strings.ToLower(underlyingTokens1[j].Hex()),
				},
			}
		}
	}

	return result, nil
}
