package curve

import (
	"context"
	"errors"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	if !cfg.SkipInitFactory {
		if err := initConfig(cfg, ethrpcClient); err != nil {
			return nil, err
		}
	}

	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	switch p.Type {
	case PoolTypeBase:
		return d.getNewPoolStateTypeBase(ctx, p, nil)
	case PoolTypePlainOracle:
		return d.getNewPoolStateTypePlainOracle(ctx, p, nil)
	case PoolTypeMeta:
		return d.getNewPoolStateTypeMeta(ctx, p, nil)
	case PoolTypeAave:
		return d.getNewPoolStateTypeAave(ctx, p, nil)
	case PoolTypeCompound:
		return d.getNewPoolStateTypeCompound(ctx, p, nil)
	case PoolTypeTwo:
		return d.getNewPoolStateTypeTwo(ctx, p, nil)
	case PoolTypeTricrypto:
		return d.getNewPoolStateTypeTricrypto(ctx, p, nil)
	default:
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
		}).Errorf("pool type is not implemented")

		return entity.Pool{}, errors.New("pool type is not implemented")
	}
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	switch p.Type {
	case PoolTypeBase:
		return d.getNewPoolStateTypeBase(ctx, p, params.Overrides)
	case PoolTypePlainOracle:
		return d.getNewPoolStateTypePlainOracle(ctx, p, params.Overrides)
	case PoolTypeMeta:
		return d.getNewPoolStateTypeMeta(ctx, p, params.Overrides)
	case PoolTypeAave:
		return d.getNewPoolStateTypeAave(ctx, p, params.Overrides)
	case PoolTypeCompound:
		return d.getNewPoolStateTypeCompound(ctx, p, params.Overrides)
	case PoolTypeTwo:
		return d.getNewPoolStateTypeTwo(ctx, p, params.Overrides)
	case PoolTypeTricrypto:
		return d.getNewPoolStateTypeTricrypto(ctx, p, params.Overrides)
	default:
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
		}).Errorf("pool type is not implemented")

		return entity.Pool{}, errors.New("pool type is not implemented")
	}
}
