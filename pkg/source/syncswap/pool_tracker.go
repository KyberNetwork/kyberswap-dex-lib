package syncswap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type Vault struct {
	VaultAddress string `json:"vaultAddress"`
}

var _ = pooltrack.RegisterFactoryCE0(DexTypeSyncSwap, NewPoolTracker)

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
	var extra Vault
	var getVaultBalances func(calls *ethrpc.Request, b0, b1 **big.Int) bool
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal extra data")
		return entity.Pool{}, err
	}
	if extra.VaultAddress != "" {
		getVaultBalances = d.getVaultBalances(extra.VaultAddress, p)
	} else {
		getVaultBalances = func(calls *ethrpc.Request, b0, b1 **big.Int) bool { return false }
	}
	switch p.Type {
	case PoolTypeSyncSwapClassic:
		return d.getClassicPoolState(ctx, p, getVaultBalances)
	case PoolTypeSyncSwapStable:
		return d.getStablePoolState(ctx, p, getVaultBalances)
	default:
		err := fmt.Errorf("can not get new pool state of address %s with type %s", p.Address, p.Type)
		logger.Errorf(err.Error())

		return entity.Pool{}, err
	}
}

func (d *PoolTracker) getVaultBalances(vault string, p entity.Pool) func(calls *ethrpc.Request, b0, b1 **big.Int) bool {
	return func(calls *ethrpc.Request, b0, b1 **big.Int) bool {
		calls.AddCall(&ethrpc.Call{
			ABI:    classicPoolABI,
			Target: p.Tokens[0].Address,
			Method: poolMethodBalanceOf,
			Params: []interface{}{
				common.HexToAddress(vault),
			},
		}, []interface{}{b0})

		calls.AddCall(&ethrpc.Call{
			ABI:    classicPoolABI,
			Target: p.Tokens[1].Address,
			Method: poolMethodBalanceOf,
			Params: []interface{}{
				common.HexToAddress(vault),
			},
		}, []interface{}{b1})
		return true
	}
}

func (d *PoolTracker) getClassicPoolState(ctx context.Context, p entity.Pool, getVaultBalances func(calls *ethrpc.Request, b0, b1 **big.Int) bool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0     *big.Int
		reserves                     = make([]*big.Int, len(p.Tokens))
		vaultAddress                 common.Address
		vaultBalance0, vaultBalance1 *big.Int
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

	ok := getVaultBalances(calls, &vaultBalance0, &vaultBalance1)

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	if !ok {
		calls = d.ethrpcClient.NewRequest().SetContext(ctx)
		d.getVaultBalances(vaultAddress.Hex(), p)(calls, &vaultBalance0, &vaultBalance1)
		if _, err := calls.Aggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"address": p.Address,
				"error":   err,
			}).Errorf("failed to get state of the pool")
			return entity.Pool{}, err
		}
	}

	extraBytes, err := json.Marshal(ExtraClassicPool{
		SwapFee0To1:   swapFee0To1,
		SwapFee1To0:   swapFee1To0,
		VaultAddress:  vaultAddress.Hex(),
		VaultBalance0: vaultBalance0,
		VaultBalance1: vaultBalance1,
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

func (d *PoolTracker) getStablePoolState(ctx context.Context, p entity.Pool, getVaultBalances func(calls *ethrpc.Request, b0, b1 **big.Int) bool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		swapFee0To1, swapFee1To0                             *big.Int
		token0PrecisionMultiplier, token1PrecisionMultiplier *big.Int
		vaultAddress                                         common.Address
		reserves                                             = make([]*big.Int, len(p.Tokens))
		vaultBalance0, vaultBalance1                         *big.Int
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

	ok := getVaultBalances(calls, &vaultBalance0, &vaultBalance1)

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}
	if !ok {
		calls = d.ethrpcClient.NewRequest().SetContext(ctx)
		d.getVaultBalances(vaultAddress.Hex(), p)(calls, &vaultBalance0, &vaultBalance1)
		if _, err := calls.Aggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"address": p.Address,
				"error":   err,
			}).Errorf("failed to get state of the pool")
			return entity.Pool{}, err
		}
	}
	extraBytes, err := json.Marshal(ExtraStablePool{
		SwapFee0To1:               swapFee0To1,
		SwapFee1To0:               swapFee1To0,
		Token0PrecisionMultiplier: token0PrecisionMultiplier,
		Token1PrecisionMultiplier: token1PrecisionMultiplier,
		VaultAddress:              vaultAddress.Hex(),
		VaultBalance0:             vaultBalance0,
		VaultBalance1:             vaultBalance1,
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
