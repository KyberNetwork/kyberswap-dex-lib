package getroutev2

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/balancerstable"
	"github.com/KyberNetwork/router-service/internal/pkg/core/balancerweighted"
	"github.com/KyberNetwork/router-service/internal/pkg/core/camelot"
	curveAave "github.com/KyberNetwork/router-service/internal/pkg/core/curve-aave"
	curveBase "github.com/KyberNetwork/router-service/internal/pkg/core/curve-base"
	curveCompound "github.com/KyberNetwork/router-service/internal/pkg/core/curve-compound"
	curveMeta "github.com/KyberNetwork/router-service/internal/pkg/core/curve-meta"
	curvePlainOracle "github.com/KyberNetwork/router-service/internal/pkg/core/curve-plain-oracle"
	curveTricrypto "github.com/KyberNetwork/router-service/internal/pkg/core/curve-tricrypto"
	curveTwo "github.com/KyberNetwork/router-service/internal/pkg/core/curve-two"
	"github.com/KyberNetwork/router-service/internal/pkg/core/dmm"
	"github.com/KyberNetwork/router-service/internal/pkg/core/dodo"
	"github.com/KyberNetwork/router-service/internal/pkg/core/fraxswap"
	"github.com/KyberNetwork/router-service/internal/pkg/core/gmx"
	"github.com/KyberNetwork/router-service/internal/pkg/core/lido"
	"github.com/KyberNetwork/router-service/internal/pkg/core/limitorder"
	"github.com/KyberNetwork/router-service/internal/pkg/core/madmex"
	"github.com/KyberNetwork/router-service/internal/pkg/core/makerpsm"
	"github.com/KyberNetwork/router-service/internal/pkg/core/metavault"
	"github.com/KyberNetwork/router-service/internal/pkg/core/platypus"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/core/promm"
	"github.com/KyberNetwork/router-service/internal/pkg/core/saddle"
	"github.com/KyberNetwork/router-service/internal/pkg/core/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/core/uni"
	"github.com/KyberNetwork/router-service/internal/pkg/core/univ3"
	"github.com/KyberNetwork/router-service/internal/pkg/core/velodrome"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	ErrInitializePoolFailed    = errors.New("initialize pool failed")
	ErrBasePoolNotFound        = errors.New("base pool not found")
	ErrPoolTypeFactoryNotFound = errors.New("there is no factory for the pool type")
	ErrUnmarshalDataFailed     = errors.New("unmarshall data failed")
)

type PoolFactory struct {
	config PoolFactoryConfig
}

func NewPoolFactory(config PoolFactoryConfig) *PoolFactory {
	return &PoolFactory{
		config: config,
	}
}

func (f *PoolFactory) NewPoolByAddress(_ context.Context, pools []*entity.Pool) map[string]poolpkg.IPool {
	curveBasePoolByAddress := f.getCurveBasePoolByAddress(pools)
	curvePlainOraclePoolByAddress := f.getCurvePlainOraclePoolByAddress(pools)

	basePools := combinePoolsMap(curveBasePoolByAddress, curvePlainOraclePoolByAddress)

	poolByAddress := make(map[string]poolpkg.IPool, len(pools))
	for _, pool := range pools {
		switch pool.Type {
		case constant.PoolTypes.CurveBase:
			iPool, ok := curveBasePoolByAddress[pool.Address]
			if !ok {
				continue // NOTE: already warned in getCurveBasePoolByAddress
			}

			poolByAddress[iPool.GetAddress()] = iPool

		case constant.PoolTypes.CurvePlainOracle:
			iPool, ok := curvePlainOraclePoolByAddress[pool.Address]
			if !ok {
				continue // NOTE: already warned in getCurvePlainOraclePoolByAddress
			}

			poolByAddress[iPool.GetAddress()] = iPool

		case constant.PoolTypes.CurveMeta:
			iPool, err := f.newCurveMeta(*pool, basePools)
			if err != nil {
				logger.Debugf(err.Error())
				continue
			}

			poolByAddress[iPool.GetAddress()] = iPool

		default:
			iPool, err := f.newPool(*pool)
			if err != nil {
				logger.Debugf(err.Error())
				continue
			}

			poolByAddress[iPool.GetAddress()] = iPool
		}
	}

	return poolByAddress
}

func (f *PoolFactory) getCurveBasePoolByAddress(
	entityPools []*entity.Pool,
) map[string]*curveBase.Pool {
	curveBasePoolByAddress := make(map[string]*curveBase.Pool)

	for _, entityPool := range entityPools {
		if entityPool.Type != constant.PoolTypes.CurveBase {
			continue
		}

		curveBasePool, err := f.newCurveBase(*entityPool)
		if err != nil {
			logger.Warn(err.Error())
			continue
		}

		curveBasePoolByAddress[curveBasePool.GetAddress()] = curveBasePool
	}

	return curveBasePoolByAddress
}

func (f *PoolFactory) getCurvePlainOraclePoolByAddress(
	entityPools []*entity.Pool,
) map[string]*curvePlainOracle.Pool {
	curvePlainOraclePoolByAddress := make(map[string]*curvePlainOracle.Pool)

	for _, entityPool := range entityPools {
		if entityPool.Type != constant.PoolTypes.CurvePlainOracle {
			continue
		}

		curvePlainOraclePool, err := f.newCurvePlainOracle(*entityPool)
		if err != nil {
			logger.Warn(err.Error())
			continue
		}

		curvePlainOraclePoolByAddress[curvePlainOraclePool.GetAddress()] = curvePlainOraclePool
	}

	return curvePlainOraclePoolByAddress
}

// newPool receives entity.Pool, based on its type to return matched factory method
// if there is no matched factory method, it returns ErrPoolTypeFactoryNotFound
func (f *PoolFactory) newPool(entityPool entity.Pool) (poolpkg.IPool, error) {
	switch entityPool.Type {
	case constant.PoolTypes.Uni, constant.PoolTypes.Firebird:
		return f.newUni(entityPool)
	case constant.PoolTypes.UniV3:
		return f.newUniV3(entityPool)
	case constant.PoolTypes.Saddle:
		return f.newSaddle(entityPool)
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
	case constant.PoolTypes.BalancerWeighted:
		return f.newBalancerWeighted(entityPool)
	case constant.PoolTypes.BalancerStable, constant.PoolTypes.BalancerMetaStable:
		return f.newBalancerStable(entityPool)
	case constant.PoolTypes.DodoClassical, constant.PoolTypes.DodoStable,
		constant.PoolTypes.DodoVendingMachine, constant.PoolTypes.DodoPrivate:
		return f.newDoDo(entityPool)
	case constant.PoolTypes.Velodrome:
		return f.newVelodrome(entityPool)
	case constant.PoolTypes.PlatypusBase, constant.PoolTypes.PlatypusPure, constant.PoolTypes.PlatypusAvax:
		return f.newPlatypus(entityPool)
	case constant.PoolTypes.GMX:
		return f.newGMX(entityPool)
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
	case constant.PoolTypes.Fraxswap:
		return f.newFraxswap(entityPool)
	case constant.PoolTypes.Camelot:
		return f.newCamelot(entityPool)
	case constant.PoolTypes.LimitOrder:
		return f.newLimitOrder(entityPool)
	default:
		return nil, errors.Wrapf(
			ErrPoolTypeFactoryNotFound,
			"[PoolFactory.newPool] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

}

func (f *PoolFactory) newUni(entityPool entity.Pool) (*uni.Pool, error) {
	corePool, err := uni.NewPool(entityPool)
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

func (f *PoolFactory) newUniV3(entityPool entity.Pool) (*univ3.Pool, error) {
	corePool, err := univ3.NewPool(entityPool, f.config.ChainID)
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

func (f *PoolFactory) newSaddle(entityPool entity.Pool) (*saddle.Pool, error) {
	corePool, err := saddle.NewPool(entityPool)
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

func (f *PoolFactory) newDMM(entityPool entity.Pool) (*dmm.Pool, error) {
	corePool, err := dmm.NewPool(entityPool)
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

func (f *PoolFactory) newElastic(entityPool entity.Pool) (*promm.Pool, error) {
	corePool, err := promm.NewPool(entityPool, f.config.ChainID)
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

func (f *PoolFactory) newCurveBase(entityPool entity.Pool) (*curveBase.Pool, error) {
	corePool, err := curveBase.NewPool(entityPool)
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
	corePool, err := curvePlainOracle.NewPool(entityPool)
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

	curveMetaPool, err := curveMeta.NewPool(entityPool, basePool)
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
	corePool, err := curveAave.NewPool(entityPool)
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
	corePool, err := curveCompound.NewPool(entityPool)
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
	corePool, err := curveTricrypto.NewPool(entityPool)
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
	corePool, err := curveTwo.NewPool(entityPool)
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

func (f *PoolFactory) newBalancerWeighted(entityPool entity.Pool) (*balancerweighted.WeightedPool2Tokens, error) {
	corePool, err := balancerweighted.NewPool(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerWeighted] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newBalancerStable(entityPool entity.Pool) (*balancerstable.StablePool, error) {
	corePool, err := balancerstable.NewPool(entityPool)
	if err != nil {
		return nil, errors.Wrapf(
			ErrInitializePoolFailed,
			"[PoolFactory.newBalancerStable] pool: [%s] » type: [%s]",
			entityPool.Address,
			entityPool.Type,
		)
	}

	return corePool, nil
}

func (f *PoolFactory) newDoDo(entityPool entity.Pool) (*dodo.Pool, error) {
	corePool, err := dodo.NewPool(entityPool)
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

func (f *PoolFactory) newVelodrome(entityPool entity.Pool) (*velodrome.Pool, error) {
	corePool, err := velodrome.NewPool(entityPool)
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

func (f *PoolFactory) newPlatypus(entityPool entity.Pool) (*platypus.Pool, error) {
	corePool, err := platypus.NewPool(entityPool, f.config.ChainID)
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

func (f *PoolFactory) newGMX(entityPool entity.Pool) (*gmx.Pool, error) {
	corePool, err := gmx.NewPool(entityPool)
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

func (f *PoolFactory) newMadMex(entityPool entity.Pool) (*madmex.Pool, error) {
	corePool, err := madmex.NewPool(entityPool)
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

func (f *PoolFactory) newMetavault(entityPool entity.Pool) (*metavault.Pool, error) {
	corePool, err := metavault.NewPool(entityPool)
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

func (f *PoolFactory) newMakerPSm(entityPool entity.Pool) (*makerpsm.Pool, error) {
	corePool, err := makerpsm.NewPool(entityPool)
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

func (f *PoolFactory) newSynthetix(entityPool entity.Pool) (*synthetix.Pool, error) {
	corePool, err := synthetix.NewPool(entityPool, f.config.ChainID)
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

func (f *PoolFactory) newLido(entityPool entity.Pool) (*lido.Pool, error) {
	corePool, err := lido.NewPool(entityPool)
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

func (f *PoolFactory) newFraxswap(entityPool entity.Pool) (*fraxswap.Pool, error) {
	corePool, err := fraxswap.NewPool(entityPool)
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
func (f *PoolFactory) newLimitOrder(entityPool entity.Pool) (*limitorder.Pool, error) {
	corePool, err := limitorder.NewPool(entityPool)
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

func (f *PoolFactory) newCamelot(entityPool entity.Pool) (*camelot.Pool, error) {
	corePool, err := camelot.NewPool(entityPool)
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

func combinePoolsMap(
	curveBasePools map[string]*curveBase.Pool,
	curvePlainOraclePools map[string]*curvePlainOracle.Pool,
) map[string]curveMeta.ICurveBasePool {
	m := make(map[string]curveMeta.ICurveBasePool)
	for k, v := range curveBasePools {
		m[k] = v
	}
	for k, v := range curvePlainOraclePools {
		m[k] = v
	}

	return m
}
