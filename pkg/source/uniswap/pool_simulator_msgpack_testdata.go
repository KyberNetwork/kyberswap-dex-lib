package uniswap

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852",
			"swapFee": 0.003,
			"type": "uniswap",
			"timestamp": 1705356253,
			"reserves": [
				"32981129686811504138006",
				"83362838693979"
			],
			"tokens": [
				{
					"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"weight": 50,
					"swappable": true
				}
			]
		}`,
	}
	poolEntities := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		err := json.Unmarshal([]byte(rawPool), &poolEntities[i])
		if err != nil {
			panic(err)
		}
	}
	var err error
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}