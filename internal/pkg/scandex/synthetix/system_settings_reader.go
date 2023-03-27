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

const (
	// SystemSettings methods

	SystemSettingsMethodPureChainlinkPriceForAtomicSwapsEnabled = "pureChainlinkPriceForAtomicSwapsEnabled"
	SystemSettingsMethodAtomicEquivalentForDexPricing           = "atomicEquivalentForDexPricing"
	SystemSettingsMethodAtomicTwapWindow                        = "atomicTwapWindow"
	SystemSettingsMethodAtomicVolatilityConsiderationWindow     = "atomicVolatilityConsiderationWindow"
	SystemSettingsMethodAtomicVolatilityUpdateThreshold         = "atomicVolatilityUpdateThreshold"
	SystemSettingsMethodAtomicExchangeFeeRate                   = "atomicExchangeFeeRate"
	SystemSettingsMethodExchangeFeeRate                         = "exchangeFeeRate"
	SystemSettingsMethodRateStalePeriod                         = "rateStalePeriod"
	SystemSettingsMethodExchangeDynamicFeeRounds                = "exchangeDynamicFeeRounds"
	SystemSettingsMethodExchangeDynamicFeeThreshold             = "exchangeDynamicFeeThreshold"
	SystemSettingsMethodExchangeDynamicFeeWeightDecay           = "exchangeDynamicFeeWeightDecay"
	SystemSettingsMethodExchangeMaxDynamicFee                   = "exchangeMaxDynamicFee"

	// Token methods

	TokenMethodDecimals = "decimals"
	TokenMethodSymbol   = "symbol"
)

type SystemSettingsReader struct {
	abi         abi.ABI
	scanService *service.ScanService
}

func NewSystemSettingsReader(scanService *service.ScanService) *SystemSettingsReader {
	return &SystemSettingsReader{
		abi:         abis.SynthetixSystemSettings,
		scanService: scanService,
	}
}

func (r *SystemSettingsReader) Read(
	ctx context.Context,
	poolState *PoolState,
) (*SystemSettings, error) {
	systemSettings := NewSystemSettings()

	if err := r.readData(ctx, poolState.Addresses.SystemSettings, systemSettings); err != nil {
		return nil, err
	}

	if err := r.readDynamicFeeConfig(ctx, poolState.Addresses.SystemSettings, systemSettings); err != nil {
		return nil, err
	}

	if err := r.readCurrencyKeyData(ctx, poolState.Addresses.SystemSettings, systemSettings, poolState.CurrencyKeys); err != nil {
		return nil, err
	}

	if err := r.readTokenData(ctx, systemSettings, poolState.CurrencyKeys); err != nil {
		return nil, err
	}

	return systemSettings, nil
}

// readData reads data which required no parameters, included:
// - AtomicTwapWindow
// - RateStalePeriod
func (r *SystemSettingsReader) readData(
	ctx context.Context,
	address string,
	systemSettings *SystemSettings,
) error {
	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodAtomicTwapWindow,
			Params: nil,
			Output: &systemSettings.AtomicTwapWindow,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodRateStalePeriod,
			Params: nil,
			Output: &systemSettings.RateStalePeriod,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	return nil
}

func (r *SystemSettingsReader) readDynamicFeeConfig(
	ctx context.Context,
	address string,
	systemSettings *SystemSettings,
) error {
	dynamicFeeConfig := NewDynamicFeeConfig()

	calls := []*repository.CallParams{
		{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeDynamicFeeRounds,
			Params: nil,
			Output: &dynamicFeeConfig.Rounds,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeDynamicFeeThreshold,
			Params: nil,
			Output: &dynamicFeeConfig.Threshold,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeDynamicFeeWeightDecay,
			Params: nil,
			Output: &dynamicFeeConfig.WeightDecay,
		},
		{
			ABI:    r.abi,
			Target: address,
			Method: SystemSettingsMethodExchangeMaxDynamicFee,
			Params: nil,
			Output: &dynamicFeeConfig.MaxFee,
		},
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
		return err
	}

	systemSettings.DynamicFeeConfig = dynamicFeeConfig

	return nil
}

// readCurrencyKeyData reads data which required currency key as parameter, included:
// - PureChainlinkPriceForAtomicSwapsEnabled
// - AtomicEquivalentForDexPricingAddresses
// - AtomicVolatilityConsiderationWindow
// - AtomicVolatilityUpdateThreshold
// - AtomicExchangeFeeRate
// - ExchangeFeeRate
func (r *SystemSettingsReader) readCurrencyKeyData(
	ctx context.Context,
	address string,
	systemSettings *SystemSettings,
	currencyKeys []string,
) error {
	currencyKeysLen := len(currencyKeys)

	pureChainlinkPriceForAtomicSwapsEnabled := make([]bool, currencyKeysLen)
	atomicEquivalentForDexPricingAddresses := make([]common.Address, currencyKeysLen)
	atomicVolatilityConsiderationWindows := make([]*big.Int, currencyKeysLen)
	atomicVolatilityUpdateThresholds := make([]*big.Int, currencyKeysLen)
	atomicExchangeFeeRates := make([]*big.Int, currencyKeysLen)
	exchangeFeeRates := make([]*big.Int, currencyKeysLen)

	var calls []*repository.CallParams
	for i, key := range currencyKeys {
		keyByte := eth.StringToBytes32(key)

		tokenCalls := []*repository.CallParams{
			{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodPureChainlinkPriceForAtomicSwapsEnabled,
				Params: []interface{}{keyByte},
				Output: &pureChainlinkPriceForAtomicSwapsEnabled[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicEquivalentForDexPricing,
				Params: []interface{}{keyByte},
				Output: &atomicEquivalentForDexPricingAddresses[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicVolatilityConsiderationWindow,
				Params: []interface{}{keyByte},
				Output: &atomicVolatilityConsiderationWindows[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicVolatilityUpdateThreshold,
				Params: []interface{}{keyByte},
				Output: &atomicVolatilityUpdateThresholds[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodAtomicExchangeFeeRate,
				Params: []interface{}{keyByte},
				Output: &atomicExchangeFeeRates[i],
			},
			{
				ABI:    r.abi,
				Target: address,
				Method: SystemSettingsMethodExchangeFeeRate,
				Params: []interface{}{keyByte},
				Output: &exchangeFeeRates[i],
			},
		}

		calls = append(calls, tokenCalls...)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
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

// readTokenData reads token data, included:
// - Decimals
// - Symbol
func (r *SystemSettingsReader) readTokenData(
	ctx context.Context,
	systemSettings *SystemSettings,
	currencyKeys []string,
) error {
	atomicEquivalentForDexPricingAddresses := systemSettings.AtomicEquivalentForDexPricingAddresses
	tokensLen := len(atomicEquivalentForDexPricingAddresses)

	decimals := make([]uint8, tokensLen)
	symbols := make([]string, tokensLen)

	var calls []*repository.CallParams
	for i, currencyKey := range currencyKeys {
		address := atomicEquivalentForDexPricingAddresses[currencyKey]
		if eth.IsZeroAddress(address) {
			continue
		}

		tokenCalls := []*repository.CallParams{
			{
				ABI:    abis.ERC20,
				Target: address.String(),
				Method: TokenMethodDecimals,
				Params: nil,
				Output: &decimals[i],
			},
			{
				ABI:    abis.ERC20,
				Target: address.String(),
				Method: TokenMethodSymbol,
				Params: nil,
				Output: &symbols[i],
			},
		}

		calls = append(calls, tokenCalls...)
	}

	if err := r.scanService.MultiCall(ctx, calls); err != nil {
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
