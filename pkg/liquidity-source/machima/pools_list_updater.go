package machima

import (
	"context"
	"math/big"
	"strconv"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client) *PoolsListUpdater {
	// entity.Pool.Exchange has to match valueobject.ExchangeMachima or router-service filters the
	// pools out, so fall back to it rather than emitting pools under an empty exchange.
	if cfg.DexID == "" {
		cfg.DexID = DexType
	}
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient, graphqlClient: graphqlClient}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{LastCreatedAtTimestamp: big.NewInt(0)}
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	subgraphPools, err := d.getPoolsList(ctx, metadata.LastCreatedAtTimestamp)
	if err != nil {
		logger.WithFields(logger.Fields{"dex": DexType, "error": err}).
			Error("failed to get machima pools list from subgraph")
		return nil, metadataBytes, err
	}

	logger.Infof("got %v %s subgraph pools", len(subgraphPools), DexType)

	// Machima launches every pool at the same fee tier today, but read both per pool anyway so an
	// added tier cannot silently mis-price: fee drives the V3 swap math and tickSpacing drives
	// tick traversal.
	addresses := make([]string, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		addresses = append(addresses, p.ID)
	}
	fees, tickSpacings := d.fetchPoolParams(ctx, addresses)

	pools := make([]entity.Pool, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		address := strings.ToLower(p.ID)
		token := strings.ToLower(p.TradedToken)
		if token == "" {
			logger.WithFields(logger.Fields{"dex": DexType, "pool": address}).
				Warn("skipping machima pool with no traded token")
			continue
		}

		staticExtra, err := json.Marshal(StaticExtra{
			Token:         token,
			RouterAddress: strings.ToLower(d.config.RouterAddress),
			WETH:          strings.ToLower(d.config.WETH),
			USDC:          strings.ToLower(d.config.USDC),
			XMA:           strings.ToLower(d.config.XMA),
		})
		if err != nil {
			return nil, metadataBytes, err
		}

		fee, ok := fees[p.ID]
		if !ok {
			fee = defaultFee
		}
		tickSpacing, ok := tickSpacings[p.ID]
		if !ok {
			tickSpacing = defaultTickSpacing
		}

		// The tracker overwrites Extra on the first cycle; tickSpacing is seeded here so it can
		// traverse ticks correctly on that very first pass.
		extra, err := json.Marshal(Extra{Extra: uniswapv3.Extra{TickSpacing: tickSpacing}})
		if err != nil {
			return nil, metadataBytes, err
		}

		createdAt, ok := new(big.Int).SetString(p.CreatedAt, 10)
		if !ok {
			return nil, metadataBytes, errors.Errorf("invalid createdAt %q for pool %s", p.CreatedAt, address)
		}

		// Reserves are placeholders; the tracker fills real wei values from balanceOf.
		pools = append(pools, entity.Pool{
			Address:     address,
			SwapFee:     float64(fee),
			Exchange:    d.config.DexID,
			Type:        DexType,
			Timestamp:   createdAt.Int64(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      []*entity.PoolToken{{Address: strings.ToLower(p.Token0.Address), Swappable: true}, {Address: strings.ToLower(p.Token1.Address), Swappable: true}},
			Extra:       string(extra),
			StaticExtra: string(staticExtra),
		})
	}

	lastCreatedAt := metadata.LastCreatedAtTimestamp
	if len(subgraphPools) > 0 {
		last := subgraphPools[len(subgraphPools)-1]
		ts, ok := new(big.Int).SetString(last.CreatedAt, 10)
		if !ok {
			return nil, metadataBytes, errors.Errorf("invalid createdAt %q for pool %s", last.CreatedAt, last.ID)
		}
		lastCreatedAt = ts
	}

	newMetadataBytes, err := json.Marshal(Metadata{LastCreatedAtTimestamp: lastCreatedAt})
	if err != nil {
		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

// fetchPoolParams reads fee() and tickSpacing() for each pool. A pool whose reads revert is simply
// absent from the maps and falls back to the launch defaults.
func (d *PoolsListUpdater) fetchPoolParams(ctx context.Context,
	addresses []string) (fees map[string]uint32, tickSpacings map[string]uint64) {
	fees = make(map[string]uint32, len(addresses))
	tickSpacings = make(map[string]uint64, len(addresses))

	for start := 0; start < len(addresses); start += rpcChunkSize {
		chunk := addresses[start:min(start+rpcChunkSize, len(addresses))]

		feeResults := make([]*big.Int, len(chunk))
		tickSpacingResults := make([]*big.Int, len(chunk))

		req := d.ethrpcClient.NewRequest().SetContext(ctx)
		for i, address := range chunk {
			req.AddCall(&ethrpc.Call{ABI: poolABI, Target: address, Method: methodFee},
				[]any{&feeResults[i]})
			req.AddCall(&ethrpc.Call{ABI: poolABI, Target: address, Method: methodTickSpacing},
				[]any{&tickSpacingResults[i]})
		}

		if _, err := req.TryAggregate(); err != nil {
			logger.WithFields(logger.Fields{"dex": DexType, "error": err}).
				Warn("failed to fetch machima pool fee/tickSpacing, falling back to launch defaults")
			continue
		}

		for i, address := range chunk {
			if feeResults[i] != nil {
				fees[address] = uint32(feeResults[i].Uint64())
			}
			if tickSpacingResults[i] != nil {
				tickSpacings[address] = tickSpacingResults[i].Uint64()
			}
		}
	}

	return fees, tickSpacings
}

func (d *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAt *big.Int) ([]SubgraphPool, error) {
	req := graphqlpkg.NewRequest(`{
		pools(
			first: ` + strconv.Itoa(graphFirstLimit) + `,
			orderBy: createdAt,
			orderDirection: asc,
			where: { createdAt_gte: ` + lastCreatedAt.String() + ` }
		) {
			id
			token0 { id }
			token1 { id }
			tradedToken
			createdAt
		}
	}`)

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}
	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Pools, nil
}
