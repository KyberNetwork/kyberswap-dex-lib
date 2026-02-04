package ekubov3

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The extra bounds on `extension` are used to include any pool keys that have no beforeSwap and afterSwap hook, see https://github.com/EkuboProtocol/evm-contracts/blob/665e8333e550003b68a94d8482cc9fda438a2bf1/src/types/callPoints.sol
const subgraphQuery = `
query NewPools(
  $startBlockNumber: BigInt!
  $coreAddress: Bytes!
  $extensions: [Bytes!]
) {
  poolInitializations(
    where: {
      and: [
        {blockNumber_gte: $startBlockNumber, coreAddress: $coreAddress}
        {or: [
          {extension_in: $extensions}
          {extension_lte: "0x1fffffffffffffffffffffffffffffffffffffff"}
          {extension_gte: "0x8000000000000000000000000000000000000000", extension_lte: "0x9fffffffffffffffffffffffffffffffffffffff"}
        ]}
      ]
    }
    orderBy: blockNumber
  ) {
    blockNumber
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
}`

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolListUpdater)

type PoolListUpdater struct {
	config *Config

	graphqlClient *graphql.Client

	dataFetchers *dataFetchers

	startBlockNumber uint64
	startBlockHash   common.Hash
}

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphql.Client,
) *PoolListUpdater {

	return &PoolListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
		dataFetchers:  NewDataFetchers(ethrpcClient, cfg),
	}
}

func (u *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]pools.AnyPoolKey, error) {
	req := graphql.NewRequest(subgraphQuery)
	req.Var("coreAddress", u.config.Core)
	req.Var("extensions", []common.Address{u.config.Oracle, u.config.Twamm, u.config.MevCapture, u.config.BoostedFeesConcentrated})
	req.Var("startBlockNumber", u.startBlockNumber)

	var res struct {
		PoolInitializations []struct {
			BlockNumber             string         `json:"blockNumber"`
			BlockHash               string         `json:"blockHash"`
			TickSpacing             *uint32        `json:"tickSpacing"`
			StableswapCenterTick    *int32         `json:"stableswapCenterTick"`
			StableswapAmplification *uint8         `json:"stableswapAmplification"`
			Extension               common.Address `json:"extension"`
			Fee                     string         `json:"fee"`
			PoolId                  common.Hash    `json:"poolId"`
			Token0                  common.Address `json:"token0"`
			Token1                  common.Address `json:"token1"`
		} `json:"poolInitializations"`
	}
	err := u.graphqlClient.Run(ctx, req, &res)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	pis := res.PoolInitializations

	if len(pis) == 0 {
		return nil, nil
	}

	if u.startBlockNumber != 0 {
		firstPi := pis[0]

		firstBlockNumber, err := kutils.Atou[uint64](firstPi.BlockNumber)
		if err != nil {
			return nil, fmt.Errorf("parsing first blockNumber: %w", err)
		}

		if firstBlockNumber != u.startBlockNumber || common.HexToHash(firstPi.BlockHash) != u.startBlockHash {
			logger.WithFields(logger.Fields{
				"dexId": DexType,
				"expected": logger.Fields{
					"number": u.startBlockNumber,
					"hash":   u.startBlockHash,
				},
				"actual": logger.Fields{
					"number": firstBlockNumber,
					"hash":   common.HexToHash(firstPi.BlockHash),
				},
			}).Warn("Subgraph reorged, refetching all pools")

			u.startBlockNumber = 0
			u.startBlockHash = common.Hash{}

			return u.getNewPoolKeys(ctx)
		}

		firstNewDataIdx := 1
		for i, pi := range pis[1:] {
			blockNumber, err := kutils.Atou[uint64](pi.BlockNumber)
			if err != nil {
				return nil, fmt.Errorf("parsing blockNumber: %w", err)
			}

			if blockNumber > firstBlockNumber {
				firstNewDataIdx = i + 1
			}
		}

		pis = pis[firstNewDataIdx:]
	}

	if len(pis) == 0 {
		return nil, nil
	}

	lastPi := pis[len(pis)-1]
	lastBlockNumber, err := kutils.Atou[uint64](lastPi.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("parsing last blockNumber: %w", err)
	}

	u.startBlockNumber = lastBlockNumber
	u.startBlockHash = common.HexToHash(lastPi.BlockHash)

	newPoolKeys := make([]pools.AnyPoolKey, 0, len(pis))
	for _, pi := range pis {
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
			return nil, fmt.Errorf("pool %v has unknown pool type config", pi.PoolId)
		}

		fee, err := strconv.ParseUint(pi.Fee, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing fee: %w", err)
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

	return newPoolKeys, nil
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	logger.Infof("Start updating pools list...")
	defer func() {
		logger.Infof("Finish updating pools list.")
	}()

	newPoolKeys, err := u.getNewPoolKeys(ctx)
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

	return newPools, nil, nil
}
