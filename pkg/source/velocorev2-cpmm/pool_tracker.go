package velocorev2cpmm

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[VelocoreV2 CPMM] Start getting new state of pool: %v", p.Address)

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var (
		reserves      [maxPoolTokenNumber]*big.Int
		fee1e9        uint32
		feeMultiplier *big.Int
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodPoolBalances,
		Params: nil,
	}, []interface{}{&reserves})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodFee1e9,
		Params: nil,
	}, []interface{}{&fee1e9})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodFeeMultiplier,
		Params: nil,
	}, []interface{}{&feeMultiplier})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.Errorf("failed to process Aggregate for pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	var (
		staticExtra  StaticExtra
		poolReserves entity.PoolReserves
	)
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.Errorf("failed to unmarshal static extra for pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}
	for i := 0; i < int(staticExtra.PoolTokenNumber); i++ {
		poolReserves = append(poolReserves, reserves[i].String())
	}

	extra := Extra{
		Fee1e9:        fee1e9,
		FeeMultiplier: feeMultiplier.String(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.Errorf("failed to marshal extra for pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	p.Reserves = poolReserves
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.Infof("[VelocoreV2 CPMM] Finish getting new state of pool: %v", p.Address)

	return p, nil
}
