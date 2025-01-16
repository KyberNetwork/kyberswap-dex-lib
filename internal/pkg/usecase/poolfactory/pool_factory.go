package poolfactory

import (
	"context"
	"math/big"
	"strings"
	"sync"

	aevmclient "github.com/KyberNetwork/aevm/client"
	dexlibprivate "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source"
	aevmpoolwrapper "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/aevm-pool/wrapper"
	ambientaevm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/ambient"
	maverickv2aevm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/maverick-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	algebraintegral "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral"
	algebrav1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient"
	balancerv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v1"
	balancerv2composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/composable-stable"
	balancerv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/stable"
	balancerv2weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/weighted"
	balancerv3stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/stable"
	balancerv3weighted "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/weighted"
	bancorv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bancor-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bedrock/unieth"
	beetsss "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/beets-ss"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clipper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	curveStableMetaNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-meta-ng"
	curveStableNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	curveTriCryptoNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/tricrypto-ng"
	curveTwoCryptoNg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng"
	daiusds "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dai-usds"
	deltaswapv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/deltaswap-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	dodoclassical "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/classical"
	dododpp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dpp"
	dododsp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dsp"
	dododvm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/dvm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ethena/susde"
	ethervista "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ether-vista"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/eeth"
	etherfivampire "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/vampire"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/weeth"
	fluiddext1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-t1"
	fluidvaultt1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/vault-t1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth"
	sfrxeth_convertor "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/sfrxeth-convertor"
	generic_simple_rate "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/generic-simple-rate"
	gyro2clp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/2clp"
	gyro3clp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/3clp"
	gyroeclp "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/eclp"
	hashflowv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/hashflow-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/integral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kelp/rseth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/litepsm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mantle/meth"
	mkrsky "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mkr-sky"
	mxtrading "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mx-trading"
	nativev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native-v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nomiswap/nomiswapstable"
	ondo_usdy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ondo-usdy"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/primeeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/puffer/pufeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/renzo/ezeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ringswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/rocketpool/reth"
	solidlyv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/solidly-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/staderethx"
	swaapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/rsweth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swell/sweth"
	syncswapv2aqua "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/aqua"
	syncswapv2classic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/classic"
	syncswapv2stable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/syncswapv2/stable"
	uniswapv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v1"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/usd0pp"
	velocorev2cpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/cpmm"
	velocorev2wombatstable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/wombat-stable"
	velodrome "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v1"
	velodromev2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v2"
	virtualfun "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/virtual-fun"
	woofiv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v2"
	woofiv21 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/woofi-v21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/camelot"
	curveAave "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/aave"
	curveBase "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	curveCompound "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/compound"
	curveMeta "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/meta"
	curvePlainOracle "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/plain-oracle"
	curveTricrypto "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/tricrypto"
	curveTwo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/two"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/dmm"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/nuriv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pancakev3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/platypus"
	polmatic "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pol-matic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/quickperps"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramsesv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/saddle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/slipstream"
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
	velodromelegacy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velodrome"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/vooi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat/wombatlsd"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat/wombatmain"
	zkera "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/zkera-finance"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	clone "github.com/huandu/go-clone/generic"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	usecasetypes "github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
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
	balanceSlotsUseCase erc20balanceslot.ICache

	lock sync.Mutex
}

func NewPoolFactory(config Config, client aevmclient.Client, balanceSlotsUseCase erc20balanceslot.ICache) *PoolFactory {
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

	curveBasePoolByAddress, curveBasePoolAddressSet := f.getCurveMetaBasePoolByAddress(ctx, pools)
	// for now the NG version of meta only support plain/stable-ng as base pool
	curveBaseNGPoolByAddress, curveBaseNGPoolAddressSet := f.getCurveMetaBaseNGPoolByAddress(ctx, pools)

	iPoolSimulators := make([]poolpkg.IPoolSimulator, 0, len(pools))
	for _, pool := range pools {
		if curveBasePoolAddressSet.Has(pool.Address) {
			iPoolSimulator, ok := curveBasePoolByAddress[pool.Address]
			if !ok {
				continue // NOTE: already warned before
			}

			iPoolSimulators = append(iPoolSimulators, iPoolSimulator.(poolpkg.IPoolSimulator)) // iPoolSimulator here is ICurveBasePool and surely is a poolpkg.IPoolSimulator
		} else if curveBaseNGPoolAddressSet.Has(pool.Address) {
			iPoolSimulator, ok := curveBaseNGPoolByAddress[pool.Address]
			if !ok {
				continue
			}

			iPoolSimulators = append(iPoolSimulators, iPoolSimulator.(poolpkg.IPoolSimulator))
		} else if pool.Type == pooltypes.PoolTypes.CurveMeta {
			iPool, err := f.newCurveMeta(*pool, curveBasePoolByAddress)
			if err != nil {
				logger.Debugf(ctx, "[poolFactory.NewPools] CurveMeta %s", err.Error())
				continue
			}

			iPoolSimulators = append(iPoolSimulators, iPool)
		} else if pool.Type == pooltypes.PoolTypes.CurveStableMetaNg {
			iPool, err := f.newCurveMetaNG(*pool, curveBaseNGPoolByAddress)
			if err != nil {
				logger.Debugf(ctx, "[poolFactory.NewPools] CurveStableMetaNg %s", err.Error())
				continue
			}

			iPoolSimulators = append(iPoolSimulators, iPool)
		} else {
			iPool, err := f.newPool(*pool, stateRoot)
			if err != nil {
				logger.Debugf(ctx, "[poolFactory.NewPools] others %s", err.Error())
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

	curveBasePoolByAddress, curveBasePoolAddressSet := f.getCurveMetaBasePoolByAddress(ctx, pools)
	curveBaseNGPoolByAddress, curveBaseNGPoolAddressSet := f.getCurveMetaBaseNGPoolByAddress(ctx, pools)

	poolByAddress := make(map[string]poolpkg.IPoolSimulator, len(pools))
	for _, pool := range pools {
		if curveBasePoolAddressSet.Has(pool.Address) {
			IPoolSimulator, ok := curveBasePoolByAddress[pool.Address]
			if !ok {
				continue // NOTE: already warned before
			}

			poolByAddress[IPoolSimulator.GetInfo().Address] = IPoolSimulator.(poolpkg.IPoolSimulator) // IPoolSimulator here is ICurveBasePool and surely is a poolpkg.IPoolSimulator
		} else if curveBaseNGPoolAddressSet.Has(pool.Address) {
			IPoolSimulator, ok := curveBaseNGPoolByAddress[pool.Address]
			if !ok {
				continue
			}

			poolByAddress[IPoolSimulator.GetInfo().Address] = IPoolSimulator.(poolpkg.IPoolSimulator)
		} else if pool.Type == pooltypes.PoolTypes.CurveMeta {
			IPoolSimulator, err := f.newCurveMeta(*pool, curveBasePoolByAddress)
			if err != nil {
				logger.Debugf(ctx, "[poolFactory.NewPoolByAddress] CurveStableMetaNg %s", err.Error())
				continue
			}

			poolByAddress[IPoolSimulator.GetAddress()] = IPoolSimulator
		} else if pool.Type == pooltypes.PoolTypes.CurveStableMetaNg {
			IPoolSimulator, err := f.newCurveMetaNG(*pool, curveBaseNGPoolByAddress)
			if err != nil {
				logger.Debugf(ctx, "[poolFactory.NewPoolByAddress] CurveStableMetaNg %s", err.Error())
				continue
			}

			poolByAddress[IPoolSimulator.GetAddress()] = IPoolSimulator
		} else {
			iPool, err := f.newPool(*pool, stateRoot)
			if err != nil {
				logger.Debugf(ctx, "[poolFactory.NewPoolByAddress] others %s", err.Error())
				continue
			}

			poolByAddress[iPool.GetAddress()] = iPool
		}
	}
	return poolByAddress
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
	ctx context.Context,
	allPools map[string]pool.IPoolSimulator,
	basePools map[string]pool.IPoolSimulator,
) []pool.IPoolSimulator {
	var cloned []pool.IPoolSimulator

	for _, pool := range allPools {
		if pool.GetType() == pooltypes.PoolTypes.CurveMeta {
			metaPool, ok := pool.(*curveMeta.Pool)
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

func (f *PoolFactory) getCurveMetaBasePoolByAddress(
	ctx context.Context,
	entityPools []*entity.Pool,
) (map[string]curveMeta.ICurveBasePool, sets.String) {
	basePoolByAddress := make(map[string]curveMeta.ICurveBasePool)
	basePoolAddresses := sets.NewString()

	for _, entityPool := range entityPools {
		switch entityPool.Type {
		case pooltypes.PoolTypes.CurveBase:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveBase(*entityPool)
				if err != nil {
					logger.Debugf(ctx, "[getCurveMetaBasePoolByAddress] CurveBase %s", err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		case pooltypes.PoolTypes.CurveStablePlain:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveStablePlain(*entityPool)
				if err != nil {
					logger.Debugf(ctx, "[getCurveMetaBasePoolByAddress] CurveStablePlain %s", err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		case pooltypes.PoolTypes.CurvePlainOracle:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurvePlainOracle(*entityPool)
				if err != nil {
					logger.Debugf(ctx, "[getCurveMetaBasePoolByAddress] CurvePlainOracle %s", err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		case pooltypes.PoolTypes.CurveAave:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveAAVE(*entityPool)
				if err != nil {
					logger.Debugf(ctx, "[getCurveMetaBasePoolByAddress] CurveAave %s", err.Error())
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

func (f *PoolFactory) getCurveMetaBaseNGPoolByAddress(
	ctx context.Context,
	entityPools []*entity.Pool,
) (map[string]curveStableMetaNg.ICurveBasePool, sets.String) {
	basePoolByAddress := make(map[string]curveStableMetaNg.ICurveBasePool)
	basePoolAddresses := sets.NewString()

	for _, entityPool := range entityPools {
		switch entityPool.Type {
		case pooltypes.PoolTypes.CurveStablePlain:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveStablePlain(*entityPool)
				if err != nil {
					logger.Debugf(ctx, "[getCurveMetaBaseNGPoolByAddress] CurveStablePlain %s", err.Error())
					continue
				}
				basePoolByAddress[basePool.GetAddress()] = basePool
			}
		case pooltypes.PoolTypes.CurveStableNg:
			{
				basePoolAddresses.Insert(entityPool.Address)
				basePool, err := f.newCurveStableNg(*entityPool)
				if err != nil {
					logger.Debugf(ctx, "[getCurveMetaBaseNGPoolByAddress] CurveStableNg %s", err.Error())
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
		pooltypes.PoolTypes.LO1inch:
		return swaplimit.NewInventory(dex, limit)

	case pooltypes.PoolTypes.KyberPMM:
		return swaplimit.NewSwappedInventory(dex, limit)

	case pooltypes.PoolTypes.LimitOrder:
		return swaplimit.NewInventoryWithAllowedSenders(
			dex,
			limit,
			poolManagerExtraData.KyberLimitOrderAllowedSenders,
		)
	}

	return nil
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

// getAEVMDexHandler gets AEVM dex handler based on the config. If the stateRoot is empty, it will return an error.
func (f *PoolFactory) getAEVMDexHandler(
	poolType string,
	entityPool entity.Pool,
	stateRoot common.Hash,
	newAEVMFunc func(entityPool entity.Pool, stateRoot common.Hash) (*aevmpoolwrapper.PoolWrapper, error),
	newGoSimulatorFunc func(entityPool entity.Pool) (poolpkg.IPoolSimulator, error),
) (poolpkg.IPoolSimulator, error) {
	if f.config.UseAEVM && f.config.DexUseAEVM[poolType] && stateRoot != (common.Hash{}) {
		return newAEVMFunc(entityPool, stateRoot)
	}

	if newGoSimulatorFunc != nil {
		return newGoSimulatorFunc(entityPool)
	}

	return nil, errors.WithMessagef(
		ErrPoolTypeFactoryNotFound,
		"[PoolFactory.getAEVMDexHandler] pool: [%s] » type: [%s]",
		entityPool.Address,
		entityPool.Type,
	)
}

// newPool receives entity.Pool, based on its type to return matched factory method
// if there is no matched factory method, it returns ErrPoolTypeFactoryNotFound
func (f *PoolFactory) newPool(entityPool entity.Pool, stateRoot common.Hash) (poolpkg.IPoolSimulator, error) {
	switch entityPool.Type {
	case pooltypes.PoolTypes.Uni, pooltypes.PoolTypes.Firebird,
		pooltypes.PoolTypes.Biswap, pooltypes.PoolTypes.Polydex:
		return f.newUni(entityPool)
	case pooltypes.PoolTypes.UniswapV3:
		return f.newUniV3(entityPool)
	case pooltypes.PoolTypes.Saddle, pooltypes.PoolTypes.Nerve,
		pooltypes.PoolTypes.OneSwap, pooltypes.PoolTypes.IronStable:
		return f.newSaddle(entityPool)
	case pooltypes.PoolTypes.RamsesV2:
		return f.newRamsesV2(entityPool)
	case pooltypes.PoolTypes.SolidlyV2:
		return f.newSolidlyV2(entityPool)
	case pooltypes.PoolTypes.SolidlyV3:
		return f.newSolidlyV3(entityPool)
	case pooltypes.PoolTypes.Dmm:
		return f.newDMM(entityPool)
	case pooltypes.PoolTypes.Elastic:
		return f.newElastic(entityPool)
	case pooltypes.PoolTypes.CurveAave:
		return f.newCurveAAVE(entityPool)
	case pooltypes.PoolTypes.CurveCompound:
		return f.newCurveCompound(entityPool)
	case pooltypes.PoolTypes.CurveTricrypto:
		return f.newCurveTricrypto(entityPool)
	case pooltypes.PoolTypes.CurveTwo:
		return f.newCurveTwo(entityPool)
	case pooltypes.PoolTypes.CurveStableNg:
		return f.newCurveStableNg(entityPool)
	case pooltypes.PoolTypes.CurveTriCryptoNg:
		return f.newCurveTriCryptoNg(entityPool)
	case pooltypes.PoolTypes.CurveTwoCryptoNg:
		return f.newCurveTwoCryptoNg(entityPool)
	case pooltypes.PoolTypes.DodoClassical:
		return f.newDoDoClassical(entityPool)
	case pooltypes.PoolTypes.DodoPrivatePool:
		return f.newDoDoPrivatePool(entityPool)
	case pooltypes.PoolTypes.DodoStablePool:
		return f.newDoDoStablePool(entityPool)
	case pooltypes.PoolTypes.DodoVendingMachine:
		return f.newDoDoVendingMachine(entityPool)
	case pooltypes.PoolTypes.Velodrome:
		return f.newVelodrome(entityPool)
	case pooltypes.PoolTypes.Ramses, pooltypes.PoolTypes.MuteSwitch, pooltypes.PoolTypes.Dystopia, pooltypes.PoolTypes.Pearl:
		return f.newVelodromeLegacy(entityPool)
	case pooltypes.PoolTypes.VelodromeV2, pooltypes.PoolTypes.SwapXV2:
		return f.newVelodromeV2(entityPool)
	case pooltypes.PoolTypes.Velocimeter:
		return f.newVelocimeter(entityPool)
	case pooltypes.PoolTypes.PlatypusBase, pooltypes.PoolTypes.PlatypusPure, pooltypes.PoolTypes.PlatypusAvax:
		return f.newPlatypus(entityPool)
	case pooltypes.PoolTypes.WombatMain:
		return f.newWombatMain(entityPool)
	case pooltypes.PoolTypes.WombatLsd:
		return f.newWombatLsd(entityPool)
	case pooltypes.PoolTypes.GMX:
		return f.newGMX(entityPool)
	case pooltypes.PoolTypes.GMXGLP:
		return f.newGmxGlp(entityPool)
	case pooltypes.PoolTypes.MakerPSM:
		return f.newMakerPSm(entityPool)
	case pooltypes.PoolTypes.Synthetix:
		return f.newSynthetix(entityPool)
	case pooltypes.PoolTypes.MadMex:
		return f.newMadMex(entityPool)
	case pooltypes.PoolTypes.Metavault:
		return f.newMetavault(entityPool)
	case pooltypes.PoolTypes.Lido:
		return f.newLido(entityPool)
	case pooltypes.PoolTypes.LidoStEth:
		return f.newLidoStEth(entityPool)
	case pooltypes.PoolTypes.Fraxswap:
		return f.newFraxswap(entityPool)
	case pooltypes.PoolTypes.Camelot:
		return f.newCamelot(entityPool)
	case pooltypes.PoolTypes.LimitOrder:
		return f.newLimitOrder(entityPool)
	case pooltypes.PoolTypes.SyncSwapClassic:
		return f.newSyncswapClassic(entityPool)
	case pooltypes.PoolTypes.SyncSwapStable:
		return f.newSyncswapStable(entityPool)
	case pooltypes.PoolTypes.SyncSwapV2Classic:
		return f.newSyncswapV2Classic(entityPool)
	case pooltypes.PoolTypes.SyncSwapV2Stable:
		return f.newSyncswapV2Stable(entityPool)
	case pooltypes.PoolTypes.SyncSwapV2Aqua:
		return f.newSyncswapV2Aqua(entityPool)
	case pooltypes.PoolTypes.PancakeV3:
		return f.newPancakeV3(entityPool)
	case pooltypes.PoolTypes.MaverickV1:
		return f.newMaverickV1(entityPool)
	case pooltypes.PoolTypes.AlgebraV1:
		return f.newAlgebraV1(entityPool)
	case pooltypes.PoolTypes.KyberPMM:
		return f.newKyberPMM(entityPool)
	case pooltypes.PoolTypes.IZiSwap:
		return f.newIZiSwap(entityPool)
	case pooltypes.PoolTypes.WooFiV2:
		return f.newWooFiV2(entityPool)
	case pooltypes.PoolTypes.WooFiV21:
		return f.newWooFiV21(entityPool)
	case pooltypes.PoolTypes.Equalizer:
		return f.newEqualizer(entityPool)
	case pooltypes.PoolTypes.SwapBasedPerp:
		return f.newSwapBasedPerp(entityPool)
	case pooltypes.PoolTypes.USDFi:
		return f.newUSDFi(entityPool)
	case pooltypes.PoolTypes.MantisSwap:
		return f.newMantisSwap(entityPool)
	case pooltypes.PoolTypes.Vooi:
		return f.newVooi(entityPool)
	case pooltypes.PoolTypes.PolMatic:
		return f.newPolMatic(entityPool)
	case pooltypes.PoolTypes.KokonutCrypto:
		return f.newKokonutCrypto(entityPool)
	case pooltypes.PoolTypes.LiquidityBookV21:
		return f.newLiquidityBookV21(entityPool)
	case pooltypes.PoolTypes.LiquidityBookV20:
		return f.newLiquidityBookV20(entityPool)
	case pooltypes.PoolTypes.Smardex:
		return f.newSmardex(entityPool)
	case pooltypes.PoolTypes.Integral:
		return f.newIntegral(entityPool)
	case pooltypes.PoolTypes.Fxdx:
		return f.newFxdx(entityPool)
	case pooltypes.PoolTypes.UniswapV1:
		return f.newUniswapV1(entityPool)
	case pooltypes.PoolTypes.UniswapV2:
		return f.newUniswapV2(entityPool)
	case pooltypes.PoolTypes.QuickPerps:
		return f.newQuickPerps(entityPool)
	case pooltypes.PoolTypes.BalancerV1:
		return f.newBalancerV1(entityPool)
	case pooltypes.PoolTypes.BalancerV2Weighted:
		return f.newBalancerV2Weighted(entityPool)
	case pooltypes.PoolTypes.BalancerV2Stable:
		return f.newBalancerV2Stable(entityPool)
	case pooltypes.PoolTypes.BalancerV2ComposableStable:
		return f.newBalancerV2ComposableStable(entityPool)
	case pooltypes.PoolTypes.BalancerV3Weighted:
		return f.newBalancerV3Weighted(entityPool)
	case pooltypes.PoolTypes.BalancerV3Stable:
		return f.newBalancerV3Stable(entityPool)
	case pooltypes.PoolTypes.VelocoreV2CPMM:
		return f.newVelocoreV2CPMM(entityPool)
	case pooltypes.PoolTypes.VelocoreV2WombatStable:
		return f.newVelocoreV2WombatStable(entityPool)
	case pooltypes.PoolTypes.Fulcrom:
		return f.newFulcrom(entityPool)
	case pooltypes.PoolTypes.Gyroscope2CLP:
		return f.newGyroscope2CLP(entityPool)
	case pooltypes.PoolTypes.Gyroscope3CLP:
		return f.newGyroscope3CLP(entityPool)
	case pooltypes.PoolTypes.GyroscopeECLP:
		return f.newGyroscopeECLP(entityPool)
	case pooltypes.PoolTypes.ZkEraFinance:
		return f.newZkEraFinance(entityPool)
	case pooltypes.PoolTypes.SwaapV2:
		return f.newSwaapV2(entityPool)
	case pooltypes.PoolTypes.BancorV3:
		return f.newBancorV3(entityPool)
	case pooltypes.PoolTypes.EtherfiEETH:
		return f.newEtherfiEETH(entityPool)
	case pooltypes.PoolTypes.EtherfiWEETH:
		return f.newEtherfiWEETH(entityPool)
	case pooltypes.PoolTypes.KelpRSETH:
		return f.newKelpRSETH(entityPool)
	case pooltypes.PoolTypes.RocketPoolRETH:
		return f.newRocketPoolRETH(entityPool)
	case pooltypes.PoolTypes.EthenaSusde:
		return f.newEthenaSusde(entityPool)
	case pooltypes.PoolTypes.MakerSavingsDai:
		return f.newMakerSavingsDai(entityPool)
	case pooltypes.PoolTypes.HashflowV3:
		return f.newHashflowV3(entityPool)
	case pooltypes.PoolTypes.NativeV1:
		return f.newNativeV1(entityPool)
	case pooltypes.PoolTypes.Bebop:
		return f.newBebop(entityPool)
	case pooltypes.PoolTypes.Dexalot:
		return f.newDexalot(entityPool)
	case pooltypes.PoolTypes.NomiSwapStable:
		return f.newNomiswapStable(entityPool)
	case pooltypes.PoolTypes.RenzoEZETH:
		return f.newRenzoEzETH(entityPool)
	case pooltypes.PoolTypes.BedrockUniETH:
		return f.newBetrockUniETH(entityPool)
	case pooltypes.PoolTypes.PufferPufETH:
		return f.newPufferPufETH(entityPool)
	case pooltypes.PoolTypes.SwellRSWETH:
		return f.newSwellRSWETH(entityPool)
	case pooltypes.PoolTypes.SwellSWETH:
		return f.newSwellSWETH(entityPool)
	case pooltypes.PoolTypes.Slipstream:
		return f.newSlipstream(entityPool)
	case pooltypes.PoolTypes.NuriV2:
		return f.newNuriV2(entityPool)
	case ambient.DexTypeAmbient:
		return f.getAEVMDexHandler(ambient.DexTypeAmbient, entityPool, stateRoot, f.newAmbientAEVM, nil)
	case pooltypes.PoolTypes.EtherVista:
		return f.newEtherVista(entityPool)
	case pooltypes.PoolTypes.MaverickV2:
		return f.getAEVMDexHandler(pooltypes.PoolTypes.MaverickV2, entityPool, stateRoot, f.newMaverickV2AEVM, nil)
	case pooltypes.PoolTypes.LitePSM:
		return f.newLitePSM(entityPool)
	case pooltypes.PoolTypes.MkrSky:
		return f.newMkrSky(entityPool)
	case pooltypes.PoolTypes.DaiUsds:
		return f.newDaiUsds(entityPool)
	case pooltypes.PoolTypes.FluidVaultT1:
		return f.newFluidVaultT1(entityPool)
	case pooltypes.PoolTypes.FluidDexT1:
		return f.newFluidDexT1(entityPool)
	case pooltypes.PoolTypes.Usd0PP:
		return f.newUsd0PP(entityPool)
	case pooltypes.PoolTypes.RingSwap:
		return f.newRingSwap(entityPool)
	case pooltypes.PoolTypes.PrimeETH:
		return f.newPrimeETH(entityPool)
	case pooltypes.PoolTypes.StaderETHx:
		return f.newStaderETHx(entityPool)
	case pooltypes.PoolTypes.GenericSimpleRate:
		return f.newGenericSimpleRate(entityPool)
	case pooltypes.PoolTypes.MantleETH:
		return f.newMantleETH(entityPool)
	case pooltypes.PoolTypes.OndoUSDY:
		return f.newOndoUSDY(entityPool)
	case pooltypes.PoolTypes.Clipper:
		return f.newClipper(entityPool)
	case pooltypes.PoolTypes.DeltaSwapV1:
		return f.newDeltaSwapV1(entityPool)
	case pooltypes.PoolTypes.SfrxETH:
		return f.newSfrxETH(entityPool)
	case pooltypes.PoolTypes.SfrxETHConvertor:
		return f.newSfrxETHConvertor(entityPool)
	case pooltypes.PoolTypes.EtherfiVampire:
		return f.newEtherfiVampire(entityPool)
	case pooltypes.PoolTypes.AlgebraIntegral:
		return f.newAlgebraIntegral(entityPool)
	case pooltypes.PoolTypes.MxTrading:
		return f.newMxTrading(entityPool)
	case pooltypes.PoolTypes.LO1inch:
		return f.newLO1inch(entityPool)
	case pooltypes.PoolTypes.VirtualFun:
		return f.newVirtualFun(entityPool)
	case pooltypes.PoolTypes.BeetsSS:
		return f.newBeetsSS(entityPool)
	default:
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newRamsesV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSolidlyV2(entityPool entity.Pool) (*solidlyv2.PoolSimulator, error) {
	corePool, err := solidlyv2.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSolidlyV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSolidlyV3(entityPool entity.Pool) (*solidlyv3.PoolSimulator, error) {
	corePool, err := solidlyv3.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveBase] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveStablePlain(entityPool entity.Pool) (*plain.PoolSimulator, error) {
	pool, err := plain.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveStablePlain] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return pool, nil
}

func (f *PoolFactory) newCurveStableNg(entityPool entity.Pool) (*curveStableNg.PoolSimulator, error) {
	pool, err := curveStableNg.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			err,
			"[PoolFactory.newCurveStableNg] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return pool, nil
}

func (f *PoolFactory) newCurveTriCryptoNg(entityPool entity.Pool) (*curveTriCryptoNg.PoolSimulator, error) {
	pool, err := curveTriCryptoNg.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			err,
			"[PoolFactory.newCurveTriCryptoNg] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return pool, nil
}

func (f *PoolFactory) newCurveTwoCryptoNg(entityPool entity.Pool) (*curveTwoCryptoNg.PoolSimulator, error) {
	pool, err := curveTwoCryptoNg.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			err,
			"[PoolFactory.newCurveTwoCryptoNg] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return pool, nil
}

func (f *PoolFactory) newCurvePlainOracle(entityPool entity.Pool) (*curvePlainOracle.Pool, error) {
	corePool, err := curvePlainOracle.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrUnmarshalDataFailed,
			"[PoolFactory.newCurveMeta] pool: [%s] » basePool: [%s]",
			entityPool.Address,
			staticExtra.BasePool,
		)
	}

	basePool, ok := curveBasePoolByAddress[staticExtra.BasePool]
	if !ok {
		return nil, errors.WithMessagef(
			ErrBasePoolNotFound,
			"[PoolFactory.newCurveMeta] pool: [%s] » basePool: [%s]",
			entityPool.Address,
			staticExtra.BasePool,
		)
	}

	curveMetaPool, err := curveMeta.NewPoolSimulator(entityPool, basePool)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newCurveTwo] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newCurveMetaNG(
	entityPool entity.Pool, curveBasePoolByAddress map[string]curveStableMetaNg.ICurveBasePool,
) (*curveStableMetaNg.PoolSimulator, error) {
	var staticExtra struct {
		BasePool string `json:"basePool"`
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, errors.WithMessagef(
			err,
			"[PoolFactory.newCurveMetaNG] pool: [%s] » basePool: [%s]",
			entityPool.Address,
			staticExtra.BasePool,
		)
	}

	basePool, ok := curveBasePoolByAddress[staticExtra.BasePool]
	if !ok {
		return nil, errors.WithMessagef(
			ErrBasePoolNotFound,
			"[PoolFactory.newCurveMetaNG] pool: [%s] » basePool: [%s]",
			entityPool.Address,
			staticExtra.BasePool,
		)
	}

	curveMetaPool, err := curveStableMetaNg.NewPoolSimulator(entityPool, basePool)
	if err != nil {
		return nil, errors.WithMessagef(
			err,
			"[PoolFactory.newCurveMeta] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return curveMetaPool, nil
}

func (f *PoolFactory) newDoDoClassical(entityPool entity.Pool) (*dodoclassical.PoolSimulator, error) {
	corePool, err := dodoclassical.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDoDoClassical] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDoDoPrivatePool(entityPool entity.Pool) (*dododpp.PoolSimulator, error) {
	corePool, err := dododpp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDoDoPrivatePool] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDoDoStablePool(entityPool entity.Pool) (*dododsp.PoolSimulator, error) {
	corePool, err := dododsp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDoDoStablePool] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDoDoVendingMachine(entityPool entity.Pool) (*dododvm.PoolSimulator, error) {
	corePool, err := dododvm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDoDoVendingMachine] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelodrome(entityPool entity.Pool) (*velodrome.PoolSimulator, error) {
	corePool, err := velodrome.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelodrome] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelodromeLegacy(entityPool entity.Pool) (*velodromelegacy.PoolSimulator, error) {
	corePool, err := velodromelegacy.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelodromeLegacy] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelodromeV2(entityPool entity.Pool) (*velodromev2.PoolSimulator, error) {
	corePool, err := velodromev2.NewPoolSimulator(entityPool)

	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSyncswapClassic] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSyncswapV2Classic(entityPool entity.Pool) (*syncswapv2classic.PoolSimulator, error) {
	corePool, err := syncswapv2classic.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSyncswapv2Classic] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSyncswapV2Stable(entityPool entity.Pool) (*syncswapv2stable.PoolSimulator, error) {
	corePool, err := syncswapv2stable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSyncswapV2Stable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSyncswapV2Aqua(entityPool entity.Pool) (*syncswapv2aqua.PoolSimulator, error) {
	corePool, err := syncswapv2aqua.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSyncswapV2Aqua] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newPancakeV3(entityPool entity.Pool) (*pancakev3.PoolSimulator, error) {
	corePool, err := pancakev3.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newMaverickV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}

func (f *PoolFactory) newAlgebraV1(entityPool entity.Pool) (*algebrav1.PoolSimulator, error) {
	defaultGas := DefaultGasAlgebraV1[valueobject.Exchange(entityPool.Exchange)]
	corePool, err := algebrav1.NewPoolSimulator(entityPool, defaultGas)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newAlgebraV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}

func (f *PoolFactory) newAlgebraIntegral(entityPool entity.Pool) (*algebraintegral.PoolSimulator, error) {
	defaultGas := DefaultGasAlgebraIntegral[valueobject.Exchange(entityPool.Exchange)]
	corePool, err := algebraintegral.NewPoolSimulator(entityPool, defaultGas)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newAlgebraIntegral] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}

func (f *PoolFactory) newKyberPMM(entityPool entity.Pool) (*kyberpmm.PoolSimulator, error) {
	corePool, err := kyberpmm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newWooFiV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newWooFiV21(entityPool entity.Pool) (*woofiv21.PoolSimulator, error) {
	corePool, err := woofiv21.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newWooFiV21] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newEqualizer(entityPool entity.Pool) (*equalizer.PoolSimulator, error) {
	corePool, err := equalizer.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newLiquidityBookV20] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSmardex(entityPool entity.Pool) (*smardex.PoolSimulator, error) {
	corePool, err := smardex.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSmardex] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newIntegral(entityPool entity.Pool) (*integral.PoolSimulator, error) {
	corePool, err := integral.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newIntegral] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newFxdx(entityPool entity.Pool) (*fxdx.PoolSimulator, error) {
	corePool, err := fxdx.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newFxdx] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newUniswapV1(entityPool entity.Pool) (*uniswapv1.PoolSimulator, error) {
	corePool, err := uniswapv1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newUniswapV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newUniswapV2(entityPool entity.Pool) (*uniswapv2.PoolSimulator, error) {
	corePool, err := uniswapv2.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV2ComposableStable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerV3Weighted(entityPool entity.Pool) (*balancerv3weighted.PoolSimulator, error) {
	corePool, err := balancerv3weighted.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV3Weighted] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerV3Stable(entityPool entity.Pool) (*balancerv3stable.PoolSimulator, error) {
	corePool, err := balancerv3stable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerV3Stable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVelocoreV2CPMM(entityPool entity.Pool) (*velocorev2cpmm.PoolSimulator, error) {
	corePool, err := velocorev2cpmm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
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
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newVelocoreV2WombatStable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newGyroscope2CLP(entityPool entity.Pool) (*gyro2clp.PoolSimulator, error) {
	corePool, err := gyro2clp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newGyroscope2CLP] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newGyroscope3CLP(entityPool entity.Pool) (*gyro3clp.PoolSimulator, error) {
	corePool, err := gyro3clp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newGyroscope3CLP] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newGyroscopeECLP(entityPool entity.Pool) (*gyroeclp.PoolSimulator, error) {
	corePool, err := gyroeclp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newGyroscopeECLP] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newZkEraFinance(entityPool entity.Pool) (*zkera.PoolSimulator, error) {
	corePool, err := zkera.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newZkEraFinance] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSwaapV2(entityPool entity.Pool) (*swaapv2.PoolSimulator, error) {
	corePool, err := swaapv2.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSwaapV2] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBancorV3(entityPool entity.Pool) (*bancorv3.PoolSimulator, error) {
	corePool, err := bancorv3.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBancorV3] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newEtherfiEETH(entityPool entity.Pool) (*eeth.PoolSimulator, error) {
	corePool, err := eeth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newEtherfiEETH] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newEtherfiWEETH(entityPool entity.Pool) (*weeth.PoolSimulator, error) {
	corePool, err := weeth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newEtherfiWEETH] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newKelpRSETH(entityPool entity.Pool) (*rseth.PoolSimulator, error) {
	corePool, err := rseth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newKelpRSETH] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newRocketPoolRETH(entityPool entity.Pool) (*reth.PoolSimulator, error) {
	corePool, err := reth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newRocketPoolRETH] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newEthenaSusde(entityPool entity.Pool) (*susde.PoolSimulator, error) {
	corePool, err := susde.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newEthenaSusde] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMakerSavingsDai(entityPool entity.Pool) (*savingsdai.PoolSimulator, error) {
	corePool, err := savingsdai.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newMakerSavingsDai] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newHashflowV3(entityPool entity.Pool) (*hashflowv3.PoolSimulator, error) {
	corePool, err := hashflowv3.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newHashflowV3] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newNativeV1(entityPool entity.Pool) (*nativev1.PoolSimulator, error) {
	corePool, err := nativev1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newNativeV1] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBebop(entityPool entity.Pool) (*bebop.PoolSimulator, error) {
	corePool, err := bebop.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBebop] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDexalot(entityPool entity.Pool) (*dexalot.PoolSimulator, error) {
	corePool, err := dexalot.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDexalot] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newNomiswapStable(entityPool entity.Pool) (*nomiswapstable.PoolSimulator, error) {
	corePool, err := nomiswapstable.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newNomiswapStable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newRenzoEzETH(entityPool entity.Pool) (*ezeth.PoolSimulator, error) {
	corePool, err := ezeth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newRenzoEzETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBetrockUniETH(entityPool entity.Pool) (*unieth.PoolSimulator, error) {
	corePool, err := unieth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBetrockUniETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newPufferPufETH(entityPool entity.Pool) (*pufeth.PoolSimulator, error) {
	corePool, err := pufeth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newPufferPufETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSwellRSWETH(entityPool entity.Pool) (*rsweth.PoolSimulator, error) {
	corePool, err := rsweth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSwellRSWETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSwellSWETH(entityPool entity.Pool) (*sweth.PoolSimulator, error) {
	corePool, err := sweth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSwellSWETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSlipstream(entityPool entity.Pool) (*slipstream.PoolSimulator, error) {
	corePool, err := slipstream.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSlipstream] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newNuriV2(entityPool entity.Pool) (*nuriv2.PoolSimulator, error) {
	corePool, err := nuriv2.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newNuriV2] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newAmbientAEVM(entityPool entity.Pool, stateRoot common.Hash) (*aevmpoolwrapper.PoolWrapper, error) {
	unimplementedPool := dexlibprivate.NewUnimplementedPool(entityPool.Address, entityPool.Exchange, entityPool.Type)

	balanceSlots := f.getBalanceSlots(&entityPool)
	aevmPool, err := ambientaevm.NewPoolAEVM(
		entityPool,
		f.client,
		stateRoot,
		balanceSlots,
	)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newAmbientAEVM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return aevmpoolwrapper.NewPoolWrapperAsAEVMPool(unimplementedPool, aevmPool, f.client), nil
}

func (f *PoolFactory) newEtherVista(entityPool entity.Pool) (*ethervista.PoolSimulator, error) {
	corePool, err := ethervista.NewPoolSimulator(entityPool, f.config.ChainID)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newEtherVista] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMaverickV2AEVM(entityPool entity.Pool, stateRoot common.Hash) (*aevmpoolwrapper.PoolWrapper, error) {
	unimplementedPool := dexlibprivate.NewUnimplementedPool(entityPool.Address, entityPool.Exchange, entityPool.Type)

	balanceSlots := f.getBalanceSlots(&entityPool)
	aevmPool, err := maverickv2aevm.NewPoolAEVM(
		entityPool,
		f.client,
		stateRoot,
		balanceSlots,
	)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newMaverickV2AEVM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return aevmpoolwrapper.NewPoolWrapperAsAEVMPool(unimplementedPool, aevmPool, f.client), nil
}

func (f *PoolFactory) newLitePSM(entityPool entity.Pool) (*litepsm.PoolSimulator, error) {
	corePool, err := litepsm.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newLitePSM] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMkrSky(entityPool entity.Pool) (*mkrsky.PoolSimulator, error) {
	corePool, err := mkrsky.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newMkrSky] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDaiUsds(entityPool entity.Pool) (*daiusds.PoolSimulator, error) {
	corePool, err := daiusds.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDaiUsds] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newFluidVaultT1(entityPool entity.Pool) (*fluidvaultt1.PoolSimulator, error) {
	corePool, err := fluidvaultt1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newFluidVaultT1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newFluidDexT1(entityPool entity.Pool) (*fluiddext1.PoolSimulator, error) {
	corePool, err := fluiddext1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDexVaultT1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newUsd0PP(entityPool entity.Pool) (*usd0pp.PoolSimulator, error) {
	corePool, err := usd0pp.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newUsd0PP] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newRingSwap(entityPool entity.Pool) (*ringswap.PoolSimulator, error) {
	corePool, err := ringswap.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newRingSwap] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newPrimeETH(entityPool entity.Pool) (*primeeth.PoolSimulator, error) {
	corePool, err := primeeth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newPrimeETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newStaderETHx(entityPool entity.Pool) (*staderethx.PoolSimulator, error) {
	corePool, err := staderethx.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newStaderETHx] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newGenericSimpleRate(entityPool entity.Pool) (*generic_simple_rate.PoolSimulator, error) {
	corePool, err := generic_simple_rate.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newGenericSimpleRate] pool: [%s] » type: [%s] » exchange: [%s]",
			entityPool.Address,
			entityPool.Type,
			entityPool.Exchange,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMantleETH(entityPool entity.Pool) (*meth.PoolSimulator, error) {
	corePool, err := meth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newMantleETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newOndoUSDY(entityPool entity.Pool) (*ondo_usdy.PoolSimulator, error) {
	corePool, err := ondo_usdy.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newOndoUSDY] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newClipper(entityPool entity.Pool) (*clipper.PoolSimulator, error) {
	corePool, err := clipper.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newClipper] pool: [%s] » type: [%s] » exchange: [%s]",
			entityPool.Address,
			entityPool.Type,
			entityPool.Exchange,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDeltaSwapV1(entityPool entity.Pool) (*deltaswapv1.PoolSimulator, error) {
	corePool, err := deltaswapv1.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newDeltaSwapV1] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSfrxETH(entityPool entity.Pool) (*sfrxeth.PoolSimulator, error) {
	corePool, err := sfrxeth.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSfrxETH] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newSfrxETHConvertor(entityPool entity.Pool) (*sfrxeth_convertor.PoolSimulator, error) {
	corePool, err := sfrxeth_convertor.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newSfrxETHConvertor] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newEtherfiVampire(entityPool entity.Pool) (*etherfivampire.PoolSimulator, error) {
	corePool, err := etherfivampire.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newEtherfiVampire] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newMxTrading(entityPool entity.Pool) (*mxtrading.PoolSimulator, error) {
	corePool, err := mxtrading.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newMxTrading] pool: [%s] » type: [%s] » exchange: [%s]",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newLO1inch(entityPool entity.Pool) (*lo1inch.PoolSimulator, error) {
	corePool, err := lo1inch.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newLO1inch] pool: [%s] » type: [%s] cause by %v",
			entityPool.Address,
			entityPool.Type,
			err,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newVirtualFun(entityPool entity.Pool) (*virtualfun.PoolSimulator, error) {
	corePool, err := virtualfun.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newVirtualFun] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}

func (f *PoolFactory) newBeetsSS(entityPool entity.Pool) (*beetsss.PoolSimulator, error) {
	corePool, err := beetsss.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, errors.WithMessagef(
			ErrInitializePoolFailed,
			"[PoolFactory.newBeetsSS] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type)
	}

	return corePool, nil
}
