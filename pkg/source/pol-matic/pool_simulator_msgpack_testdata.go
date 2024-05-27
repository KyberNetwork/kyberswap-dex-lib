package polmatic

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0x29e7df7b6a1b2b07b731457f499e1696c60e2c4e",
			"type": "pol-matic",
			"timestamp": 1705354961,
			"reserves": [
				"22046699825896000703658510",
				"9977954296312119119296341490"
			],
			"tokens": [
				{
					"address": "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0",
					"decimals": 18,
					"swappable": true
				},
				{
					"address": "0x455e53cbb86018ac2b8092fdcd39d8444affc3f6",
					"decimals": 18,
					"swappable": true
				}
			]
		}`,
	}
	var pools []*PoolSimulator
	for _, rawPool := range rawPools {
		poolEntity := new(entity.Pool)
		err := json.Unmarshal([]byte(rawPool), poolEntity)
		if err != nil {
			panic(err)
		}
		pool, err := NewPoolSimulator(*poolEntity)
		if err != nil {
			panic(err)
		}
		pools = append(pools, pool)
	}
	return pools
}
