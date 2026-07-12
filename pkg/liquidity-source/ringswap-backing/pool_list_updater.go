package ringswapbacking

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: config, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(
	ctx context.Context,
	metadataBytes []byte,
) ([]entity.Pool, []byte, error) {
	if err := u.config.validate(); err != nil {
		return nil, metadataBytes, err
	}

	metadata := PoolsListUpdaterMetadata{}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}
	known, knownPairs, err := knownSourceSet(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	candidates := make([]RouterConfig, 0, len(u.config.Routers))
	seen := make(map[string]struct{}, len(known)+len(u.config.Routers))
	for router := range known {
		seen[router] = struct{}{}
	}
	for _, configured := range u.config.Routers {
		router := strings.ToLower(common.HexToAddress(configured.Address).Hex())
		if _, exists := seen[router]; exists {
			continue
		}
		configured.Address = router
		candidates = append(candidates, configured)
		seen[router] = struct{}{}
	}
	if len(candidates) == 0 {
		return nil, metadataBytes, nil
	}

	origin0s := make([]common.Address, len(candidates))
	origin1s := make([]common.Address, len(candidates))
	wrapper0s := make([]common.Address, len(candidates))
	wrapper1s := make([]common.Address, len(candidates))
	pairs := make([]common.Address, len(candidates))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, configured := range candidates {
		router := configured.Address
		req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "origin0"}, []any{&origin0s[i]})
		req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "origin1"}, []any{&origin1s[i]})
		req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "token0"}, []any{&wrapper0s[i]})
		req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "token1"}, []any{&wrapper1s[i]})
		req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "pair"}, []any{&pairs[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, metadataBytes, err
	}

	zero := common.Address{}
	pools := make([]entity.Pool, 0, len(candidates))
	for i, configured := range candidates {
		callIndex := i * 5
		if len(resp.Result) <= callIndex+4 || !resp.Result[callIndex] || !resp.Result[callIndex+1] ||
			!resp.Result[callIndex+2] || !resp.Result[callIndex+3] || !resp.Result[callIndex+4] ||
			origin0s[i] == zero || origin1s[i] == zero || wrapper0s[i] == zero ||
			wrapper1s[i] == zero || pairs[i] == zero || origin0s[i] == origin1s[i] ||
			wrapper0s[i] == wrapper1s[i] {
			continue
		}

		pairAddress := strings.ToLower(hexutil.Encode(pairs[i][:]))
		if _, exists := knownPairs[pairAddress]; exists {
			return nil, metadataBytes, ErrDuplicatePair
		}
		staticExtra, err := json.Marshal(StaticExtra{
			RouterAddress:       configured.Address,
			PairAddress:         pairAddress,
			Wrapper0:            strings.ToLower(hexutil.Encode(wrapper0s[i][:])),
			Wrapper1:            strings.ToLower(hexutil.Encode(wrapper1s[i][:])),
			ReplaceOrdinaryPair: configured.ReplaceOrdinaryPair,
			NoRecallGasToken0:   configured.NoRecallGasToken0,
			NoRecallGasToken1:   configured.NoRecallGasToken1,
			RecallGasToken0:     configured.RecallGasToken0,
			RecallGasToken1:     configured.RecallGasToken1,
		})
		if err != nil {
			return nil, metadataBytes, err
		}
		pools = append(pools, entity.Pool{
			// This source replaces the ordinary Ring source for the configured Pair. It handles
			// both direct hot-backing execution and recall-backed execution as one inventory.
			Address:   pairAddress,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(hexutil.Encode(origin0s[i][:])), Swappable: true},
				{Address: strings.ToLower(hexutil.Encode(origin1s[i][:])), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
		known[configured.Address] = struct{}{}
		knownPairs[pairAddress] = struct{}{}
	}

	newMetadata := PoolsListUpdaterMetadata{
		KnownRouters: make([]string, 0, len(known)),
		KnownPairs:   make([]string, 0, len(knownPairs)),
	}
	for router := range known {
		newMetadata.KnownRouters = append(newMetadata.KnownRouters, router)
	}
	sort.Strings(newMetadata.KnownRouters)
	for pairAddress := range knownPairs {
		newMetadata.KnownPairs = append(newMetadata.KnownPairs, pairAddress)
	}
	sort.Strings(newMetadata.KnownPairs)
	newMetadataBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadataBytes, nil
}

func knownSourceSet(
	metadata PoolsListUpdaterMetadata,
) (map[string]struct{}, map[string]struct{}, error) {
	if len(metadata.KnownRouters) != len(metadata.KnownPairs) {
		return nil, nil, ErrInvalidMetadata
	}
	knownRouters := make(map[string]struct{}, len(metadata.KnownRouters))
	for _, routerAddress := range metadata.KnownRouters {
		if !common.IsHexAddress(routerAddress) {
			return nil, nil, ErrInvalidMetadata
		}
		normalized := strings.ToLower(common.HexToAddress(routerAddress).Hex())
		if common.HexToAddress(normalized) == (common.Address{}) {
			return nil, nil, ErrInvalidMetadata
		}
		if _, exists := knownRouters[normalized]; exists {
			return nil, nil, ErrInvalidMetadata
		}
		knownRouters[normalized] = struct{}{}
	}
	knownPairs := make(map[string]struct{}, len(metadata.KnownPairs))
	for _, pairAddress := range metadata.KnownPairs {
		if !common.IsHexAddress(pairAddress) {
			return nil, nil, ErrInvalidMetadata
		}
		normalized := strings.ToLower(common.HexToAddress(pairAddress).Hex())
		if common.HexToAddress(normalized) == (common.Address{}) {
			return nil, nil, ErrInvalidMetadata
		}
		if _, exists := knownPairs[normalized]; exists {
			return nil, nil, ErrInvalidMetadata
		}
		knownPairs[normalized] = struct{}{}
	}
	return knownRouters, knownPairs, nil
}
