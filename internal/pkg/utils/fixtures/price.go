package fixtures

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

var Prices = []*entity.Price{
	{
		Address:     "tokenaddress1",
		Price:       100000,
		Liquidity:   10000,
		LpAddress:   "lpaddress1",
		MarketPrice: 100000,
	},
	{
		Address:     "tokenaddress2",
		Price:       200000,
		Liquidity:   20000,
		LpAddress:   "lpaddress2",
		MarketPrice: 200000,
	},
	{
		Address:     "tokenaddress3",
		Price:       300000,
		Liquidity:   30000,
		LpAddress:   "lpaddress3",
		MarketPrice: 300000,
	},
	{
		Address:     "tokenaddress4",
		Price:       400000,
		Liquidity:   40000,
		LpAddress:   "lpaddress4",
		MarketPrice: 400000,
	},
}
