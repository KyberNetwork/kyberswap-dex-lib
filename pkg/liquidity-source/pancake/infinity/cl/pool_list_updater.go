package cl

import (
	"cmp"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config        *Config
		ethrpcClient  *ethrpc.Client
		graphqlClient *graphqlpkg.Client
	}

	Metadata struct {
		LastCreatedAtTimestamp int    `json:"lastCreatedAtTimestamp"`
		LastProcessedPoolId    string `json:"lastProcessedPoolID"`
		LastBlock              uint64 `json:"lastBlock,omitempty"` // log discovery cursor
	}
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		err := json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	if u.config.SubgraphAPI == "" {
		return u.getNewPoolsFromLogs(ctx, metadata)
	}

	subgraphPools, err := u.getPoolsList(ctx, metadata.LastCreatedAtTimestamp, u.config.NewPoolLimit)
	if err != nil {
		return nil, metadataBytes, err
	}

	// Currently disable filter which will lead to dup process getnewpools, but it's oke
	// If we enable, we have to change logic of for loop in pool-service where "len(poolsList) < newPoolLimit"

	// subgraphPools = lo.Filter(subgraphPools, func(p SubgraphPool, _ int) bool {
	//	return p.ID != metadata.LastProcessedPoolId
	// })

	pools := make([]entity.Pool, 0, len(subgraphPools))
	hooks := make([]common.Address, 0, len(subgraphPools))
	for _, p := range subgraphPools {
		hooks = append(hooks, common.HexToAddress(p.Hooks))
	}
	classifiedStableHooks := classifyStableHooks(ctx, u.ethrpcClient, u.config.StableHookFactories, hooks)

	chainID := valueobject.ChainID(u.config.ChainID)
	for _, p := range subgraphPools {
		token0Decimals, err := kutils.Atou[uint8](p.Token0.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		token1Decimals, err := kutils.Atou[uint8](p.Token1.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		tokens := []*entity.PoolToken{
			{Address: p.Token0.ID, Decimals: token0Decimals, Swappable: true},
			{Address: p.Token1.ID, Decimals: token1Decimals, Swappable: true},
		}
		for idx, token := range tokens {
			if token.Address == valueobject.ZeroAddress {
				tokens[idx].Address = strings.ToLower(valueobject.WrappedNativeMap[chainID])
			}
		}

		fee, err := kutils.Atou[uint32](p.Fee)
		if err != nil {
			return nil, metadataBytes, err
		}

		params := common.FromHex(p.Parameters)

		staticExtra := StaticExtra{
			HasSwapPermissions: shared.HasSwapPermissions(params),
			IsNative:           [2]bool{p.Token0.ID == valueobject.ZeroAddress, p.Token1.ID == valueobject.ZeroAddress},
			Fee:                fee,
			Parameters:         p.Parameters,
			TickSpacing:        GetTickSpacing(params),
			HooksAddress:       common.HexToAddress(p.Hooks),
			Permit2Address:     common.HexToAddress(u.config.Permit2Address),
			VaultAddress:       common.HexToAddress(u.config.VaultAddress),
			PoolManagerAddress: common.HexToAddress(u.config.CLPoolManagerAddress),
			Multicall3Address:  common.HexToAddress(u.config.Multicall3Address),
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, metadataBytes, err
		}

		exchange := ""
		if classifiedStableHooks[staticExtra.HooksAddress] {
			exchange = valueobject.ExchangePancakeInfinityCLStable
		} else {
			hook, _ := GetHook(staticExtra.HooksAddress, &HookParam{Cfg: u.config})
			exchange = hook.GetExchange()
		}
		pool := entity.Pool{
			Address:     p.ID,
			SwapFee:     float64(fee),
			Exchange:    exchange,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
			Extra:       "{}",
		}
		pools = append(pools, pool)
	}

	// Update metadata
	if len(subgraphPools) > 0 {
		lastCreatedAtTimestamp, err := strconv.Atoi(subgraphPools[len(subgraphPools)-1].CreatedAtTimestamp)
		if err != nil {
			return nil, metadataBytes, err
		}

		metadata.LastCreatedAtTimestamp = lastCreatedAtTimestamp
		metadata.LastProcessedPoolId = subgraphPools[len(subgraphPools)-1].ID
		metadataBytes, err = json.Marshal(metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
	}

	logger.WithFields(logger.Fields{
		"dexId": u.config.DexID,
		"pools": len(pools),
	}).Info("finished getting new pools")

	return pools, metadataBytes, nil
}

// getNewPoolsFromLogs discovers pools from the manager's Initialize logs, for managers without a subgraph.
func (u *PoolsListUpdater) getNewPoolsFromLogs(ctx context.Context, metadata Metadata) ([]entity.Pool, []byte, error) {
	head, err := u.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return nil, nil, err
	}

	decoder := NewPoolFactory(u.config, u.ethrpcClient)
	from := max(metadata.LastBlock+1, u.config.StartBlock)
	blockRange := cmp.Or(u.config.LogBlockRange, shared.DefaultLogBlockRange)
	var pools []entity.Pool
	for from <= head && len(pools) < u.config.NewPoolLimit {
		to := min(from+blockRange-1, head)
		logs, err := shared.FilterLogs(ctx, u.ethrpcClient, common.HexToAddress(u.config.CLPoolManagerAddress),
			[][]common.Hash{{shared.CLPoolManagerABI.Events["Initialize"].ID}}, from, to, blockRange)
		if err != nil {
			return nil, nil, err
		}
		for _, l := range logs {
			p, err := decoder.DecodePoolCreated(l)
			if err != nil {
				return nil, nil, err
			}
			pools = append(pools, *p)
		}
		from = to + 1
	}
	metadata.LastBlock = from - 1

	metadataBytes, err := json.Marshal(metadata)
	logger.WithFields(logger.Fields{"dexId": u.config.DexID, "pools": len(pools)}).Info("finished getting new pools")
	return pools, metadataBytes, err
}

func (u *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp int, first int) ([]SubgraphPool,
	error) {
	req := graphqlpkg.NewRequest(getPoolsListQuery(lastCreatedAtTimestamp, first))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := u.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"dexId": u.config.DexID,
			"error": err,
		}).Errorf("failed to query subgraph")
		return nil, err
	}

	return response.Pools, nil
}

func GetTickSpacing(params []byte) uint64 {
	res := new(uint256.Int).SetBytes32(params)
	res.Rsh(res, _OFFSET_TICK_SPACING)
	res.And(res, _MASK24)

	return res.Uint64()
}
