package poolfactory

import (
	"context"
	"math/big"
	"strings"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	dexlibprivate "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source"
	aevmpool "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/aevm-pool"
	aevmpoolwrapper "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/aevm-pool/wrapper"
	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	curveStableMetaNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-meta-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	curveMeta "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/meta"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	clone "github.com/huandu/go-clone/generic"
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

func NewPoolFactory(config Config, ethClient bind.ContractBackend, client aevmclient.Client,
	balanceSlotsUseCase erc20balanceslot.ICache) *PoolFactory {
	return &PoolFactory{
		config:              config,
		ethClient:           ethClient,
		client:              client,
		balanceSlotsUseCase: balanceSlotsUseCase,
	}
}

func (f *PoolFactory) ApplyConfig(config Config) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.config = config
}

var (
	basePoolTypesSets = []mapset.Set[string]{ // ordered sets of base pools that must be created first
		mapset.NewThreadUnsafeSet(
			pooltypes.PoolTypes.CurveBase,
			pooltypes.PoolTypes.CurveStablePlain,
			pooltypes.PoolTypes.CurvePlainOracle,
			pooltypes.PoolTypes.CurveAave,
			pooltypes.PoolTypes.CurveStablePlain,
			pooltypes.PoolTypes.CurveStableNg,
		),
	}
)

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
			if !basePoolTypes.ContainsOne(pool.Type) {
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
	err = ErrPoolTypeFactoryNotFound

	poolFactory := poolpkg.Factory(entityPool.Type)
	if poolFactory != nil {
		pool, err = poolFactory(factoryParams)
		if err == nil {
			return pool, nil
		}
	}

	if f.config.UseAEVM && f.config.DexUseAEVM[entityPool.Type] && stateRoot != (common.Hash{}) {
		aevmPoolFactory := aevmpool.Factory(entityPool.Type)
		if aevmPoolFactory == nil {
			return nil, errors.WithMessagef(ErrPoolTypeFactoryNotFound, "%s aevm(%s/%s)",
				entityPool.Address, entityPool.Exchange, entityPool.Type)
		}
		return f.newAEVMPoolWrapper(entityPool, aevmPoolFactory, stateRoot)
	}

	return nil, errors.WithMessagef(err, "%s (%s/%s)",
		entityPool.Address, entityPool.Exchange, entityPool.Type)
}

func (f *PoolFactory) newAEVMPoolWrapper(entityPool entity.Pool, poolFactory aevmpool.FactoryFn,
	stateRoot common.Hash) (*aevmpoolwrapper.PoolWrapper, error) {
	unimplementedPool := dexlibprivate.NewUnimplementedPool(entityPool.Address, entityPool.Exchange, entityPool.Type)

	aevmPool, err := poolFactory(aevmpool.FactoryParams{
		EntityPool:   entityPool,
		ChainID:      f.config.ChainID,
		AEVMClient:   f.client,
		StateRoot:    stateRoot,
		BalanceSlots: f.getBalanceSlots(&entityPool),
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

func (f *PoolFactory) CloneCurveMetaForBasePools(
	_ context.Context,
	allPools map[string]poolpkg.IPoolSimulator,
	basePools map[string]poolpkg.IPoolSimulator,
) []poolpkg.IPoolSimulator {
	var cloned []poolpkg.IPoolSimulator

	for _, pool := range allPools {
		if pool.GetType() == pooltypes.PoolTypes.CurveMeta {
			metaPool, ok := pool.(*curveMeta.PoolSimulator)
			if !ok {
				continue
			}
			basePoolAddress := strings.ToLower(metaPool.BasePool.GetInfo().Address)

			if basePool, ok := basePools[basePoolAddress]; ok {
				if basePoolCorrect, ok := basePool.(curveMeta.ICurveBasePool); ok {
					newMetaPool := clone.Slowly(metaPool)
					newMetaPool.BasePool = basePoolCorrect
					cloned = append(cloned, newMetaPool)
				}
			}
		} else if pool.GetType() == pooltypes.PoolTypes.CurveStableMetaNg {
			metaPool, ok := pool.(*curveStableMetaNg.PoolSimulator)
			if !ok {
				continue
			}
			basePoolAddress := strings.ToLower(metaPool.GetBasePool().GetInfo().Address)

			if basePool, ok := basePools[basePoolAddress]; ok {
				if basePoolCorrect, ok := basePool.(curveStableMetaNg.ICurveBasePool); ok {
					newMetaPool := clone.Slowly(metaPool)
					newMetaPool.SetBasePool(basePoolCorrect)
					cloned = append(cloned, newMetaPool)
				}
			}
		}
	}

	return cloned
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
		pooltypes.PoolTypes.MxTrading,
		pooltypes.PoolTypes.LO1inch,
		pooltypes.PoolTypes.KyberPMM,
		pooltypes.PoolTypes.OneBit:
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
