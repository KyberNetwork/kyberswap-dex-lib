package liquid

type PoolInfo struct {
	SupportedDepositAssets  []string
	SupportedWithdrawAssets []bool // true if the asset at the same index in SupportedDepositAssets can be withdrawn
}

var pools = map[string]PoolInfo{
	// LiquidETH
	"0xf0bb20865277abd641a307ece5ee04e79073416c": {
		SupportedDepositAssets: []string{
			"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			"0x35fa164735182de50811e8e2e824cfb9b6118ac2",
		},
		SupportedWithdrawAssets: []bool{false, true, true},
	},

	// LiquidUSD
	"0x08c6f91e2b681faf5e17227f2a44c307b3c1364c": {
		SupportedDepositAssets: []string{
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		SupportedWithdrawAssets: []bool{true, true},
	},

	// LiquidBTC
	"0x5f46d540b6eD704C3c8789105F30E075AA900726": {
		SupportedDepositAssets: []string{
			"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
			"0x657e8c867d8b37dcc18fa4caead9c45eb088c642",
			"0x8236a87084f8b84306f72007f36f2618a5634494",
		},
		SupportedWithdrawAssets: []bool{false, true, false},
	},
}
