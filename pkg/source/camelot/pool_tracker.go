package camelot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/timer"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	finish := timer.Start(fmt.Sprintf("[%s] get new pool state", d.cfg.DexID))
	defer finish()

	factory, err := d.getFactory(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get factory")
		return entity.Pool{}, err
	}

	pair, err := d.getPair(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair")
		return entity.Pool{}, err
	}

	extra := Extra{
		StableSwap:           pair.StableSwap,
		Token0FeePercent:     big.NewInt(int64(pair.Token0FeePercent)),
		Token1FeePercent:     big.NewInt(int64(pair.Token1FeePercent)),
		PrecisionMultiplier0: pair.PrecisionMultiplier0,
		PrecisionMultiplier1: pair.PrecisionMultiplier1,
		Factory:              factory,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"pool":  p.Address,
			"error": err,
		}).Error("can not marshal extra")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{pair.Reserve0.String(), pair.Reserve1.String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (d *PoolTracker) getPair(ctx context.Context, address string) (*Pair, error) {
	var pair Pair

	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodStableSwap,
			Params: nil,
		}, []interface{}{&pair.StableSwap}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodToken0FeePercent,
			Params: nil,
		}, []interface{}{&pair.Token0FeePercent}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodToken1FeePercent,
			Params: nil,
		}, []interface{}{&pair.Token1FeePercent}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodPrecisionMultiplier0,
			Params: nil,
		}, []interface{}{&pair.PrecisionMultiplier0}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodPrecisionMultiplier1,
			Params: nil,
		}, []interface{}{&pair.PrecisionMultiplier1}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodGetReserves,
			Params: nil,
		}, []interface{}{&pair})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair info")
		return nil, err
	}

	return &pair, nil
}

func (d *PoolTracker) getFactory(ctx context.Context) (*Factory, error) {
	var factory Factory
	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodFeeTo,
			Params: nil,
		}, []interface{}{&factory.FeeTo}).
		AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodOwnerFeeShare,
			Params: nil,
		}, []interface{}{&factory.OwnerFeeShare})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get factory")
		return nil, err
	}

	return &factory, nil
}
