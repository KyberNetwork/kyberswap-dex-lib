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

type SystemSettingsReader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewSystemSettingsReader(cfg *Config, ethrpcClient *ethrpc.Client) *SystemSettingsReader {
	return &SystemSettingsReader{
		abi:          systemSettings,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (r *SystemSettingsReader) Read(ctx context.Context, poolState *PoolState) (*SystemSettings, error) {
	systemSettings := NewSystemSettings()

	if err := r.readData(ctx, poolState.Addresses.SystemSettings, systemSettings); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return nil, err
	}

	if err := r.readDynamicFeeConfig(ctx, poolState.Addresses.SystemSettings, systemSettings); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read dynamic fee config")
		return nil, err
	}

	if err := r.readCurrencyKeyData(ctx, poolState.Addresses.SystemSettings, systemSettings, poolState.CurrencyKeys); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read currency key data")
		return nil, err
	}

	if err := r.readTokenData(ctx, systemSettings, poolState.CurrencyKeys); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read token data")
		return nil, err
	}

	return systemSettings, nil
}

// readTokenData reads token data, included:
// - Decimals
// - Symbol
func (r *SystemSettingsReader) readTokenData(ctx context.Context, systemSettings *SystemSettings, currencyKeys []string) error {
	var (
		atomicEquivalentForDexPricingAddresses = systemSettings.AtomicEquivalentForDexPricingAddresses
		tokensLen                              = len(atomicEquivalentForDexPricingAddresses)

		decimals = make([]uint8, tokensLen)
		symbols  = make([]string, tokensLen)
	)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, currencyKey := range currencyKeys {
		address := atomicEquivalentForDexPricingAddresses[currencyKey]
		if eth.IsZeroAddress(address) {
			continue
		}

		req.
			AddCall(&ethrpc.Call{
				ABI:    erc20,
				Target: address.String(),
				Method: TokenMethodDecimals,
				Params: nil,
			}, []any{&decimals[i]}).
			AddCall(&ethrpc.Call{
				ABI:    erc20,
				Target: address.String(),
				Method: TokenMethodSymbol,
				Params: nil,
			}, []any{&symbols[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read token data")
		return err
	}

	for i, currencyKey := range currencyKeys {
		address := atomicEquivalentForDexPricingAddresses[currencyKey]
		if eth.IsZeroAddress(address) {
			continue
		}

		token := Token{
			Address:  address,
			Decimals: decimals[i],
			Symbol:   symbols[i],
		}
		systemSettings.AtomicEquivalentForDexPricing[currencyKey] = token
	}

	return nil
}

// readCurrencyKeyData reads data which required currency key as parameter, included:
// - PureChainlinkPriceForAtomicSwapsEnabled
// - AtomicEquivalentForDexPricingAddresses
// - AtomicVolatilityConsiderationWindow
// - AtomicVolatilityUpdateThreshold
// - AtomicExchangeFeeRate
// - ExchangeFeeRate
func (r *SystemSettingsReader) readCurrencyKeyData(ctx context.Context, address string, systemSettings *SystemSettings, currencyKeys []string) error {
	var (
		currencyKeysLen = len(currencyKeys)

		pureChainlinkPriceForAtomicSwapsEnabled = make([]bool, currencyKeysLen)
		atomicEquivalentForDexPricingAddresses  = make([]common.Address, currencyKeysLen)
		atomicVolatilityConsiderationWindows    = make([]*big.Int, currencyKeysLen)
		atomicVolatilityUpdateThresholds        = make([]*big.Int, currencyKeysLen)
		atomicExchangeFeeRates                  = make([]*big.Int, currencyKeysLen)
		exchangeFeeRates                        = make([]*big.Int, currencyKeysLen)
	)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		req.
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodPureChainlinkPriceForAtomicSwapsEnabled,
				Params: []any{keyByte},
			}, []any{&pureChainlinkPriceForAtomicSwapsEnabled[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicEquivalentForDexPricing,
				Params: []any{keyByte},
			}, []any{&atomicEquivalentForDexPricingAddresses[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicVolatilityConsiderationWindow,
				Params: []any{keyByte},
			}, []any{&atomicVolatilityConsiderationWindows[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicVolatilityUpdateThreshold,
				Params: []any{keyByte},
			}, []any{&atomicVolatilityUpdateThresholds[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicExchangeFeeRate,
				Params: []any{keyByte},
			}, []any{&atomicExchangeFeeRates[i]}).
			AddCall(&ethrpc.Call{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodExchangeFeeRate,
				Params: []any{keyByte},
			}, []any{&exchangeFeeRates[i]})
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
		systemSettings.PureChainlinkPriceForAtomicSwapsEnabled[key] = pureChainlinkPriceForAtomicSwapsEnabled[i]
		systemSettings.AtomicEquivalentForDexPricingAddresses[key] = atomicEquivalentForDexPricingAddresses[i]
		systemSettings.AtomicVolatilityConsiderationWindow[key] = atomicVolatilityConsiderationWindows[i]
		systemSettings.AtomicVolatilityUpdateThreshold[key] = atomicVolatilityUpdateThresholds[i]
		systemSettings.AtomicExchangeFeeRate[key] = atomicExchangeFeeRates[i]
		systemSettings.ExchangeFeeRate[key] = exchangeFeeRates[i]
	}

	return nil
}

func (r *SystemSettingsReader) readDynamicFeeConfig(ctx context.Context, address string, systemSettings *SystemSettings) error {
	dynamicFeeConfig := NewDynamicFeeConfig()

	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeDynamicFeeRounds,
			Params: nil,
		}, []any{&dynamicFeeConfig.Rounds}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeDynamicFeeThreshold,
			Params: nil,
		}, []any{&dynamicFeeConfig.Threshold}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeDynamicFeeWeightDecay,
			Params: nil,
		}, []any{&dynamicFeeConfig.WeightDecay}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeMaxDynamicFee,
			Params: nil,
		}, []any{&dynamicFeeConfig.MaxFee})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read dynamic fee config")
		return err
	}

	systemSettings.DynamicFeeConfig = dynamicFeeConfig

	return nil
}

// readData reads data which required no parameters, included:
// - AtomicTwapWindow
// - RateStalePeriod
func (r *SystemSettingsReader) readData(ctx context.Context, address string, systemSettings *SystemSettings) error {
	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodAtomicTwapWindow,
			Params: nil,
		}, []any{&systemSettings.AtomicTwapWindow}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodRateStalePeriod,
			Params: nil,
		}, []any{&systemSettings.RateStalePeriod})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}

	return nil
}
