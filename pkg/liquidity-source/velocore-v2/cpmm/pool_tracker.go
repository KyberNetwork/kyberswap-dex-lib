package cpmm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
	cfg          *Config
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
		cfg:          cfg,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"pool": p.Address,
		"type": DexType,
	}).Info("start getting new state of pool")
	defer func(s time.Time) {
		logger.WithFields(logger.Fields{
			"pool": p.Address,
			"type": DexType,
			"exec": time.Since(s).String(),
		}).Info("finish getting new state of pool")
	}(time.Now())

	var (
		reserves      [maxPoolTokenNumber]*big.Int
		fee1e9        uint32
		feeMultiplier *big.Int
	)

	req := d.ethrpcClient.R().SetRequireSuccess(true)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodPoolBalances,
		Params: nil,
	}, []interface{}{&reserves})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodFee1e9,
		Params: nil,
	}, []interface{}{&fee1e9})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodFeeMultiplier,
		Params: nil,
	}, []interface{}{&feeMultiplier})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool": p.Address,
			"type": DexType,
		}).Error(err.Error())
		return entity.Pool{}, err
	}

	var (
		staticExtra  StaticExtra
		poolReserves entity.PoolReserves
	)
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"pool": p.Address,
			"type": DexType,
		}).Error(err.Error())
		return entity.Pool{}, err
	}
	for i := 0; i < int(staticExtra.PoolTokenNumber); i++ {
		poolReserves = append(poolReserves, reserves[i].String())
	}

	extra := Extra{
		ChainID:       d.cfg.ChainID,
		Fee1e9:        fee1e9,
		FeeMultiplier: feeMultiplier,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool": p.Address,
			"type": DexType,
		}).Error(err.Error())
		return entity.Pool{}, err
	}

	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = poolReserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
