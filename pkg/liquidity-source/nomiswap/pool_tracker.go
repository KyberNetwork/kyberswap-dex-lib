package nomiswap

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type NomiStableReserve struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}
func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee                                              uint32
		token0PrecisionMultiplier, token1PrecisionMultiplier *big.Int
		reserve                                              NomiStableReserve
		A                                                    *big.Int
	)
	stablePoolABI, _ := NomiStablePoolMetaData.GetAbi()
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    *stablePoolABI,
		Target: p.Address,
		Method: "swapFee",
		Params: nil,
	}, []interface{}{&swapFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    *stablePoolABI,
		Target: p.Address,
		Method: "token0PrecisionMultiplier",
		Params: nil,
	}, []interface{}{&token0PrecisionMultiplier})
	calls.AddCall(&ethrpc.Call{
		ABI:    *stablePoolABI,
		Target: p.Address,
		Method: "token1PrecisionMultiplier",
		Params: nil,
	}, []interface{}{&token1PrecisionMultiplier})
	calls.AddCall(&ethrpc.Call{
		ABI:    *stablePoolABI,
		Target: p.Address,
		Method: "getReserves",
		Params: nil,
	}, []interface{}{&reserve})
	calls.AddCall(&ethrpc.Call{
		ABI:    *stablePoolABI,
		Target: p.Address,
		Method: "getA",
		Params: nil,
	}, []interface{}{&A})
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(ExtraStablePool{
		SwapFee:                   swapFee,
		Token0PrecisionMultiplier: uint256.MustFromBig(token0PrecisionMultiplier),
		Token1PrecisionMultiplier: uint256.MustFromBig(token1PrecisionMultiplier),
		A:                         uint256.MustFromBig(A),
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}
	p.Reserves = entity.PoolReserves{reserve.Reserve0.String(), reserve.Reserve1.String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
