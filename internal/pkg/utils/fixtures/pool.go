package fixtures

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

var Pools = []*entity.Pool{
	{
		Address:      "pooladdress1",
		ReserveUsd:   10000,
		AmplifiedTvl: 10,
		SwapFee:      100,
		Exchange:     "exchange1",
		Type:         "type1",
		Timestamp:    1658373335,
		Reserves:     []string{"10000", "20000"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "tokenaddress1",
				Name:      "tokenName1",
				Symbol:    "tokenSymbol1",
				Decimals:  6,
				Weight:    50,
				Swappable: true,
			},
			{
				Address:   "tokenaddress2",
				Name:      "tokenName2",
				Symbol:    "tokenSymbol2",
				Decimals:  6,
				Weight:    50,
				Swappable: true,
			},
		},
		Extra:       "extra1",
		StaticExtra: "staticExtra1",
		TotalSupply: "10000",
	},
	{
		Address:      "poolAddress2",
		ReserveUsd:   20000,
		AmplifiedTvl: 20,
		SwapFee:      200,
		Exchange:     "exchange2",
		Type:         "type2",
		Timestamp:    1658373335,
		Reserves:     []string{"20000", "30000"},
		Tokens: []*entity.PoolToken{
			{
				Address:   "tokenaddress2",
				Name:      "tokenName2",
				Symbol:    "tokenSymbol2",
				Decimals:  6,
				Weight:    50,
				Swappable: true,
			},
			{
				Address:   "tokenaddress3",
				Name:      "tokenName3",
				Symbol:    "tokenSymbol3",
				Decimals:  6,
				Weight:    50,
				Swappable: true,
			},
		},
		Extra:       "extra2",
		StaticExtra: "staticExtra2",
		TotalSupply: "20000",
	},
}
