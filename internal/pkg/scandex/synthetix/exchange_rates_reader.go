package synthetix

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/eth"
)

// ExchangeRates methods
const (
	ExchangeRatesMethodAggregators         = "aggregators"
	ExchangeRatesMethodCurrencyKeyDecimals = "currencyKeyDecimals"
	ExchangeRatesMethodGetCurrentRoundId   = "getCurrentRoundId"
)

type ExchangeRatesReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewExchangeRatesReader(scanService *service.ScanService) *ExchangeRatesReader {
	return &ExchangeRatesReader{
		abi:         abis.SynthetixExchangeRates,
		scanService: scanService,
	}
}

func (r *ExchangeRatesReader) Read(
	ctx context.Context,
	poolState *PoolState,
) (*PoolState, error) {
	if err := r.readCurrencyKeyData(ctx, poolState); err != nil {
		return nil, err
	}

	return poolState, nil
}

// readCurrencyKeyData reads data which required currency key as parameter, included:
// - AggregatorAddresses
// - Aggregators
// - CurrencyKeyDecimals
// - CurrentRoundIds
func (r *ExchangeRatesReader) readCurrencyKeyData(
	ctx context.Context,
	poolState *PoolState,
) error {
	currencyKeys := poolState.CurrencyKeys
	currencyKeysLen := len(currencyKeys)
	address := poolState.Addresses.ExchangeRates

	aggregatorAddresses := make([]common.Address, currencyKeysLen)
	currencyKeyDecimals := make([]uint8, currencyKeysLen)
	currentRoundIds := make([]*big.Int, currencyKeysLen)

	var calls []*repository.CallParams
	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		tokenCalls := []*repository.CallParams{
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesMethodAggregators,
				Params: []interface{}{keyByte},
				Output: &aggregatorAddresses[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesMethodCurrencyKeyDecimals,
				Params: []interface{}{keyByte},
				Output: &currencyKeyDecimals[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesMethodGetCurrentRoundId,
				Params: []interface{}{keyByte},
				Output: &currentRoundIds[i],
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
	}

	return nil
}
