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

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

var (
	_ = poollist.RegisterFactoryCEG(DexTypeUniswapV3, NewUniswapV3PoolsListUpdater)
	_ = poollist.RegisterFactoryCEG(DexTypePancakeV3, NewPancakeV3PoolsListUpdater)
	_ = poollist.RegisterFactoryCEG(DexTypeRamsesV2, NewRamsesV2PoolsListUpdater)
	_ = poollist.RegisterFactoryCEG(DexTypeSolidlyV3, NewSolidlyV3PoolsListUpdater)
	_ = poollist.RegisterFactoryCEG(DexTypeSlipstream, NewSlipstreamPoolsListUpdater)
	_ = poollist.RegisterFactoryCEG(DexTypeNuriV2, NewNuriV2PoolsListUpdater)
)

func NewUniswapV3PoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return newPoolsListUpdater(cfg, ethrpcClient, graphqlClient, DexTypeUniswapV3, true)
}

func NewPancakeV3PoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return newPoolsListUpdater(cfg, ethrpcClient, graphqlClient, DexTypePancakeV3, true)
}

func NewRamsesV2PoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return newPoolsListUpdater(cfg, ethrpcClient, graphqlClient, DexTypeRamsesV2, true)
}

func NewSolidlyV3PoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return newPoolsListUpdater(cfg, ethrpcClient, graphqlClient, DexTypeSolidlyV3, true)
}

func NewSlipstreamPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	// slipstream's subgraph has no feeTier field at all (fee is fully dynamic, unknown
	// until the tracker's first fee()/currentFee() refresh).
	return newPoolsListUpdater(cfg, ethrpcClient, graphqlClient, DexTypeSlipstream, false)
}

func NewNuriV2PoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	return newPoolsListUpdater(cfg, ethrpcClient, graphqlClient, DexTypeNuriV2, true)
}

// PoolsListUpdater discovers pools for every uniswap-v3 fork merged into this package purely
// from the subgraph. It deliberately does not fetch tickSpacing (or fee) over RPC at listing
// time: a freshly listed pool has no ticks yet, so NewPoolSimulatorWithExtra already refuses
// to construct a simulator for it (ErrV3TicksEmpty) until the tracker's first refresh - which
// fetches tickSpacing/fee anyway. Pre-fetching them here would just be a wasted round trip.
type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	graphqlClient  *graphqlpkg.Client
	dexType        string
	includeFeeTier bool
}

func newPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
	dexType string,
	includeFeeTier bool,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		graphqlClient:  graphqlClient,
		dexType:        dexType,
		includeFeeTier: includeFeeTier,
	}
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp *big.Int, first, skip int) ([]SubgraphPool, error) {
	allowSubgraphError := d.config.IsAllowSubgraphError()

	req := graphqlpkg.NewRequest(getPoolsListQuery(
		d.config.SubgraphPoolField, d.includeFeeTier, allowSubgraphError, lastCreatedAtTimestamp, first, skip))

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

	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		tokens := make([]*entity.PoolToken, 0, 2)
		reserves := make([]string, 0, 2)

		if p.Token0.Address != "" {
			token0Decimals, err := kutils.Atou[uint8](p.Token0.Decimals)
			if err != nil {
				token0Decimals = defaultTokenDecimals
			}

			tokens = append(tokens, &entity.PoolToken{
				Address:   p.Token0.Address,
				Symbol:    p.Token0.Symbol,
				Decimals:  token0Decimals,
				Swappable: true,
			})
			reserves = append(reserves, "0")
		}

		if p.Token1.Address != "" {
			token1Decimals, err := kutils.Atou[uint8](p.Token1.Decimals)
			if err != nil {
				token1Decimals = defaultTokenDecimals
			}

			tokens = append(tokens, &entity.PoolToken{
				Address:   p.Token1.Address,
				Symbol:    p.Token1.Symbol,
				Decimals:  token1Decimals,
				Swappable: true,
			})
			reserves = append(reserves, "0")
		}

		var swapFee float64
		if d.includeFeeTier {
			swapFee, _ = strconv.ParseFloat(p.FeeTier, 64)
		}

		createdAtTimestamp, err := kutils.Atoi[int64](p.CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, fmt.Errorf("invalid CreatedAtTimestamp: %v, pool: %v", p.CreatedAtTimestamp, p.ID)
		}

		extraBytes, _ := json.Marshal(ExtraTickU256{})
		staticExtraBytes, _ := json.Marshal(StaticExtra{PoolId: p.ID})

		pools = append(pools, entity.Pool{
			Address:     p.ID,
			SwapFee:     swapFee,
			Exchange:    d.config.DexID,
			Type:        d.dexType,
			Timestamp:   createdAtTimestamp,
			Reserves:    reserves,
			Tokens:      tokens,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		})
	}

	// Track the last pool's CreatedAtTimestamp
	lastCreatedAtTimestamp := metadata.LastCreatedAtTimestamp
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
