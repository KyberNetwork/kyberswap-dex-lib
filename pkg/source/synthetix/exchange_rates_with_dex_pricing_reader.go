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

type ExchangeRatesWithDexPricingReader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewExchangeRatesWithDexPricingReader(cfg *Config, ethrpcClient *ethrpc.Client) *ExchangeRatesWithDexPricingReader {
	return &ExchangeRatesWithDexPricingReader{
		abi:          exchangeRatesWithDexPricing,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (r *ExchangeRatesWithDexPricingReader) Read(ctx context.Context, poolState *PoolState) (*PoolState, error) {
	if err := r.readData(ctx, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return nil, err
	}

	if err := r.readCurrencyKeyData(ctx, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read currency key data")
		return nil, err
	}

	return poolState, nil
}

// readData reads data which required no parameters, included:
// - DexPriceAggregator
func (r *ExchangeRatesWithDexPricingReader) readData(
	ctx context.Context,
	poolState *PoolState,
) error {
	address := poolState.Addresses.ExchangeRates

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: ExchangeRatesWithDexPricingMethodDexPriceAggregator,
			Params: nil,
		}, []interface{}{&poolState.DexPriceAggregatorAddress})

	_, err := req.Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}

	return nil
}

// readCurrencyKeyData reads data which required currency key as parameter, included:
// - AggregatorAddresses
// - Aggregators
// - CurrencyKeyDecimals
// - CurrentRoundIds
func (r *ExchangeRatesWithDexPricingReader) readCurrencyKeyData(
	ctx context.Context,
	poolState *PoolState,
) error {
	currencyKeys := poolState.CurrencyKeys
	currencyKeysLen := len(currencyKeys)
	address := poolState.Addresses.ExchangeRates

	aggregatorAddresses := make([]common.Address, currencyKeysLen)
	currencyKeyDecimals := make([]uint8, currencyKeysLen)
	currentRoundIds := make([]*big.Int, currencyKeysLen)
	synthTooVolatileForAtomicExchanges := make([]bool, currencyKeysLen)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		req.
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodAggregators,
				Params: []interface{}{keyByte},
			}, []interface{}{&aggregatorAddresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodCurrencyKeyDecimals,
				Params: []interface{}{keyByte},
			}, []interface{}{&currencyKeyDecimals[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodGetCurrentRoundId,
				Params: []interface{}{keyByte},
			}, []interface{}{&currentRoundIds[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodSynthTooVolatileForAtomicExchange,
				Params: []interface{}{keyByte},
			}, []interface{}{&synthTooVolatileForAtomicExchanges[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read currency key data")
		return err
	}

	for i, key := range currencyKeys {
		poolState.AggregatorAddresses[key] = aggregatorAddresses[i]
		poolState.CurrencyKeyDecimals[key] = currencyKeyDecimals[i]
		poolState.CurrentRoundIds[key] = currentRoundIds[i]
		poolState.SynthTooVolatileForAtomicExchanges[key] = synthTooVolatileForAtomicExchanges[i]
	}

	return nil
}
