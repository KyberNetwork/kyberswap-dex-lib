package synthetix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type PoolStateReader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolStateReader(cfg *Config, ethrpcClient *ethrpc.Client) *PoolStateReader {
	return &PoolStateReader{
		abi:          synthetix,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

// Read reads all data required for finding route.
func (r *PoolStateReader) Read(ctx context.Context, address string) (*PoolState, error) {
	poolState := NewPoolState()

	if err := r.readBlockTimestamp(ctx, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("read block timestamp error")
		return nil, err
	}

	if err := r.readData(ctx, address, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("read data error")
		return nil, err
	}

	if err := r.readSynthTokens(ctx, address, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("read synth tokens error")
		return nil, err
	}

	return poolState, nil
}

// readSynthTokens reads synths data and require currencyKey parameter, included:
//   - Synths
//   - SynthsTotalSupply
func (r *PoolStateReader) readSynthTokens(ctx context.Context, address string, poolState *PoolState) error {
	var (
		synthsLen         = int(poolState.AvailableSynthCount.Int64())
		currencyKeys      = poolState.CurrencyKeys
		currencyKeysLen   = len(currencyKeys)
		synths            = make([]common.Address, currencyKeysLen)
		synthProxyResults = make([]common.Address, synthsLen)
		totalSupply       = make([]*big.Int, synthsLen)
	)

	// call synthetix
	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		req.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodGetSynthAddressByCurrencyKey,
			Params: []interface{}{keyByte},
		}, []interface{}{&synths[i]})
	}
	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read synth tokens")
		return err
	}

	// call multiCollateralSynth
	req = r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, synthAddress := range synths {
		req.AddCall(&ethrpc.Call{
			ABI:    multiCollateralSynth,
			Target: synthAddress.String(),
			Method: MultiCollateralSynthMethodGetProxy,
			Params: nil,
		}, []interface{}{&synthProxyResults[i]})
	}
	_, err = req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not reads synth tokens")
		return err
	}

	// call multiCollateralSynth
	req = r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, proxyAddress := range synthProxyResults {
		req.AddCall(&ethrpc.Call{
			ABI:    multiCollateralSynth,
			Target: proxyAddress.String(),
			Method: ProxyERC20MethodTotalSupply,
			Params: nil,
		}, []interface{}{&totalSupply[i]})
	}
	_, err = req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read synth tokens")
		return err
	}

	for i, key := range currencyKeys {
		poolState.Synths[key] = synthProxyResults[i]
		poolState.SynthsTotalSupply[key] = totalSupply[i]
		poolState.CurrencyKeyBySynth[synthProxyResults[i]] = key
	}

	return nil
}

// readData reads data which required no parameters, included:
//   - CurrencyKeys
//   - AvailableSynthCount
//   - sUSD
//   - TotalIssuedSUSD
func (r *PoolStateReader) readData(ctx context.Context, address string, poolState *PoolState) error {
	var (
		currencyKeysResult [][32]byte
		sUSDResult         [32]byte
		totalIssuedSUSD    *big.Int
	)

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodAvailableCurrencyKeys,
			Params: nil,
		}, []interface{}{&currencyKeysResult}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodAvailableSynthCount,
			Params: nil,
		}, []interface{}{&poolState.AvailableSynthCount}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodGetSUSDCurrencyKey,
			Params: nil,
		}, []interface{}{&sUSDResult})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}

	currencyKeys := make([]string, len(currencyKeysResult))
	for i, key := range currencyKeysResult {
		currencyKeys[i] = common.BytesToHash(key[:]).String()
	}
	poolState.CurrencyKeys = currencyKeys

	poolState.SUSDCurrencyKey = common.BytesToHash(sUSDResult[:]).String()

	req = r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: PoolStateMethodTotalIssuedSynths,
			Params: []interface{}{sUSDResult},
		}, []interface{}{&totalIssuedSUSD})

	_, err = req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}
	poolState.TotalIssuedSUSD = totalIssuedSUSD

	return nil
}

func (r *PoolStateReader) readBlockTimestamp(ctx context.Context, poolState *PoolState) error {
	blockTimestamp, err := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		GetCurrentBlockTimestamp()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read block timestamp")
		return err
	}

	poolState.BlockTimestamp = blockTimestamp

	return nil
}
