package shared

import (
	"context"

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

// getNewPoolsOnchain lists pools from config.Factories (getPools) for deployments without a subgraph,
// then feeds them through initPools like subgraph pools. Metadata.Offsets tracks pools seen per factory.
func (u *PoolsListUpdater) getNewPoolsOnchain(ctx context.Context, metadata Metadata, oldMetadata []byte) ([]entity.Pool, []byte, error) {
	factoryPools := make([][]common.Address, len(u.config.Factories))
	req := u.ethrpcClient.R().SetContext(ctx)
	for i, factory := range u.config.Factories {
		req.AddCall(&ethrpc.Call{ABI: PoolFactoryABI, Target: factory, Method: "getPools"}, []any{&factoryPools[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}

	offsets := make([]int, len(factoryPools))
	copy(offsets, metadata.Offsets)
	var addrs []common.Address
	for i, pools := range factoryPools {
		batch := pools[min(offsets[i], len(pools)):]
		if limit := u.config.NewPoolLimit; limit > 0 && len(batch) > limit {
			batch = batch[:limit]
		}
		offsets[i] = min(offsets[i], len(pools)) + len(batch)
		addrs = append(addrs, batch...)
	}
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
