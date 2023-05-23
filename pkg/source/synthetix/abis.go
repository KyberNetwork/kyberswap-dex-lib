package synthetix

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	synthetix                       abi.ABI
	systemSettings                  abi.ABI
	exchanger                       abi.ABI
	exchangerWithFeeRecAlternatives abi.ABI
	exchangeRates                   abi.ABI
	exchangeRatesWithDexPricing     abi.ABI
	chainlinkDataFeed               abi.ABI
	dexPriceAggregatorUniswapV3     abi.ABI
	multiCollateralSynth            abi.ABI
	erc20                           abi.ABI
	uniswapV3Pool                   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&synthetix, synthetixBytes},
		{&systemSettings, systemSettingsBytes},
		{&exchanger, exchangerBytes},
		{&exchangerWithFeeRecAlternatives, exchangerWithFeeRecAlternativesBytes},
		{&exchangeRates, exchangeRatesBytes},
		{&exchangeRatesWithDexPricing, exchangeRatesWithDexPricingBytes},
		{&chainlinkDataFeed, chainlinkDataFeedBytes},
		{&dexPriceAggregatorUniswapV3, dexPriceAggregatorUniswapV3Bytes},
		{&multiCollateralSynth, multiCollateralSynthBytes},
		{&erc20, erc20Bytes},
		{&uniswapV3Pool, uniswapv3PoolBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
