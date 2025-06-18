package bin

import (
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
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

	chainID := valueobject.ChainID(u.config.ChainID)
	for _, p := range subgraphPools {
		token0Decimals, err := kutils.Atou[uint8](p.TokenX.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		token1Decimals, err := kutils.Atou[uint8](p.TokenY.Decimals)
		if err != nil {
			return nil, metadataBytes, err
		}
		tokens := []*entity.PoolToken{
			{Address: p.TokenX.ID, Decimals: token0Decimals, Swappable: true},
			{Address: p.TokenY.ID, Decimals: token1Decimals, Swappable: true},
		}
		for idx, token := range tokens {
			if token.Address == valueobject.ZeroAddress {
				tokens[idx].Address = strings.ToLower(valueobject.WrappedNativeMap[chainID])
			}
		}

		params := common.FromHex(p.Parameters)

		feePct, _ := strconv.ParseFloat(p.BaseFeePct, 64)
		fee := uint32(feePct * BasisPoint)

		staticExtra := StaticExtra{
			HasSwapPermissions: shared.HasSwapPermissions(params),
			IsNative:           [2]bool{p.TokenX.ID == valueobject.ZeroAddress, p.TokenY.ID == valueobject.ZeroAddress},
			Fee:                fee,
			Parameters:         p.Parameters,
			BinStep:            GetBinStep(params),
			HooksAddress:       common.HexToAddress(p.Hooks),
			Permit2Address:     common.HexToAddress(u.config.Permit2Address),
			VaultAddress:       common.HexToAddress(u.config.VaultAddress),
			PoolManagerAddress: common.HexToAddress(u.config.BinPoolManagerAddress),
			Multicall3Address:  common.HexToAddress(u.config.Multicall3Address),
			IsDynamicFee:       shared.IsDynamicFee(fee),
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, metadataBytes, err
		}

		hook, _ := GetHook(staticExtra.HooksAddress)
		pool := entity.Pool{
			Address:     p.ID,
			Exchange:    hook.GetExchange(),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, pool)
	}

	// Update metadata
	if len(subgraphPools) > 0 {
		lastCreatedAtTimestamp, err := strconv.Atoi(subgraphPools[len(subgraphPools)-1].Timestamp)
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

func (u *PoolsListUpdater) getPoolsList(ctx context.Context, lastCreatedAtTimestamp int, first int) ([]SubgraphPool,
	error) {
	req := graphqlpkg.NewRequest(getPoolsListQuery(lastCreatedAtTimestamp, first))

	var response struct {
		Pools []SubgraphPool `json:"lbpairs"`
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

func GetBinStep(params []byte) uint16 {
	res := new(uint256.Int).SetBytes32(params)
	res.Rsh(res, _OFFSET_BIN_STEP)
	res.And(res, _MASK16)

	return uint16(res.Uint64())
}
