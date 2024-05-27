package velodromev2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			SwapFee:     0.0005, // from factory https://optimistic.etherscan.io/address/0x25cbddb98b35ab1ff77413456b31ec81a6b6b746#readContract
			Reserves:    entity.PoolReserves{"2082415614000308399878", "3631620514949"},
			Tokens:      []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 6}},
			StaticExtra: "{\"stable\": false}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
