package synthetix

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
)

// ExchangeRatesWithDexPricing methods
const (
	ExchangeRatesWithDexPricingMethodAggregators                       = "aggregators"
	ExchangeRatesWithDexPricingMethodCurrencyKeyDecimals               = "currencyKeyDecimals"
	ExchangeRatesWithDexPricingMethodDexPriceAggregator                = "dexPriceAggregator"
	ExchangeRatesWithDexPricingMethodGetCurrentRoundId                 = "getCurrentRoundId"
	ExchangeRatesWithDexPricingMethodSynthTooVolatileForAtomicExchange = "synthTooVolatileForAtomicExchange"
)

type ExchangeRatesWithDexPricingReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewExchangeRatesWithDexPricingReader(scanService *service.ScanService) *ExchangeRatesWithDexPricingReader {
	return &ExchangeRatesWithDexPricingReader{
		abi:         abis.SynthetixExchangeRatesWithDexPricing,
		scanService: scanService,
	}
}

func (r *ExchangeRatesWithDexPricingReader) Read(
	ctx context.Context,
	poolState *PoolState,
) (*PoolState, error) {
	if err := r.readData(ctx, poolState); err != nil {
		return nil, err
	}

	if err := r.readCurrencyKeyData(ctx, poolState); err != nil {
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

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: ExchangeRatesWithDexPricingMethodDexPriceAggregator,
			Params: nil,
			Output: &poolState.DexPriceAggregatorAddress,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
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

	var calls []*repository.CallParams
	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		tokenCalls := []*repository.CallParams{
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodAggregators,
				Params: []interface{}{keyByte},
				Output: &aggregatorAddresses[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodCurrencyKeyDecimals,
				Params: []interface{}{keyByte},
				Output: &currencyKeyDecimals[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodGetCurrentRoundId,
				Params: []interface{}{keyByte},
				Output: &currentRoundIds[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesWithDexPricingMethodSynthTooVolatileForAtomicExchange,
				Params: []interface{}{keyByte},
				Output: &synthTooVolatileForAtomicExchanges[i],
			},
		}

		calls = append(calls, tokenCalls...)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
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
