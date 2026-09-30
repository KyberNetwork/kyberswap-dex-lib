package camelot

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/timer"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeCamelot, NewPoolTracker)

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

	factory, blockNumber, err := d.getFactory(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get factory")
		return entity.Pool{}, err
	}

	pair, _, err := d.getPair(ctx, p.Address, blockNumber)
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
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (d *PoolTracker) getPair(ctx context.Context, address string, blockNumber *big.Int) (*Pair, *big.Int, error) {
	var rpcPair struct {
		Reserve0             *uint256.Int
		Reserve1             *uint256.Int
		StableSwap           bool
		Token0FeePercent     uint16
		Token1FeePercent     uint16
		PrecisionMultiplier0 *uint256.Int
		PrecisionMultiplier1 *uint256.Int
	}

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}
	req.
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodStableSwap,
			Params: nil,
		}, []any{&rpcPair.StableSwap}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodToken0FeePercent,
			Params: nil,
		}, []any{&rpcPair.Token0FeePercent}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodToken1FeePercent,
			Params: nil,
		}, []any{&rpcPair.Token1FeePercent}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodPrecisionMultiplier0,
			Params: nil,
		}, []any{&rpcPair.PrecisionMultiplier0}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodPrecisionMultiplier1,
			Params: nil,
		}, []any{&rpcPair.PrecisionMultiplier1}).
		AddCall(&ethrpc.Call{
			ABI:    camelotPairABI,
			Target: address,
			Method: pairMethodGetReserves,
			Params: nil,
		}, []any{&rpcPair})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pair info")
		return nil, nil, err
	}

	return &Pair{
		Reserve0:             u256ToBig(rpcPair.Reserve0),
		Reserve1:             u256ToBig(rpcPair.Reserve1),
		StableSwap:           rpcPair.StableSwap,
		Token0FeePercent:     rpcPair.Token0FeePercent,
		Token1FeePercent:     rpcPair.Token1FeePercent,
		PrecisionMultiplier0: u256ToBig(rpcPair.PrecisionMultiplier0),
		PrecisionMultiplier1: u256ToBig(rpcPair.PrecisionMultiplier1),
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) getFactory(ctx context.Context) (*Factory, *big.Int, error) {
	var (
		feeTo         common.Address
		ownerFeeShare *uint256.Int
	)
	req := d.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodFeeTo,
			Params: nil,
		}, []any{&feeTo}).
		AddCall(&ethrpc.Call{
			ABI:    camelotFactoryABI,
			Target: d.cfg.FactoryAddress,
			Method: factoryMethodOwnerFeeShare,
			Params: nil,
		}, []any{&ownerFeeShare})

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get factory")
		return nil, nil, err
	}

	return &Factory{
		FeeTo:         feeTo,
		OwnerFeeShare: u256ToBig(ownerFeeShare),
	}, resp.BlockNumber, nil
}
