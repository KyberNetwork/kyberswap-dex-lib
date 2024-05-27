package base

import (
	"fmt"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolBaseSimulator {
	var pools []*PoolBaseSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
				"3000000",    // 0.0003
				"5000000000", // 0.5
				150000, 150000),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
				"100",
				"1000000000000", "1000000000000",
				"1000000000000000000000000000000", "1000000000000000000000000000000"),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)

		now := time.Now().Unix()
		p, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"futureATime\": %v}",
				"3000000",    // 0.0003
				"5000000000", // 0.5
				100000, 200000,
				now*2),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
				"100",
				"1000000000000", "1000000000000",
				"1000000000000000000000000000000", "1000000000000000000000000000000"),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
