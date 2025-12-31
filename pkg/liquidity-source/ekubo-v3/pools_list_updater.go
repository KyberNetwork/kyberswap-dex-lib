package ekubov3

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/pools"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

const (
	reorgBackfillBlocks = 12
	subgraphQuery       = `
query NewPools($startBlock: BigInt!, $coreAddress: Bytes!, $extensions: [Bytes!]) {
  poolInitializations(
    where: {blockNumber_gte: $startBlock, coreAddress: $coreAddress, extension_in: $extensions}, orderBy: blockNumber
  ) {
    blockNumber
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
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolListUpdater)

type (
	PoolListUpdater struct {
		config *Config

		graphqlClient *graphql.Client
		graphqlReq    *graphql.Request

		dataFetchers *dataFetchers

		registeredPools     map[string]bool
		supportedExtensions map[common.Address]ExtensionType

		startBlock int64
	}

	poolData struct {
		BlockNumber             string         `json:"blockNumber"`
		TickSpacing             *uint32        `json:"tickSpacing"`
		StableswapCenterTick    *int32         `json:"stableswapCenterTick"`
		StableswapAmplification *uint8         `json:"stableswapAmplification"`
		Extension               common.Address `json:"extension"`
		Fee                     string         `json:"fee"`
		PoolId                  common.Hash    `json:"poolId"`
		Token0                  common.Address `json:"token0"`
		Token1                  common.Address `json:"token1"`
	}

	getAllPoolsResult struct {
		Data []poolData `json:"poolInitializations"`
	}
)

func (u *PoolListUpdater) getNewPoolKeys(ctx context.Context) ([]*pools.PoolKey[pools.PoolTypeConfig], error) {
	u.graphqlReq.Var("startBlock", max(u.startBlock-reorgBackfillBlocks, 0))

	var res getAllPoolsResult
	err := u.graphqlClient.Run(ctx, u.graphqlReq, &res)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	newPoolKeys := make([]*pools.PoolKey[pools.PoolTypeConfig], 0)
	for _, p := range res.Data {
		var poolTypeConfig pools.PoolTypeConfig

		if p.TickSpacing != nil {
			poolTypeConfig = pools.NewConcentratedPoolTypeConfig(*p.TickSpacing)
		} else if p.StableswapAmplification != nil && p.StableswapCenterTick != nil {
			if *p.StableswapAmplification == 0 && *p.StableswapCenterTick == 0 {
				poolTypeConfig = pools.NewFullRangePoolTypeConfig()
			} else {
				poolTypeConfig = pools.NewStableswapPoolTypeConfig(*p.StableswapCenterTick, *p.StableswapAmplification)
			}
		} else {
			return nil, fmt.Errorf("pool %v has unknown pool type config", p.PoolId)
		}

		fee, err := strconv.ParseUint(p.Fee, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing fee: %w", err)
		}

		poolKey := pools.NewPoolKey(
			p.Token0,
			p.Token1,
			pools.NewPoolConfig(p.Extension, fee, poolTypeConfig),
		)

		if u.registeredPools[poolKey.StringId()] {
			continue
		}

		blockNumber, err := strconv.ParseInt(p.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing blockNumber: %w", err)
		}

		u.startBlock = blockNumber + 1

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

	newEkuboPools, err := u.dataFetchers.fetchPools(ctx, newPoolKeys, nil)
	if err != nil {
		return nil, nil, err
	}

	newPools := make([]entity.Pool, 0, len(newPoolKeys))
	for i, poolKey := range newPoolKeys {
		extensionType, ok := u.supportedExtensions[poolKey.Extension()]
		if !ok {
			logger.WithFields(logger.Fields{
				"poolKey": poolKey,
			}).Warn("skipping pool key with unknown extension")
			continue
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			Core:          u.config.Core,
			ExtensionType: extensionType,
			PoolKey:       &pools.AnyPoolKey{PoolKey: poolKey},
		})
		if err != nil {
			return nil, nil, err
		}

		extraBytes, err := json.Marshal(Extra(newEkuboPools[i]))
		if err != nil {
			return nil, nil, err
		}

		poolAddress, err := poolKey.ToPoolAddress()
		if err != nil {
			return nil, nil, err
		}

		newPools = append(newPools, entity.Pool{
			Address:   poolAddress,
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{
					Address:   FromEkuboAddress(poolKey.Token0.String(), u.config.ChainId),
					Swappable: true,
				},
				{
					Address:   FromEkuboAddress(poolKey.Token1.String(), u.config.ChainId),
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
			Extra:       string(extraBytes),
			BlockNumber: newEkuboPools[i].blockNumber,
		})

		u.registeredPools[poolKey.StringId()] = true
	}

	return newPools, nil, nil
}

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphql.Client,
) *PoolListUpdater {
	req := graphql.NewRequest(subgraphQuery)

	req.Var("coreAddress", cfg.Core)
	req.Var("extensions", []common.Address{{}, cfg.Oracle, cfg.Twamm, cfg.MevCapture})

	return &PoolListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
		graphqlReq:    req,
		dataFetchers:  NewDataFetchers(ethrpcClient, cfg),

		registeredPools:     make(map[string]bool),
		supportedExtensions: cfg.SupportedExtensions,

		startBlock: 0,
	}
}
