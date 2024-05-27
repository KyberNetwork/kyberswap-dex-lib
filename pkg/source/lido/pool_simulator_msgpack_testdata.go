package lido

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"2264571555224494676557305", "2005870067403083354670050"},
			Tokens:   []*entity.PoolToken{{Address: "stETH"}, {Address: "wstETH"}},
			Extra: fmt.Sprintf("{\"stEthPerToken\": %v, \"tokensPerStEth\": %v}",
				"1128972205632615487",
				"885761398740240572"),
			StaticExtra: "{\"lpToken\": \"wstETH\"}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
