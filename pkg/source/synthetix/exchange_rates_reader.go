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

type ExchangeRatesReader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewExchangeRatesReader(cfg *Config, ethrpcClient *ethrpc.Client) *ExchangeRatesReader {
	return &ExchangeRatesReader{
		abi:          exchangeRates,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (r *ExchangeRatesReader) Read(ctx context.Context, poolState *PoolState) (*PoolState, error) {
	if err := r.readCurrencyKeyData(ctx, poolState); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read")
		return nil, err
	}

	return poolState, nil
}

// readCurrencyKeyData reads data which required currency key as parameter, included:
// - AggregatorAddresses
// - Aggregators
// - CurrencyKeyDecimals
// - CurrentRoundIds
func (r *ExchangeRatesReader) readCurrencyKeyData(ctx context.Context, poolState *PoolState) error {
	var (
		currencyKeys    = poolState.CurrencyKeys
		currencyKeysLen = len(currencyKeys)
		address         = poolState.Addresses.ExchangeRates

		aggregatorAddresses = make([]common.Address, currencyKeysLen)
		currencyKeyDecimals = make([]uint8, currencyKeysLen)
		currentRoundIds     = make([]*big.Int, currencyKeysLen)
	)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		req.
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesMethodAggregators,
				Params: []interface{}{keyByte},
			}, []interface{}{&aggregatorAddresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesMethodCurrencyKeyDecimals,
				Params: []interface{}{keyByte},
			}, []interface{}{&currencyKeyDecimals[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: ExchangeRatesMethodGetCurrentRoundId,
				Params: []interface{}{keyByte},
			}, []interface{}{&currentRoundIds[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not currency key data")
		return err
	}

	for i, key := range currencyKeys {
		poolState.AggregatorAddresses[key] = aggregatorAddresses[i]
		poolState.CurrencyKeyDecimals[key] = currencyKeyDecimals[i]
		poolState.CurrentRoundIds[key] = currentRoundIds[i]
	}

	return nil
}
