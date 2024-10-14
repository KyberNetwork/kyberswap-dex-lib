package ambient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolListUpdater struct {
	cfg           Config
	poolDatastore IPoolDatastore
	subgraph      *graphql.Client
}

func NewPoolsListUpdater(
	cfg Config,
	poolDatastore IPoolDatastore,
) (*PoolListUpdater, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &PoolListUpdater{
		cfg:           cfg,
		poolDatastore: poolDatastore,
		subgraph: graphqlpkg.New(graphqlpkg.Config{
			Url:     cfg.SubgraphURL,
			Header:  cfg.SubgraphHeaders,
			Timeout: cfg.SubgraphRequestTimeout.Duration,
		}),
	}, nil
}

// GetNewPools fetch new pools from the subgraph
func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.cfg.DexID
		startTime = time.Now()
		meta      PoolListUpdaterMetadata
	)
	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			logger.
				WithFields(logger.Fields{"dex_id": dexID, "err": err}).
				Error("unmarshal metadata failed")
			return nil, nil, err
		}
	}

	subgraphPairs, err := u.fetchSubgraph(ctx, meta.LastCreateTime)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("fetchSubgraph failed")
		return nil, metadataBytes, err
	}
	if len(subgraphPairs) == 0 {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":      dexID,
					"pools_len":   0,
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pools")
		return nil, metadataBytes, nil
	}

	subgraphPairs = u.excludePoolsWithWrappedNativeToken(subgraphPairs)

	upsertPool, extra, err := u.getOrInitializePool(ctx, u.cfg.SwapDexContractAddress)
	if err != nil {
		return nil, metadataBytes, err
	}

	u.appendTokenPairsToExtra(extra, subgraphPairs)

	encodedExtra, err := json.Marshal(extra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "address": u.cfg.SwapDexContractAddress, "err": err}).
			Error("could not marshal Extra")
		return nil, metadataBytes, err
	}
	upsertPool.Extra = string(encodedExtra)

	u.updateTokens(&upsertPool, subgraphPairs)

	newMetaBytes, err := u.metadataBytes(subgraphPairs)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "address": u.cfg.SwapDexContractAddress, "err": err}).
			Error("get new metadata bytes failed")
		return nil, nil, err
	}

	return []entity.Pool{upsertPool}, newMetaBytes, nil
}

func (u *PoolListUpdater) fetchSubgraph(ctx context.Context, lastCreateTime uint64) ([]SubgraphPool, error) {
	limit := uint64(defaultSubgraphLimit)
	if u.cfg.SubgraphLimit != 0 {
		limit = u.cfg.SubgraphLimit
	}
	var (
		req = graphql.NewRequest(fmt.Sprintf(`{
	pools(
		where: {
			timeCreate_gt: %d,
		}
		orderBy: timeCreate,
		orderDirection: asc,
		first: %d,
	) {
		id
		blockCreate
		timeCreate
		base
		quote
		poolIdx
	}
}`, lastCreateTime, limit))
		resp SubgraphPoolsResponse
	)

	if err := u.subgraph.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Pools, nil
}

// Ambient uses native token (ETH) instead of ERC20 wrapped native token (WETH) as native reserve.
// So we exclude any pools involing ERC20 wrapped native token, if any.
func (u *PoolListUpdater) excludePoolsWithWrappedNativeToken(pairs []SubgraphPool) []SubgraphPool {
	wrappedNativeAddr := common.HexToAddress(u.cfg.NativeTokenAddress)
	excluded := make([]SubgraphPool, 0, len(pairs))
	for _, pair := range pairs {
		base := common.HexToAddress(pair.Base)
		quote := common.HexToAddress(pair.Quote)
		if base == wrappedNativeAddr || quote == wrappedNativeAddr {
			continue
		}
		excluded = append(excluded, pair)
	}
	return excluded
}

func (u *PoolListUpdater) getOrInitializePool(ctx context.Context, address string) (entity.Pool, *Extra, error) {
	upsertPool, err := u.poolDatastore.Get(ctx, strings.ToLower(address))
	if err != nil {
		upsertPool = entity.Pool{
			Address:  address,
			Exchange: u.cfg.DexID,
			Type:     DexTypeAmbient,
		}
		logger.
			WithFields(logger.Fields{"dex_id": u.cfg.DexID, "address": address, "err": err}).
			Warn("error when getting pool by address from datastore, assume there is no pool")
	}

	staticExtra := StaticExtra{
		NativeTokenAddress: common.HexToAddress(u.cfg.NativeTokenAddress),
	}
	encodedStaticExtra, err := json.Marshal(staticExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": u.cfg.DexID, "address": address, "err": err}).
			Error("could not unmarshal StaticExtra")
		return entity.Pool{}, nil, err
	}
	upsertPool.StaticExtra = string(encodedStaticExtra)

	extra := new(Extra)
	if len(upsertPool.Extra) != 0 {
		if err := json.Unmarshal([]byte(upsertPool.Extra), extra); err != nil {
			logger.
				WithFields(logger.Fields{"dex_id": u.cfg.DexID, "address": address, "err": err}).
				Error("could not unmarshal Extra")
			return entity.Pool{}, nil, err
		}
	}
	if extra.TokenPairs == nil {
		extra.TokenPairs = make(map[TokenPair]*TokenPairInfo)
	}

	return upsertPool, extra, nil
}

func (u *PoolListUpdater) appendTokenPairsToExtra(extra *Extra, tokenPair []SubgraphPool) {
	for _, p := range tokenPair {
		pair := TokenPair{
			Base:  common.HexToAddress(p.Base),
			Quote: common.HexToAddress(p.Quote),
		}
		if _, ok := extra.TokenPairs[pair]; ok {
			// don't overwrite existing pair
			continue
		}
		poolIdx := bignumber.NewBig10(p.PoolIdx)
		if poolIdx.Cmp(u.cfg.PoolIdx) != 0 {
			// skip pools whose PoolIdx is unrelated to the config
			continue
		}
		extra.TokenPairs[pair] = &TokenPairInfo{
			PoolIdx: bignumber.NewBig10(p.PoolIdx),
		}
	}
}

func (u *PoolListUpdater) updateTokens(pool *entity.Pool, tokenPairs []SubgraphPool) {
	existingTokenSet := make(map[common.Address]struct{})
	for _, token := range pool.Tokens {
		existingTokenSet[common.HexToAddress(token.Address)] = struct{}{}
	}
	// update upsertPool's tokens list
	for _, p := range tokenPairs {
		baseToken := common.HexToAddress(p.Base)
		if baseToken == NativeTokenPlaceholderAddress {
			baseToken = common.HexToAddress(u.cfg.NativeTokenAddress)
		}
		if _, ok := existingTokenSet[baseToken]; !ok {
			existingTokenSet[baseToken] = struct{}{}
			pool.Tokens = append(pool.Tokens, &entity.PoolToken{
				Address:   strings.ToLower(baseToken.String()),
				Swappable: true,
			})
			pool.Reserves = append(pool.Reserves, "0")
		}

		quoteToken := common.HexToAddress(p.Quote)
		if _, ok := existingTokenSet[quoteToken]; !ok {
			existingTokenSet[quoteToken] = struct{}{}
			pool.Tokens = append(pool.Tokens, &entity.PoolToken{
				Address:   strings.ToLower(quoteToken.String()),
				Swappable: true,
			})
			pool.Reserves = append(pool.Reserves, "0")
		}
	}

}

func (u *PoolListUpdater) metadataBytes(subgraphPools []SubgraphPool) ([]byte, error) {
	var metadata PoolListUpdaterMetadata
	if len(subgraphPools) == 0 {
		metadata.LastCreateTime = 0
	} else {
		metadata.LastCreateTime = subgraphPools[len(subgraphPools)-1].TimeCreate
	}

	return json.Marshal(metadata)
}
