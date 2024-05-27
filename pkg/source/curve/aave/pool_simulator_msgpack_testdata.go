package aave

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*AavePool {
	var pools []*AavePool
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"8374598852113385564139023", "8328286891683", "5035549096857"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra: fmt.Sprintf("{\"offpegFeeMultiplier\": \"%v\", \"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
				"20000000000",
				"4000000",
				"5000000000",
				20000, 200000),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"precisionMultipliers\": [\"%v\", \"%v\", \"%v\"], \"underlyingTokens\": [\"%v\", \"%v\", \"%v\"]}",
				"1", "1000000000000", "1000000000000",
				"Au", "Bu", "Cu"),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
