package stable

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "balancer-v3-stable"

	SubgraphPoolType = "STABLE"

	poolMethodGetAmplificationParameter = "getAmplificationParameter"

	stableSurgeHookMethodGetMaxSurgeFeePercentage    = "getMaxSurgeFeePercentage"
	stableSurgeHookMethodGetSurgeThresholdPercentage = "getSurgeThresholdPercentage"

	baseGas = 237494
)

var (
	// AcceptableMaxSurgeFeePercentage caps max acceptable surge fee to avoid high slippage
	AcceptableMaxSurgeFeePercentage = uint256.NewInt(0.1e18) // 10%
	// AcceptableMaxSurgeFeeByImbalance caps max acceptable surge fee per imbalance to avoid high slippage
	AcceptableMaxSurgeFeeByImbalance = uint256.NewInt(0.1e18) // 0.1% per 1% of imbalance

	nonNativesByChain = map[valueobject.ChainID]map[string]bool{
		valueobject.ChainIDArbitrumOne: {
			"0xaf88d065e77c8cc2239327c5edb3a432268e5831": true, // USDC
			"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": true, // USDC.e
			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f": true, // WBTC
			"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf": true, // cbBTC
		},
		valueobject.ChainIDAvalancheCChain: {
			"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e": true, // USDC
			"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664": true, // USDC.e
			"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7": true, // USDt
			"0xc7198437980c041c805a1edcba50c1ce5db95118": true, // USDT.e
			"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab": true, // WETH.e
			"0x152b9d0fdc40c096757f570a51e494bd4b943e50": true, // BTC.b
			"0x50b7545627a5162f82a992c33b87adc75187b218": true, // WBTC.e
		},
		valueobject.ChainIDBase: {
			"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": true, // USDC
			"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca": true, // USDbC
			"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf": true, // cbBTC
			"0x0555e30da8f98308edb960aa94c0db47230d2b9c": true, // WBTC
		},
		valueobject.ChainIDHyperEVM: {
			"0xb88339cb7199b77e23db6e890353e22632ba630f": true, // USDC
			"0xb8ce59fc3717ada4c02eadf9682a9e934f625ebb": true, // USD₮0
			"0x111111a1a0667d36bd57c0a9f569b98057111111": true, // USDH
			"0xbe6727b535545c67d5caa73dea54865b92cf7907": true, // UETH
		},
		valueobject.ChainIDSonic: {
			"0x29219dd400f2bf60e5a23d13be72b486d4038894": true, // USDC
		},
	}
)
