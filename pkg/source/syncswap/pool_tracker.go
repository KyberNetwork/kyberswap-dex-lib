package syncswap

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
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

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[Syncswap] Start getting new state of pool")

	var (
		swapFee0To1, swapFee1To0 *big.Int
		reserves                 = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[0].Address),
			common.HexToAddress(p.Tokens[1].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee0To1})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFee,
		Params: []interface{}{
			common.HexToAddress(addressZero),
			common.HexToAddress(p.Tokens[1].Address),
			common.HexToAddress(p.Tokens[0].Address),
			[]byte{},
		},
	}, []interface{}{&swapFee1To0})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		SwapFee0To1: swapFee0To1,
		SwapFee1To0: swapFee1To0,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{reserves[0].String(), reserves[1].String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[Syncswap] Finish getting new state of pool")

	return p, nil
}
