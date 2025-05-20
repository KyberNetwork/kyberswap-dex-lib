package poolfactory

import (
	"context"
	"math/big"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	dexlibprivate "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source"
	aevmpool "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/aevm-pool"
	aevmpoolwrapper "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/aevm-pool/wrapper"
	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	usecasetypes "github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	ErrPoolTypeFactoryNotFound = errors.New("pool type factory not found")
)

type PoolFactory struct {
	config              Config
	ethClient           bind.ContractBackend
	client              aevmclient.Client
	balanceSlotsUseCase erc20balanceslot.ICache

	lock sync.Mutex
}

func NewPoolFactory(config Config, ethClient bind.ContractBackend, aevmClient aevmclient.Client,
	balanceSlotsUseCase erc20balanceslot.ICache) *PoolFactory {

	return &PoolFactory{
		config:              config,
		ethClient:           ethClient,
		client:              aevmClient,
		balanceSlotsUseCase: balanceSlotsUseCase,
	}

}

func (f *PoolFactory) ApplyConfig(config Config) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.config = config
}

func (f *PoolFactory) NewPools(ctx context.Context, pools []*entity.Pool,
	stateRoot common.Hash) []poolpkg.IPoolSimulator {
	span, _ := tracer.StartSpanFromContext(ctx, "poolFactory.NewPools")
	defer span.End()

	poolSims := make([]poolpkg.IPoolSimulator, 0, len(pools))
	f.newPools(ctx, pools, stateRoot, func(pool poolpkg.IPoolSimulator) {
		poolSims = append(poolSims, pool)
	})
	return poolSims
}

func (f *PoolFactory) NewPoolByAddress(ctx context.Context, pools []*entity.Pool,
	stateRoot common.Hash) map[string]poolpkg.IPoolSimulator {
	span, _ := tracer.StartSpanFromContext(ctx, "poolFactory.NewPoolByAddress")
	defer span.End()

	poolByAddress := make(map[string]poolpkg.IPoolSimulator, len(pools))
	f.newPools(ctx, pools, stateRoot, func(pool poolpkg.IPoolSimulator) {
		poolByAddress[pool.GetAddress()] = pool
	})
	return poolByAddress
}

func (f *PoolFactory) newPools(ctx context.Context, pools []*entity.Pool,
	stateRoot common.Hash, yieldPool func(poolpkg.IPoolSimulator)) {
	basePoolMap := make(map[string]poolpkg.IPoolSimulator)
	factoryParams := poolpkg.FactoryParams{
		BasePoolMap: basePoolMap,
		ChainID:     f.config.ChainID,
		EthClient:   f.ethClient,
	}

	for _, basePoolTypes := range basePoolTypesSets {
		for _, pool := range pools {
			if !matchesAny(pool, basePoolTypes) {
				continue
			}

			poolSim, err := f.newPool(*pool, factoryParams, stateRoot)
			if err != nil {
				logger.Debugf(ctx, "%+v", err)
				continue
			}
			basePoolMap[pool.Address] = poolSim
		}
	}

	for _, pool := range pools {
		poolSim, ok := basePoolMap[pool.Address]
		if !ok {
			var err error
			poolSim, err = f.newPool(*pool, factoryParams, stateRoot)
			if err != nil {
				logger.Debugf(ctx, "%+v", err)
				continue
			}
		}
		yieldPool(poolSim)
	}
}

// newPool receives entity.Pool, based on its type to return matched factory method
// if there is no matched factory method, it returns ErrPoolTypeFactoryNotFound
func (f *PoolFactory) newPool(entityPool entity.Pool, factoryParams poolpkg.FactoryParams,
	stateRoot common.Hash) (pool poolpkg.IPoolSimulator, err error) {
	factoryParams.EntityPool = entityPool

	if poolFactory := poolpkg.Factory(entityPool.Type); poolFactory != nil {
		if pool, err = poolFactory(factoryParams); err == nil {
			return pool, nil
		}
	}

	// TODO: should extend DexUseAEVM config
	// - Add support for specifying which DEXes can use AEVM without requiring UseAEVM=true
	// - Consider adding a new field to configure AEVM behavior per DEX
	shouldUseAEVM := f.config.DexUseAEVM[entityPool.Type] && (f.config.UseAEVM ||
		entityPool.Type == pooltypes.PoolTypes.UniswapV4 ||
		entityPool.Type == pooltypes.PoolTypes.PancakeInfinityBin ||
		entityPool.Type == pooltypes.PoolTypes.PancakeInfinityCL)

	if shouldUseAEVM {
		if aevmPoolFactory := aevmpool.Factory(entityPool.Type); aevmPoolFactory != nil {
			return f.newAEVMPoolWrapper(entityPool, aevmPoolFactory, stateRoot)
		}
	}

	return nil, errors.WithMessagef(ErrPoolTypeFactoryNotFound, "%s (%s/%s)",
		entityPool.Address, entityPool.Exchange, entityPool.Type)
}

func (f *PoolFactory) newAEVMPoolWrapper(entityPool entity.Pool, poolFactory aevmpool.FactoryFn,
	stateRoot common.Hash) (*aevmpoolwrapper.PoolWrapper, error) {
	unimplementedPool := dexlibprivate.NewUnimplementedPool(entityPool.Address, entityPool.Exchange, entityPool.Type)

	var balanceSlots = make(map[common.Address]*types.ERC20BalanceSlot)
	if f.balanceSlotsUseCase != nil {
		balanceSlots = f.getBalanceSlots(&entityPool)
	}
	aevmPool, err := poolFactory(aevmpool.FactoryParams{
		EntityPool:   entityPool,
		ChainID:      f.config.ChainID,
		AEVMClient:   f.client,
		EthClient:    f.ethClient,
		StateRoot:    stateRoot,
		BalanceSlots: balanceSlots,
	})
	if err != nil {
		return nil, err
	}

	return aevmpoolwrapper.NewPoolWrapperAsAEVMPool(unimplementedPool, aevmPool, f.client), nil
}

func (f *PoolFactory) getBalanceSlots(pool *entity.Pool) map[common.Address]*types.ERC20BalanceSlot {
	balanceSlots := make(map[common.Address]*types.ERC20BalanceSlot)
	for _, token := range pool.Tokens {
		tokenAddr := common.HexToAddress(token.Address)
		bl, err := f.balanceSlotsUseCase.Get(context.Background(), tokenAddr, pool)
		if err != nil {
			continue
		}
		balanceSlots[tokenAddr] = bl
	}
	return balanceSlots
}

func (f *PoolFactory) NewSwapLimit(
	limits map[string]map[string]*big.Int,
	poolManagerExtraData usecasetypes.PoolManagerExtraData,
) map[string]poolpkg.SwapLimit {
	var limitMap = make(map[string]poolpkg.SwapLimit, len(limits))
	for dex, limit := range limits {
		limitMap[dex] = newSwapLimit(dex, limit, poolManagerExtraData)
	}
	return limitMap
}

func newSwapLimit(
	dex string,
	limit map[string]*big.Int,
	poolManagerExtraData usecasetypes.PoolManagerExtraData,
) poolpkg.SwapLimit {
	switch dex {
	case pooltypes.PoolTypes.Synthetix,
		pooltypes.PoolTypes.NativeV1,
		pooltypes.PoolTypes.Dexalot,
		pooltypes.PoolTypes.RingSwap,
		pooltypes.PoolTypes.LO1inch,
		pooltypes.PoolTypes.KyberPMM,
		pooltypes.PoolTypes.Pmm1,
		pooltypes.PoolTypes.Pmm2:
		return swaplimit.NewInventory(dex, limit)

	case pooltypes.PoolTypes.LimitOrder:
		return swaplimit.NewInventoryWithAllowedSenders(
			dex,
			limit,
			poolManagerExtraData.KyberLimitOrderAllowedSenders,
		)
	}

	return nil
}
