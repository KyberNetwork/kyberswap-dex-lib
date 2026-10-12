package shared

import (
	"context"
	"maps"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// HookTypeUnknown marks a pool whose hook contract is not supported, so the tracker disables it.
const HookTypeUnknown HookType = "UNKNOWN"

// getNewPoolsOnchain lists pools from config.Factories for deployments without a subgraph, then feeds them
// through initPools like subgraph pools. Relisting is not needed: see ListFactoryPools.
func (u *PoolsListUpdater) getNewPoolsOnchain(ctx context.Context, metadata Metadata, oldMetadata []byte) ([]entity.Pool, []byte, error) {
	factoryPools, offsets, err := ListFactoryPools(ctx, u.ethrpcClient, u.config.Factories, metadata.Offsets,
		u.config.NewPoolLimit)
	if err != nil {
		return nil, nil, err
	}
	addrs := lo.Flatten(factoryPools)
	if len(addrs) == 0 {
		return nil, oldMetadata, nil
	}
	metadata.Offsets = offsets

	subgraphPools, err := u.queryVault(ctx, addrs)
	if err != nil {
		return nil, nil, err
	}
	pools, err := u.initPools(ctx, subgraphPools)
	if err != nil {
		return nil, nil, err
	}
	metadataBytes, err := json.Marshal(metadata)
	return pools, metadataBytes, err
}

// ListFactoryPools returns, per factory, the pools added since offsets (getPoolCount + getPoolsInRange, at most
// limit per factory when limit > 0), and the advanced offsets. A factory's pool list is append-only, so the
// cursor only advances and never needs a relist. offsets is keyed by lowercased factory address and not mutated.
func ListFactoryPools(ctx context.Context, client *ethrpc.Client, factories []string, offsets map[string]int,
	limit int) ([][]common.Address, map[string]int, error) {
	keys := make([]string, len(factories))
	counts := make([]*big.Int, len(factories))
	req := client.R().SetContext(ctx)
	for i, factory := range factories {
		keys[i] = strings.ToLower(factory)
		req.AddCall(&ethrpc.Call{ABI: PoolFactoryABI, Target: factory, Method: "getPoolCount"}, []any{&counts[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}

	newOffsets := maps.Clone(offsets)
	if newOffsets == nil {
		newOffsets = make(map[string]int, len(factories))
	}
	starts := make([]int, len(factories))
	factoryPools := make([][]common.Address, len(factories))
	req = client.R().SetContext(ctx)
	var hasNew bool
	for i, factory := range factories {
		starts[i] = newOffsets[keys[i]]
		n := int(counts[i].Uint64()) - starts[i]
		if limit > 0 {
			n = min(n, limit)
		}
		if n <= 0 {
			continue
		}
		hasNew = true
		req.AddCall(&ethrpc.Call{ABI: PoolFactoryABI, Target: factory, Method: "getPoolsInRange",
			Params: []any{big.NewInt(int64(starts[i])), big.NewInt(int64(n))}}, []any{&factoryPools[i]})
	}
	if !hasNew {
		return factoryPools, newOffsets, nil
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}
	for i, pools := range factoryPools {
		newOffsets[keys[i]] = starts[i] + len(pools)
	}
	return factoryPools, newOffsets, nil
}

// queryVault reads tokens, hook and buffer (ERC4626 wrapper) info that the subgraph would otherwise provide.
func (u *PoolsListUpdater) queryVault(ctx context.Context, addrs []common.Address) ([]*SubgraphPool, error) {
	vault := hexutil.Encode(Vault(u.config.ChainID, u.config.DexID).Bytes())
	tokens := make([][]common.Address, len(addrs))
	hooks := make([]HooksConfigRPC, len(addrs))
	req := u.ethrpcClient.R().SetContext(ctx)
	for i, addr := range addrs {
		params := []any{addr}
		req.AddCall(&ethrpc.Call{ABI: VaultExplorerABI, Target: vault, Method: "getPoolTokens", Params: params},
			[]any{&tokens[i]})
		req.AddCall(&ethrpc.Call{ABI: VaultExplorerABI, Target: vault, Method: VaultMethodGetHooksConfig, Params: params},
			[]any{&hooks[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	// A token with a buffer is an ERC4626 wrapper; getBufferAsset returns its underlying, or zero.
	allTokens := lo.Uniq(lo.Flatten(tokens))
	assets := make([]common.Address, len(allTokens))
	req = u.ethrpcClient.R().SetContext(ctx)
	for i, token := range allTokens {
		req.AddCall(&ethrpc.Call{ABI: VaultExplorerABI, Target: vault, Method: VaultMethodGetBufferAsset,
			Params: []any{token}}, []any{&assets[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	assetOf := make(map[common.Address]common.Address, len(allTokens))
	for i, token := range allTokens {
		assetOf[token] = assets[i]
	}

	pools := make([]*SubgraphPool, len(addrs))
	for i, addr := range addrs {
		pool := &SubgraphPool{Address: hexutil.Encode(addr[:])}
		if hook := hooks[i].HooksConfigData.HooksContract; hook != valueobject.AddrZero {
			pool.Hook.Address = hexutil.Encode(hook[:])
			// Only the pool itself (reCLAMM) is a supported hook contract here.
			pool.Hook.Type = lo.Ternary(hook == addr, ReClammHookType, HookTypeUnknown)
		}
		for _, token := range tokens[i] {
			t := SubgraphToken{Address: hexutil.Encode(token[:])}
			if asset := assetOf[token]; asset != valueobject.AddrZero {
				t.CanUseBufferForSwaps = true
				t.UnderlyingToken.Address = hexutil.Encode(asset[:])
			}
			pool.PoolTokens = append(pool.PoolTokens, t)
		}
		pools[i] = pool
	}
	return pools, nil
}
