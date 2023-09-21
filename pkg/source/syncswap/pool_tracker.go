package syncswap

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	switch p.Type {
	case poolTypeSyncSwapClassic:
		return d.getClassicPoolState(ctx, p)
	case poolTypeSyncSwapStable:
		return d.getStablePoolState(ctx, p)
	default:
		err := fmt.Errorf("can not get new pool state of address %s with type %s", p.Address, p.Type)
		logger.Errorf(err.Error())

		return entity.Pool{}, err
	}
}

func (d *PoolTracker) getClassicPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0 *big.Int
		reserves                 = make([]*big.Int, len(p.Tokens))
		vaultAddress             common.Address
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

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(ExtraClassicPool{
		SwapFee0To1:  swapFee0To1,
		SwapFee1To0:  swapFee1To0,
		VaultAddress: vaultAddress.Hex(),
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
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (d *PoolTracker) getStablePoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0                             *big.Int
		token0PrecisionMultiplier, token1PrecisionMultiplier *big.Int
		vaultAddress                                         common.Address
		reserves                                             = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
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
		ABI:    stablePoolABI,
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
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodToken0PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token0PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    stablePoolABI,
		Target: p.Address,
		Method: poolMethodToken1PrecisionMultiplier,
		Params: nil,
	}, []interface{}{&token1PrecisionMultiplier})

	calls.AddCall(&ethrpc.Call{
		ABI:    classicPoolABI,
		Target: p.Address,
		Method: poolMethodVault,
		Params: nil,
	}, []interface{}{&vaultAddress})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(ExtraStablePool{
		SwapFee0To1:               swapFee0To1,
		SwapFee1To0:               swapFee1To0,
		Token0PrecisionMultiplier: token0PrecisionMultiplier,
		Token1PrecisionMultiplier: token1PrecisionMultiplier,
		VaultAddress:              vaultAddress.Hex(),
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
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
