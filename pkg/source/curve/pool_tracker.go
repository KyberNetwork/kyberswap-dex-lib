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
		return d.getNewPoolStateTypeBase(ctx, p)
	case PoolTypePlainOracle:
		return d.getNewPoolStateTypePlainOracle(ctx, p)
	case PoolTypeMeta:
		return d.getNewPoolStateTypeMeta(ctx, p)
	case PoolTypeAave:
		return d.getNewPoolStateTypeAave(ctx, p)
	case PoolTypeCompound:
		return d.getNewPoolStateTypeCompound(ctx, p)
	case PoolTypeTwo:
		return d.getNewPoolStateTypeTwo(ctx, p)
	case PoolTypeTricrypto:
		return d.getNewPoolStateTypeTricrypto(ctx, p)
	default:
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
		}).Errorf("pool type is not implemented")

		return entity.Pool{}, errors.New("pool type is not implemented")
	}
}
