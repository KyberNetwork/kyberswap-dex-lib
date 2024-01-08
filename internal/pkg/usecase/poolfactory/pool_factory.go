package poolfactory

import (
	"context"
	"encoding/json"
	"math/big"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	balancerv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v1"
	balancerv2composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/composable-stable"
	balancerv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/stable"
	balancerv2weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/weighted"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	velocorev2cpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/cpmm"
	velocorev2wombatstable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/wombat-stable"
	woofiv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/algebrav1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/camelot"
	curveAave "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/aave"
	curveBase "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	curveCompound "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/compound"
	curveMeta "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/meta"
	curvePlainOracle "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/plain-oracle"
	curveTricrypto "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/tricrypto"
	curveTwo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/two"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dodo"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/elastic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/equalizer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fraxswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fulcrom"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/fxdx"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
	gmxglp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx-glp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap"
	kokonutcrypto "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kokonut-crypto"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido"
	lidosteth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/lido-steth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/madmex"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/makerpsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/mantisswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/maverickv1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/metavault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pancakev3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/platypus"
	polmatic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pol-matic"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/quickperps"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramsesv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/smardex"
	solidlyv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/solidly-v3"
	swapbasedperp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/swapbased-perp"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap/syncswapclassic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap/syncswapstable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/usdfi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocimeter"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velodrome"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velodromev2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/vooi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat/wombatlsd"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat/wombatmain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	liquiditybookv20aevm "github.com/KyberNetwork/router-service/internal/pkg/core/liquiditybookv20"
	liquiditybookv21aevm "github.com/KyberNetwork/router-service/internal/pkg/core/liquiditybookv21"
	routerentity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	ErrInitializePoolFailed    = errors.New("initialize pool failed")
	ErrBasePoolNotFound        = errors.New("base pool not found")
	ErrPoolTypeFactoryNotFound = errors.New("there is no factory for the pool type")
	ErrUnmarshalDataFailed     = errors.New("unmarshall data failed")
)

type PoolFactory struct {
	config              Config
	client              aevmclient.Client
	balanceSlotsUseCase *erc20balanceslot.Cache

	lock sync.Mutex
}

func NewPoolFactory(config Config, client aevmclient.Client, balanceSlotsUseCase *erc20balanceslot.Cache) *PoolFactory {
	return &PoolFactory{
		config:              config,
		client:              client,
		balanceSlotsUseCase: balanceSlotsUseCase,
	}
}

func (f *PoolFactory) ApplyConfig(config Config) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.config = config
}

func (f *PoolFactory) NewPools(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) []poolpkg.IPoolSimulator {
	span, _ := tracer.StartSpanFromContext(ctx, "poolFactory.NewPoolByAddress")
	defer span.End()

	curveBasePoolByAddress, curveBasePoolAddressSet := f.getCurveMetaBasePoolByAddress(pools)

	iPoolSimulators := make([]poolpkg.IPoolSimulator, 0, len(pools))
	for _, pool := range pools {
		if curveBasePoolAddressSet.Has(pool.Address) {
			iPoolSimulator, ok := curveBasePoolByAddress[pool.Address]
			if !ok {
				continue // NOTE: already warned before
			}

			iPoolSimulators = append(iPoolSimulators, iPoolSimulator.(poolpkg.IPoolSimulator)) // iPoolSimulator here is ICurveBasePool and surely is a poolpkg.IPoolSimulator
		} else if pool.Type == constant.PoolTypes.CurveMeta {
			iPool, err := f.newCurveMeta(*pool, curveBasePoolByAddress)
			if err != nil {
				logger.Debugf(err.Error())
				continue
			}

			iPoolSimulators = append(iPoolSimulators, iPool)
		} else {
			iPool, err := f.newPool(*pool, stateRoot)
			if err != nil {
				logger.Debug(err.Error())
				continue
			}

			iPoolSimulators = append(iPoolSimulators, iPool)
		}
	}

	return iPoolSimulators
}

func (f *PoolFactory) NewPoolByAddress(ctx context.Context, pools []*entity.Pool, stateRoot common.Hash) map[string]poolpkg.IPoolSimulator {
	span, _ := tracer.StartSpanFromContext(ctx, "poolFactory.NewPoolByAddress")
	defer span.End()

	curveBasePoolByAddress, curveBasePoolAddressSet := f.getCurveMetaBasePoolByAddress(pools)

	poolByAddress := make(map[string]poolpkg.IPoolSimulator, len(pools))
	for _, pool := range pools {
		if curveBasePoolAddressSet.Has(pool.Address) {
			IPoolSimulator, ok := curveBasePoolByAddress[pool.Address]
			if !ok {
				continue // NOTE: already warned before
			}

			poolByAddress[IPoolSimulator.GetInfo().Address] = IPoolSimulator.(poolpkg.IPoolSimulator) // IPoolSimulator here is ICurveBasePool and surely is a poolpkg.IPoolSimulator
		} else if pool.Type == constant.PoolTypes.CurveMeta {
			IPoolSimulator, err := f.newCurveMeta(*pool, curveBasePoolByAddress)
			if err != nil {
				logger.Debugf(err.Error())
				continue
			}

			poolByAddress[IPoolSimulator.GetAddress()] = IPoolSimulator
		} else {
			iPool, err := f.newPool(*pool, stateRoot)
			if err != nil {
				logger.Debugf(err.Error())
				continue
			}

			poolByAddress[iPool.GetAddress()] = iPool
		}
	}

	return poolByAddress
}

func (f *PoolFactory) getBalanceSlots(pool *entity.Pool) map[common.Address]*routerentity.ERC20BalanceSlot {
	balanceSlots := make(map[common.Address]*routerentity.ERC20BalanceSlot)
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

func (f *PoolFactory) getCurveMetaBasePoolByAddress(
	entityPools []*entity.Pool,
) (map[string]curveMeta.ICurveBasePool, sets.String) {
	basePoolByAddress := make(map[string]curveMeta.ICurveBasePool)
	basePoolAddresses := sets.NewString()

	for _, entityPool := range entityPools {
		switch entityPool.Type {
		case constant.PoolTypes.CurveBase:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveBase(*entityPool)
				if err != nil {
					logger.Warn(err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		case constant.PoolTypes.CurvePlainOracle:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurvePlainOracle(*entityPool)
				if err != nil {
					logger.Warn(err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		case constant.PoolTypes.CurveAave:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveAAVE(*entityPool)
				if err != nil {
					logger.Warn(err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		default:
			continue
		}

	}
	return basePoolByAddress, basePoolAddresses
}

func newSwapLimit(dex string, limit map[string]*big.Int) poolpkg.SwapLimit {
	switch dex {
	case constant.PoolTypes.KyberPMM:
		return kyberpmm.NewInventory(limit)
	case constant.PoolTypes.Synthetix:
		return synthetix.NewLimits(limit)
	}
	return nil
}

func (f *PoolFactory) NewSwapLimit(limits map[string]map[string]*big.Int) map[string]poolpkg.SwapLimit {
	var limitMap = make(map[string]poolpkg.SwapLimit, len(limits))
	for dex, limit := range limits {
		limitMap[dex] = newSwapLimit(dex, limit)
	}
	return limitMap
}

// newPool receives entity.Pool, based on its type to return matched factory method
// if there is no matched factory method, it returns ErrPoolTypeFactoryNotFound
func (f *PoolFactory) newPool(entityPool entity.Pool, stateRoot common.Hash) (poolpkg.IPoolSimulator, error) {
	switch entityPool.Type {
	case constant.PoolTypes.Uni, constant.PoolTypes.Firebird,
		constant.PoolTypes.Biswap, constant.PoolTypes.Polydex:
		return f.newUni(entityPool)
	case constant.PoolTypes.UniV3:
		return f.newUniV3(entityPool)
	case constant.PoolTypes.Saddle, constant.PoolTypes.Nerve,
		constant.PoolTypes.OneSwap, constant.PoolTypes.IronStable:
		return f.newSaddle(entityPool)
	case constant.PoolTypes.RamsesV2:
		return f.newRamsesV2(entityPool)
	case constant.PoolTypes.SolidlyV3:
		return f.newSolidlyV3(entityPool)
	case constant.PoolTypes.Dmm:
		return f.newDMM(entityPool)
	case constant.PoolTypes.Elastic:
		return f.newElastic(entityPool)
	case constant.PoolTypes.CurveAave:
		return f.newCurveAAVE(entityPool)
	case constant.PoolTypes.CurveCompound:
		return f.newCurveCompound(entityPool)
	case constant.PoolTypes.CurveTricrypto:
		return f.newCurveTricrypto(entityPool)
	case constant.PoolTypes.CurveTwo:
		return f.newCurveTwo(entityPool)
	case constant.PoolTypes.DodoClassical, constant.PoolTypes.DodoStable,
		constant.PoolTypes.DodoVendingMachine, constant.PoolTypes.DodoPrivate:
		return f.newDoDo(entityPool)
	case constant.PoolTypes.Velodrome, constant.PoolTypes.Ramses,
		constant.PoolTypes.MuteSwitch, constant.PoolTypes.Dystopia, constant.PoolTypes.Pearl:
		return f.newVelodrome(entityPool)
	case constant.PoolTypes.VelodromeV2:
		return f.newVelodromeV2(entityPool)
	case constant.PoolTypes.Velocimeter:
		return f.newVelocimeter(entityPool)
	case constant.PoolTypes.PlatypusBase, constant.PoolTypes.PlatypusPure, constant.PoolTypes.PlatypusAvax:
		return f.newPlatypus(entityPool)
	case constant.PoolTypes.WombatMain:
		return f.newWombatMain(entityPool)
	case constant.PoolTypes.WombatLsd:
		return f.newWombatLsd(entityPool)
	case constant.PoolTypes.GMX:
		return f.newGMX(entityPool)
	case constant.PoolTypes.GMXGLP:
		return f.newGmxGlp(entityPool)
	case constant.PoolTypes.MakerPSM:
		return f.newMakerPSm(entityPool)
	case constant.PoolTypes.Synthetix:
		return f.newSynthetix(entityPool)
	case constant.PoolTypes.MadMex:
		return f.newMadMex(entityPool)
	case constant.PoolTypes.Metavault:
		return f.newMetavault(entityPool)
	case constant.PoolTypes.Lido:
		return f.newLido(entityPool)
	case constant.PoolTypes.LidoStEth:
		return f.newLidoStEth(entityPool)
	case constant.PoolTypes.Fraxswap:
		return f.newFraxswap(entityPool)
	case constant.PoolTypes.Camelot:
		return f.newCamelot(entityPool)
	case constant.PoolTypes.LimitOrder:
		return f.newLimitOrder(entityPool)
	case constant.PoolTypes.SyncSwapClassic:
		return f.newSyncswapClassic(entityPool)
	case constant.PoolTypes.SyncSwapStable:
		return f.newSyncswapStable(entityPool)
	case constant.PoolTypes.PancakeV3:
		return f.newPancakeV3(entityPool)
	case constant.PoolTypes.MaverickV1:
		return f.newMaverickV1(entityPool)
	case constant.PoolTypes.AlgebraV1:
		return f.newAlgebraV1(entityPool)
	case constant.PoolTypes.KyberPMM:
		return f.newKyberPMM(entityPool)
	case constant.PoolTypes.IZiSwap:
		return f.newIZiSwap(entityPool)
	case constant.PoolTypes.WooFiV2:
		return f.newWooFiV2(entityPool)
	case constant.PoolTypes.Equalizer:
		return f.newEqualizer(entityPool)
	case constant.PoolTypes.SwapBasedPerp:
		return f.newSwapBasedPerp(entityPool)
	case constant.PoolTypes.USDFi:
		return f.newUSDFi(entityPool)
	case constant.PoolTypes.MantisSwap:
		return f.newMantisSwap(entityPool)
	case constant.PoolTypes.Vooi:
		return f.newVooi(entityPool)
	case constant.PoolTypes.PolMatic:
		return f.newPolMatic(entityPool)
	case constant.PoolTypes.KokonutCrypto:
		return f.newKokonutCrypto(entityPool)
	case constant.PoolTypes.LiquidityBookV21:
		if f.config.UseAEVM {
			return f.newLiquidityBookV21AEVM(entityPool, stateRoot)
		}
		return f.newLiquidityBookV21(entityPool)
	case constant.PoolTypes.LiquidityBookV20:
		if f.config.UseAEVM {
			return f.newLiquidityBookV20AEVM(entityPool, stateRoot)
		}
		return f.newLiquidityBookV20(entityPool)
	case constant.PoolTypes.Smardex:
		return f.newSmardex(entityPool)
	case constant.PoolTypes.Fxdx:
		return f.newFxdx(entityPool)
	case constant.PoolTypes.UniswapV2:
		return f.newUniswapV2(entityPool)
	case constant.PoolTypes.QuickPerps:
		return f.newQuickPerps(entityPool)
	case constant.PoolTypes.BalancerV1:
		return f.newBalancerV1(entityPool)
	case constant.PoolTypes.BalancerV2Weighted:
		return f.newBalancerV2Weighted(entityPool)
	case constant.PoolTypes.BalancerV2Stable:
		return f.newBalancerV2Stable(entityPool)
	case constant.PoolTypes.BalancerV2ComposableStable:
		return f.newBalancerV2ComposableStable(entityPool)
	case constant.PoolTypes.VelocoreV2CPMM:
		return f.newVelocoreV2CPMM(entityPool)
	case constant.PoolTypes.VelocoreV2WombatStable:
		return f.newVelocoreV2WombatStable(entityPool)
	case constant.PoolTypes.Fulcrom:
		return f.newFulcrom(entityPool)
	default:
		return nil, errors.Wrapf(
			ErrPoolTypeFactoryNotFound,
			"[PoolFactory.NewPoolSimulator] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

}

func (f *PoolFactory) newUni(entityPool entity.Pool) (*uniswap.PoolSimulator, error) {
	corePool, err := uniswap.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newUni] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newUniV3(entityPool entity.Pool) (*uniswapv3.PoolSimulator, error) {
	corePool, err := uniswapv3.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newUniV3] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSaddle(entityPool entity.Pool) (*saddle.PoolSimulator, error) {
	corePool, err := saddle.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSaddle] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newRamsesV2(entityPool entity.Pool) (*ramsesv2.PoolSimulator, error) {
	corePool, err := ramsesv2.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newRamsesV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSolidlyV3(entityPool entity.Pool) (*solidlyv3.PoolSimulator, error) {
	corePool, err := solidlyv3.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSolidlyV3] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDMM(entityPool entity.Pool) (*dmm.PoolSimulator, error) {
	corePool, err := dmm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newDMM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newElastic(entityPool entity.Pool) (*elastic.PoolSimulator, error) {
	corePool, err := elastic.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newElastic] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveBase(entityPool entity.Pool) (*curveBase.PoolBaseSimulator, error) {
	corePool, err := curveBase.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveBase] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurvePlainOracle(entityPool entity.Pool) (*curvePlainOracle.Pool, error) {
	corePool, err := curvePlainOracle.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurvePlainOracle] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveMeta(
	entityPool entity.Pool, curveBasePoolByAddress map[string]curveMeta.ICurveBasePool,
) (*curveMeta.Pool, error) {
	var staticExtra struct {
		BasePool string `json:"basePool"`
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, errors.Wrapf(
			ErrUnmarshalDataFailed,
			"[PoolFactory.newCurveMeta] pool: [%s] » basePool: [%s]",
			entityPool.Address,
			staticExtra.BasePool,
		)
	}

	basePool, ok := curveBasePoolByAddress[staticExtra.BasePool]
	if !ok {
		return nil, errors.Wrapf(
			ErrBasePoolNotFound,
			"[PoolFactory.newCurveMeta] pool: [%s] » basePool: [%s]",
			entityPool.Address,
			staticExtra.BasePool,
		)
	}

	curveMetaPool, err := curveMeta.NewPoolSimulator(entityPool, basePool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveMeta] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return curveMetaPool, nil
}

func (f *PoolFactory) newCurveAAVE(entityPool entity.Pool) (*curveAave.AavePool, error) {
	corePool, err := curveAave.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveAAVE] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveCompound(entityPool entity.Pool) (*curveCompound.CompoundPool, error) {
	corePool, err := curveCompound.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveCompound] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveTricrypto(entityPool entity.Pool) (*curveTricrypto.Pool, error) {
	corePool, err := curveTricrypto.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveTricrypto] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveTwo(entityPool entity.Pool) (*curveTwo.Pool, error) {
	corePool, err := curveTwo.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveTwo] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDoDo(entityPool entity.Pool) (*dodo.PoolSimulator, error) {
	corePool, err := dodo.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newDoDo] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelodrome(entityPool entity.Pool) (*velodrome.PoolSimulator, error) {
	corePool, err := velodrome.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelodrome] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelodromeV2(entityPool entity.Pool) (*velodromev2.PoolSimulator, error) {
	corePool, err := velodromev2.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelodromeV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelocimeter(entityPool entity.Pool) (*velocimeter.Pool, error) {
	corePool, err := velocimeter.NewPool(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelocimeter] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newPlatypus(entityPool entity.Pool) (*platypus.PoolSimulator, error) {
	corePool, err := platypus.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newPlatypus] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newWombatMain(entityPool entity.Pool) (*wombatmain.PoolSimulator, error) {
	corePool, err := wombatmain.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newWombatMain] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newWombatLsd(entityPool entity.Pool) (*wombatlsd.PoolSimulator, error) {
	corePool, err := wombatlsd.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newWombatLsd] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newGMX(entityPool entity.Pool) (*gmx.PoolSimulator, error) {
	corePool, err := gmx.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newGMX] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newGmxGlp(entityPool entity.Pool) (*gmxglp.PoolSimulator, error) {
	corePool, err := gmxglp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newGmxGlp] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMadMex(entityPool entity.Pool) (*madmex.PoolSimulator, error) {
	corePool, err := madmex.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newMadMex] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMetavault(entityPool entity.Pool) (*metavault.PoolSimulator, error) {
	corePool, err := metavault.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newMetavault] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMakerPSm(entityPool entity.Pool) (*makerpsm.PoolSimulator, error) {
	corePool, err := makerpsm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newMakerPSm] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSynthetix(entityPool entity.Pool) (*synthetix.PoolSimulator, error) {
	corePool, err := synthetix.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSynthetix] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newLido(entityPool entity.Pool) (*lido.PoolSimulator, error) {
	corePool, err := lido.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLido] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newLidoStEth(entityPool entity.Pool) (*lidosteth.PoolSimulator, error) {
	corePool, err := lidosteth.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLidoStEth] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newFraxswap(entityPool entity.Pool) (*fraxswap.PoolSimulator, error) {
	corePool, err := fraxswap.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newFraxswap] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}
	return corePool, nil

}
func (f *PoolFactory) newLimitOrder(entityPool entity.Pool) (*limitorder.PoolSimulator, error) {
	corePool, err := limitorder.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLimitOrder] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCamelot(entityPool entity.Pool) (*camelot.PoolSimulator, error) {
	corePool, err := camelot.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newCamelot] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSyncswapClassic(entityPool entity.Pool) (*syncswapclassic.PoolSimulator, error) {
	corePool, err := syncswapclassic.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSyncswapClassic] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSyncswapStable(entityPool entity.Pool) (*syncswapstable.PoolSimulator, error) {
	corePool, err := syncswapstable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSyncswapClassic] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newPancakeV3(entityPool entity.Pool) (*pancakev3.PoolSimulator, error) {
	corePool, err := pancakev3.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newPancakeV3] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMaverickV1(entityPool entity.Pool) (*maverickv1.Pool, error) {
	corePool, err := maverickv1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newMaverickV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}

func (f *PoolFactory) newAlgebraV1(entityPool entity.Pool) (*algebrav1.PoolSimulator, error) {
	defaultGas := DefaultGasAlgebra[valueobject.Exchange(entityPool.Exchange)]
	corePool, err := algebrav1.NewPoolSimulator(entityPool, defaultGas)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newAlgebraV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}

func (f *PoolFactory) newKyberPMM(entityPool entity.Pool) (*kyberpmm.PoolSimulator, error) {
	corePool, err := kyberpmm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newKyberPMM] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newIZiSwap(entityPool entity.Pool) (*iziswap.PoolSimulator, error) {
	corePool, err := iziswap.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newIZiSwap] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newWooFiV2(entityPool entity.Pool) (*woofiv2.PoolSimulator, error) {
	corePool, err := woofiv2.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newWooFiV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newEqualizer(entityPool entity.Pool) (*equalizer.PoolSimulator, error) {
	corePool, err := equalizer.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newEqualizer] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSwapBasedPerp(entityPool entity.Pool) (*swapbasedperp.PoolSimulator, error) {
	corePool, err := swapbasedperp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSwapBasedPerp] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newUSDFi(entityPool entity.Pool) (*usdfi.PoolSimulator, error) {
	corePool, err := usdfi.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newUSDFi] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMantisSwap(entityPool entity.Pool) (*mantisswap.PoolSimulator, error) {
	corePool, err := mantisswap.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newMantisSwap] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVooi(entityPool entity.Pool) (*vooi.PoolSimulator, error) {
	corePool, err := vooi.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.vooi] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newPolMatic(entityPool entity.Pool) (*polmatic.PoolSimulator, error) {
	corePool, err := polmatic.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.polmatic] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newKokonutCrypto(entityPool entity.Pool) (*kokonutcrypto.PoolSimulator, error) {
	corePool, err := kokonutcrypto.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.kokonutCrypto] pool: [%s] » type: [%s]",

			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newLiquidityBookV21(entityPool entity.Pool) (*liquiditybookv21.PoolSimulator, error) {
	corePool, err := liquiditybookv21.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLiquidityBookV21] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newLiquidityBookV20(entityPool entity.Pool) (*liquiditybookv20.PoolSimulator, error) {
	corePool, err := liquiditybookv20.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLiquidityBookV20] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newLiquidityBookV21AEVM(entityPool entity.Pool, stateRoot common.Hash) (*liquiditybookv21aevm.Pool, error) {
	if f.balanceSlotsUseCase == nil || f.client == nil {
		return nil, errors.New("AEVM is not initialized")
	}
	balanceSlots := f.getBalanceSlots(&entityPool)
	corePool, err := liquiditybookv21aevm.NewPoolAEVM(f.config.ChainID, entityPool, f.client, stateRoot, balanceSlots)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLiquidityBookV21AEVM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}
	return corePool, nil
}

func (f *PoolFactory) newLiquidityBookV20AEVM(entityPool entity.Pool, stateRoot common.Hash) (*liquiditybookv20aevm.Pool, error) {
	if f.balanceSlotsUseCase == nil || f.client == nil {
		return nil, errors.New("AEVM is not initialized")
	}
	balanceSlots := f.getBalanceSlots(&entityPool)
	corePool, err := liquiditybookv20aevm.NewPoolAEVM(f.config.ChainID, entityPool, f.client, stateRoot, balanceSlots)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newLiquidityBookV20AEVM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}
	return corePool, nil
}

func (f *PoolFactory) newSmardex(entityPool entity.Pool) (*smardex.PoolSimulator, error) {
	corePool, err := smardex.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newSmardex] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newFxdx(entityPool entity.Pool) (*fxdx.PoolSimulator, error) {
	corePool, err := fxdx.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newFxdx] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newUniswapV2(entityPool entity.Pool) (*uniswapv2.PoolSimulator, error) {
	corePool, err := uniswapv2.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newUniswapV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newQuickPerps(entityPool entity.Pool) (*quickperps.PoolSimulator, error) {
	corePool, err := quickperps.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newQuickperps] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerV1(entityPool entity.Pool) (*balancerv1.PoolSimulator, error) {
	corePool, err := balancerv1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newFulcrom(entityPool entity.Pool) (*fulcrom.PoolSimulator, error) {
	corePool, err := fulcrom.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newFulcrom] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerV2Weighted(entityPool entity.Pool) (*balancerv2weighted.PoolSimulator, error) {
	corePool, err := balancerv2weighted.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV2Weighted] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerV2Stable(entityPool entity.Pool) (*balancerv2stable.PoolSimulator, error) {
	corePool, err := balancerv2stable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV2Stable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerV2ComposableStable(entityPool entity.Pool) (*balancerv2composablestable.PoolSimulator, error) {
	corePool, err := balancerv2composablestable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV2ComposableStable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelocoreV2CPMM(entityPool entity.Pool) (*velocorev2cpmm.PoolSimulator, error) {
	corePool, err := velocorev2cpmm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelocoreV2CPMM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelocoreV2WombatStable(entityPool entity.Pool) (*velocorev2wombatstable.PoolSimulator, error) {
	corePool, err := velocorev2wombatstable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelocoreV2WombatStable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}
