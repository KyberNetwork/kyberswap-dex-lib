package plainoracle

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*Pool {
	var pools []*Pool
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"4929038393526761949570", "4622174777771844922336", "9849021650836480441313"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"rates\": [%v, %v]}",
				"4000000",
				"5000000000",
				5000, 5000,
				"1000000000000000000", "1128972205632615487"),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"oracle\": \"%v\"}",
				"100",
				"1", "1",
				"0xe59EBa0D492cA53C6f46015EEa00517F2707dc77"),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
