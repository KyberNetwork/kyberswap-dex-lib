package liquid

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type PoolInfo struct {
	Exchange string // empty means etherfi-liquid
	// SimpleTeller: teller exposes isSupported(asset) instead of assetData(asset). Deposits go
	// through Depositor (spender) with CommunityCode, as the whlp source does.
	SimpleTeller           bool
	Depositor              string
	Accountant             string
	CommunityCode          string
	SupportedDepositAssets []string
	TellerContract         string
}

var poolsByChain = map[valueobject.ChainID]map[string]PoolInfo{
	valueobject.ChainIDEthereum: ethereumPools,
	valueobject.ChainIDHyperEVM: hyperevmPools,
}

var hyperevmPools = map[string]PoolInfo{
	// LHYPE
	"0x5748ae796ae46a4f1348a1693de4b50560485562": {
		Exchange: valueobject.ExchangeLoopLiquid,
		SupportedDepositAssets: []string{
			"0x5555555555555555555555555555555555555555", // WHYPE
			"0xffaa4a3d97fe9107cef8a3f48c069f577ff76cc1", // stHYPE, pegged 1:1 to the base rate
		},
		TellerContract: "0xfd83c1ca0c04e096d129275126fade1dc45bf4f0",
		SimpleTeller:   true,
		Depositor:      "0x6e358dd1204c3fb1d24e569df0899f48fabe5337",
		Accountant:     "0xce621a3ca6f72706678cff0572ae8d15e5f001c3",
		CommunityCode:  "kyberswap",
	},
}

var ethereumPools = map[string]PoolInfo{
	// LiquidETH
	"0xf0bb20865277abd641a307ece5ee04e79073416c": {
		SupportedDepositAssets: []string{
			"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			"0x35fa164735182de50811e8e2e824cfb9b6118ac2",
		},
		TellerContract: "0x9aa79c84b79816ab920bbce20f8f74557b514734",
	},

	// LiquidUSD
	"0x08c6f91e2b681faf5e17227f2a44c307b3c1364c": {
		SupportedDepositAssets: []string{
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		TellerContract: "0x4de413a26fc24c3fc27cc983be70aa9c5c299387",
	},

	// LiquidBTC
	"0x5f46d540b6eD704C3c8789105F30E075AA900726": {
		SupportedDepositAssets: []string{
			"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			"0x657e8c867d8b37dcc18fa4caead9c45eb088c642",
			"0x8236a87084f8b84306f72007f36f2618a5634494",
		},
		TellerContract: "0x8ea0b382d054dbebeb1d0ae47ee4ac433c730353",
	},
}
