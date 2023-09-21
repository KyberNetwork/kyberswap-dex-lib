package dmm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[DMM] Start getting new state of pool")

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var (
		tradeInfo TradeInfo
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    dmmPoolABI,
		Target: p.Address,
		Method: poolMethodGetTradeInfo,
		Params: nil,
	}, []interface{}{&tradeInfo})

	_, err := rpcRequest.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process RPC call")
		return entity.Pool{}, err
	}

	reserve0Str := tradeInfo.Reserve0.String()
	reserve1Str := tradeInfo.Reserve1.String()
	vReserve0Str := tradeInfo.VReserve0.String()
	vReserve1Str := tradeInfo.VReserve1.String()

	extra := ExtraField{
		VReserves: []string{
			vReserve0Str,
			vReserve1Str,
		},
		FeeInPrecision: tradeInfo.FeeInPrecision.String(),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserve0Str,
		reserve1Str,
	}

	logger.Infof("[DMM] Finish getting new state of pool: %v", p.Address)

	return p, nil
}
