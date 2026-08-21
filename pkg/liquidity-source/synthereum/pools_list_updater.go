package synthereum

import (
	"context"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools discovers the deployment on-chain, anchored only on the configured
// Finder address: the Finder resolves the PoolRegistry and FixedRateRegistry, each
// registry enumerates its pools, and every pool then reports its own tokens (plus,
// for a wrapper, the lending vault it deposits into). Nothing about the pool set is
// hardcoded, so a newly deployed pool is picked up without a config or library
// change.
//
// The full set is returned on every call rather than only once: pool-service filters
// out pools it already has (allowOverwriteNewPool=false), so re-emitting is
// idempotent and is what lets a later deployment show up at all.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if !common.IsHexAddress(u.config.Finder) {
		return nil, nil, ErrMissingFinder
	}

	registries, err := u.getRegistries(ctx)
	if err != nil {
		return nil, nil, err
	}

	var discovered []discoveredPool
	for poolType, registry := range registries {
		addresses, err := u.getRegisteredPools(ctx, registry)
		if err != nil {
			logger.WithFields(logger.Fields{
				"exchange": u.config.DexID,
				"poolType": poolType,
				"registry": registry.Hex(),
				"error":    err,
			}).Error("failed to enumerate registry")
			return nil, nil, err
		}
		for _, address := range addresses {
			discovered = append(discovered, discoveredPool{address: address, poolType: poolType})
		}
	}

	pools, err := u.buildPools(ctx, discovered)
	if err != nil {
		logger.WithFields(logger.Fields{
			"exchange": u.config.DexID,
			"pools":    len(discovered),
			"error":    err,
		}).Error("failed to read discovered pools")
		return nil, nil, err
	}

	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
		"pools":    len(pools),
	}).Info("finish fetching pools")

	return pools, nil, nil
}

// getRegistries resolves each pool type's registry address from the Finder. A
// registry that is not deployed on this chain resolves to the zero address and is
// skipped, so a chain with only one of the two still discovers normally.
func (u *PoolsListUpdater) getRegistries(ctx context.Context) (map[string]common.Address, error) {
	addresses := make(map[string]*common.Address, len(registriesByPoolType))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for poolType, name := range registriesByPoolType {
		var address common.Address
		addresses[poolType] = &address
		req.AddCall(&ethrpc.Call{
			ABI:    finderABI,
			Target: u.config.Finder,
			Method: finderMethodGetImplementationAddress,
			Params: []any{name},
		}, []any{&address})
	}
	if _, err := req.TryAggregate(); err != nil {
		return nil, err
	}

	registries := make(map[string]common.Address, len(addresses))
	for poolType, address := range addresses {
		if *address != (common.Address{}) {
			registries[poolType] = *address
		}
	}
	if len(registries) == 0 {
		return nil, ErrRegistryUnavailable
	}
	return registries, nil
}

// getRegisteredPools enumerates one registry. The registry indexes pools by
// (synthetic token symbol, collateral, version), so the deployed set is the product
// of the three lists it exposes; combinations that were never deployed simply return
// an empty list.
func (u *PoolsListUpdater) getRegisteredPools(ctx context.Context, registry common.Address) ([]common.Address, error) {
	var (
		symbols     []string
		collaterals []common.Address
		versions    []uint8
	)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI: registryABI, Target: registry.Hex(), Method: registryMethodGetSyntheticTokens,
	}, []any{&symbols})
	req.AddCall(&ethrpc.Call{
		ABI: registryABI, Target: registry.Hex(), Method: registryMethodGetCollaterals,
	}, []any{&collaterals})
	req.AddCall(&ethrpc.Call{
		ABI: registryABI, Target: registry.Hex(), Method: registryMethodGetVersions,
	}, []any{&versions})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	type key struct {
		symbol     string
		collateral common.Address
		version    uint8
	}
	keys := make([]key, 0, len(symbols)*len(collaterals)*len(versions))
	results := make([][]common.Address, 0, cap(keys))
	elemReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	for _, symbol := range symbols {
		for _, collateral := range collaterals {
			for _, version := range versions {
				keys = append(keys, key{symbol, collateral, version})
				results = append(results, nil)
				elemReq.AddCall(&ethrpc.Call{
					ABI:    registryABI,
					Target: registry.Hex(),
					Method: registryMethodGetElements,
					Params: []any{symbol, collateral, version},
				}, []any{&results[len(results)-1]})
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	// TryAggregate: an undeployed (symbol, collateral, version) triple may revert
	// rather than return empty, and that must not fail discovery of the rest.
	if _, err := elemReq.TryAggregate(); err != nil {
		return nil, err
	}

	var addresses []common.Address
	seen := make(map[common.Address]struct{}, len(results))
	for _, result := range results {
		for _, address := range result {
			if _, ok := seen[address]; ok || address == (common.Address{}) {
				continue
			}
			seen[address] = struct{}{}
			addresses = append(addresses, address)
		}
	}
	return addresses, nil
}

// discoveredPool is a pool address enumerated from a registry, before its own
// on-chain metadata has been read.
type discoveredPool struct {
	address  common.Address
	poolType string
}

// buildPools reads every discovered pool's own view of itself -- its collateral and
// synthetic token, and for a wrapper the ERC4626 vault behind its lending module --
// in a single multicall for the whole set rather than one round trip per pool.
// Reserves stay at the zero placeholder and Symbol/Decimals are left unset: the
// tracker and the token registry fill those in later.
func (u *PoolsListUpdater) buildPools(ctx context.Context, discovered []discoveredPool) ([]entity.Pool, error) {
	if len(discovered) == 0 {
		return nil, nil
	}

	type poolInfo struct {
		collateralToken, syntheticToken common.Address
		// lendingModule() returns a (moduleId, bearingToken) pair; ethrpc unpacks a
		// multi-return call into a single struct matched against the ABI's named
		// outputs, not into separate destinations.
		lendingModule struct {
			ModuleId     string
			BearingToken common.Address
		}
	}

	infos := make([]poolInfo, len(discovered))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range discovered {
		address, isWrapper := discovered[i].address.Hex(), discovered[i].poolType == PoolTypeWrapper
		poolABI := multiLpPoolABI
		if isWrapper {
			poolABI = wrapperABI
		}
		req.AddCall(&ethrpc.Call{
			ABI: poolABI, Target: address, Method: poolMethodCollateralToken,
		}, []any{&infos[i].collateralToken})
		req.AddCall(&ethrpc.Call{
			ABI: poolABI, Target: address, Method: poolMethodSyntheticToken,
		}, []any{&infos[i].syntheticToken})
		if isWrapper {
			req.AddCall(&ethrpc.Call{
				ABI: wrapperABI, Target: address, Method: wrapperMethodLendingModule,
			}, []any{&infos[i].lendingModule})
		}
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(discovered))
	for i := range discovered {
		address, poolType := discovered[i].address, discovered[i].poolType
		info := &infos[i]

		if info.collateralToken == (common.Address{}) || info.syntheticToken == (common.Address{}) {
			return nil, fmt.Errorf("%w: pool %s reported a zero token", ErrInvalidToken, address.Hex())
		}

		staticExtra := StaticExtra{PoolType: poolType}
		if poolType == PoolTypeWrapper {
			if info.lendingModule.BearingToken == (common.Address{}) {
				return nil, fmt.Errorf("%w: pool %s", ErrMissingVault, address.Hex())
			}
			staticExtra.Vault = hexutil.Encode(info.lendingModule.BearingToken[:])
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(address[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{reserveZero, reserveZero},
			// Token order is the protocol's own: index 0 collateral, index 1 synthetic.
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(info.collateralToken[:]), Swappable: true},
				{Address: hexutil.Encode(info.syntheticToken[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}
	return pools, nil
}
