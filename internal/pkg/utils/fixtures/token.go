package fixtures

import "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"

var Tokens = []entity.Token{
	{
		Address:     "tokenaddress1",
		Symbol:      "symbol1",
		Name:        "name1",
		Decimals:    6,
		CgkID:       "ckgid1",
		Type:        "type1",
		PoolAddress: "",
	},
	{
		Address:     "tokenaddress2",
		Symbol:      "symbol2",
		Name:        "name2",
		Decimals:    6,
		CgkID:       "ckgid2",
		Type:        "type2",
		PoolAddress: "",
	},
	{
		Address:     "tokenaddress3",
		Symbol:      "symbol3",
		Name:        "name3",
		Decimals:    6,
		CgkID:       "ckgid3",
		Type:        "type3",
		PoolAddress: "",
	},
	{
		Address:     "tokenaddress4",
		Symbol:      "symbol4",
		Name:        "name4",
		Decimals:    6,
		CgkID:       "ckgid4",
		Type:        "type4",
		PoolAddress: "pooladdress1",
	},
}
