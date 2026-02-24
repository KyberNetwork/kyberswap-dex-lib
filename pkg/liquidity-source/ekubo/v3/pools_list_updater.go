package ekubov3

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const subgraphInitialStartID = "0x00000000000000000000000000000000"
const subgraphPageSize = 1000

// The extra bounds on `extension` are used to include any pool keys that have no beforeSwap and afterSwap hook, see https://github.com/EkuboProtocol/evm-contracts/blob/665e8333e550003b68a94d8482cc9fda438a2bf1/src/types/callPoints.sol
var subgraphQuery = fmt.Sprintf(`
query NewPools(
  $startId: Bytes!
  $extensions: [Bytes!]
) {
  poolInitializations(
    first: %d
    where: {
      and: [
        {id_gte: $startId}
        {or: [
          {extension_in: $extensions}
          {extension_lte: "0x1fffffffffffffffffffffffffffffffffffffff"}
          {extension_gte: "0x8000000000000000000000000000000000000000", extension_lte: "0x9fffffffffffffffffffffffffffffffffffffff"}
        ]}
      ]
    }
    orderBy: id
  ) {
    id
    blockHash
    tickSpacing
    stableswapCenterTick
    stableswapAmplification
    extension
    fee
    poolId
    token0
    token1
  }
}`, subgraphPageSize)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolListUpdater)

type (
	PoolListUpdater struct {
		config *Config

		graphqlClient *graphql.Client

		dataFetchers *dataFetchers

		subgraphCursor subgraphCursor
	}
	subgraphCursor struct {
		id        string
		blockHash string
	}
)

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphql.Client,
) *PoolListUpdater {

	return &PoolListUpdater{
		config:         cfg,
		graphqlClient:  graphqlClient,
		dataFetchers:   NewDataFetchers(ethrpcClient, cfg),
		subgraphCursor: newLastRowInfo(),
	}
}

func newLastRowInfo() subgraphCursor {
	return subgraphCursor{
		id: subgraphInitialStartID,
	}
}

func (u *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]pools.AnyPoolKey, subgraphCursor, error) {
	type poolInitialization struct {
		Id                      string         `json:"id"`
		BlockHash               string         `json:"blockHash"`
		TickSpacing             *uint32        `json:"tickSpacing"`
		StableswapCenterTick    *int32         `json:"stableswapCenterTick"`
		StableswapAmplification *uint8         `json:"stableswapAmplification"`
		Extension               common.Address `json:"extension"`
		Fee                     string         `json:"fee"`
		PoolId                  common.Hash    `json:"poolId"`
		Token0                  common.Address `json:"token0"`
		Token1                  common.Address `json:"token1"`
	}

	allPIs := make([]poolInitialization, 0)
	cursor := u.subgraphCursor

	for {
		req := graphql.NewRequest(subgraphQuery)
		req.Var("startId", cursor.id)
		req.Var("extensions", []common.Address{u.config.Oracle, u.config.Twamm, u.config.MevCapture, u.config.BoostedFeesConcentrated})

		var res struct {
			PoolInitializations []poolInitialization `json:"poolInitializations"`
		}
		if err := u.graphqlClient.Run(ctx, req, &res); err != nil {
			return nil, subgraphCursor{}, fmt.Errorf("request failed: %w", err)
		}

		rawPage := res.PoolInitializations
		pageSize := len(rawPage)

		var page []poolInitialization
		if cursor.blockHash != "" {
			var firstPi *poolInitialization
			if pageSize > 0 {
				firstPi = &rawPage[0]
			}

			if firstPi == nil || firstPi.Id != cursor.id || firstPi.BlockHash != cursor.blockHash {
				logger.WithFields(logger.Fields{
					"dexId": DexType,
					"expected": logger.Fields{
						"id":   cursor.id,
						"hash": cursor.blockHash,
					},
				}).Warn("Subgraph reorged, refetching all pools")

				u.subgraphCursor = newLastRowInfo()

				return u.getNewPoolKeys(ctx)
			}

			page = rawPage[1:]
		} else {
			page = rawPage
		}

		allPIs = slices.Concat(allPIs, page)

		if pageSize > 0 {
			lastPi := rawPage[pageSize-1]
			cursor = subgraphCursor{
				id:        lastPi.Id,
				blockHash: lastPi.BlockHash,
			}
		}

		if pageSize < subgraphPageSize {
			break
		}
	}

	newPoolKeys := make([]pools.AnyPoolKey, 0, len(allPIs))
	for _, pi := range allPIs {
		var poolTypeConfig pools.PoolTypeConfig

		if pi.TickSpacing != nil {
			poolTypeConfig = pools.NewConcentratedPoolTypeConfig(*pi.TickSpacing)
		} else if pi.StableswapAmplification != nil && pi.StableswapCenterTick != nil {
			if *pi.StableswapAmplification == 0 && *pi.StableswapCenterTick == 0 {
				poolTypeConfig = pools.NewFullRangePoolTypeConfig()
			} else {
				poolTypeConfig = pools.NewStableswapPoolTypeConfig(*pi.StableswapCenterTick, *pi.StableswapAmplification)
			}
		} else {
			return nil, subgraphCursor{}, fmt.Errorf("pool %v has unknown pool type config", pi.PoolId)
		}

		fee, err := strconv.ParseUint(pi.Fee, 10, 64)
		if err != nil {
			return nil, subgraphCursor{}, fmt.Errorf("parsing fee: %w", err)
		}

		poolKey := pools.AnyPoolKey{
			PoolKey: pools.NewPoolKey(
				pi.Token0,
				pi.Token1,
				pools.NewPoolConfig(pi.Extension, fee, poolTypeConfig),
			),
		}

		newPoolKeys = append(newPoolKeys, poolKey)
	}

	return newPoolKeys, cursor, nil
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("Start updating pools list...")
	defer func() {
		logger.Infof("Finish updating pools list.")
	}()

	newPoolKeys, newCursor, err := u.getNewPoolKeys(ctx)
	if err != nil {
		return nil, nil, err
	}

	newFetchedPools, err := u.dataFetchers.fetchPools(ctx, newPoolKeys, nil)
	if err != nil {
		return nil, nil, err
	}

	newPools := make([]entity.Pool, 0, len(newFetchedPools))
	for _, pool := range newFetchedPools {
		poolKey := pool.key

		staticExtraBytes, err := json.Marshal(StaticExtra{
			Core:             u.config.Core,
			ExtensionType:    u.config.ExtensionType(poolKey.Extension()),
			PoolKey:          poolKey,
			MevCaptureRouter: u.config.MevCaptureRouter,
		})
		if err != nil {
			return nil, nil, err
		}

		extraBytes, err := json.Marshal(Extra(pool))
		if err != nil {
			return nil, nil, err
		}

		poolAddress, err := poolKey.ToPoolAddress()
		if err != nil {
			return nil, nil, err
		}

		newPools = append(newPools, entity.Pool{
			Address:   poolAddress,
			Exchange:  string(u.config.DexId),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.ZeroToWrappedLower(poolKey.Token0Address().String(), u.config.ChainId),
					Swappable: true,
				},
				{
					Address:   valueobject.ZeroToWrappedLower(poolKey.Token1Address().String(), u.config.ChainId),
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
			Extra:       string(extraBytes),
			BlockNumber: pool.blockNumber,
		})
	}

	u.subgraphCursor = newCursor

	return newPools, nil, nil
}
