package equalizer

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0xf3f1f5760a614b8146eec5d1c94658720c2425b9",
			"swapFee": 0.002666666666666667,
			"type": "equalizer",
			"timestamp": 1705345162,
			"reserves": [
				"173810100394741222630",
				"441959784673"
			],
			"tokens": [
				{
					"address": "0x4200000000000000000000000000000000000006",
					"decimals": 18,
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
					"decimals": 6,
					"weight": 50,
					"swappable": true
				}
			],
			"staticExtra": "{\"stable\":false}"
		}`,
	}
	poolEntites := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		err := json.Unmarshal([]byte(rawPool), &poolEntites[i])
		if err != nil {
			panic(err)
		}
	}
	var err error
	pools := make([]*PoolSimulator, len(rawPools))
	for i, poolEntity := range poolEntites {
		pools[i], err = NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
