package wombatlsd

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0x3161f40ea6c0c4cc8b2433d6d530ef255816e854",
			"type": "wombat-lsd",
			"timestamp": 1705357248,
			"reserves": [
				"38310717687612156529",
				"60557257422784379622",
				"31480055744644999606"
			],
			"tokens": [
				{
					"address": "0xac3e018457b222d93114458476f3e3416abbe38f",
					"decimals": 18,
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0x5e8422345238f34275888049021821e8e08caa1f",
					"decimals": 18,
					"weight": 50,
					"swappable": true
				},
				{
					"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					"decimals": 18,
					"weight": 50,
					"swappable": true
				}
			],
			"extra": "{\"paused\":false,\"haircutRate\":100000000000000,\"ampFactor\":2000000000000000,\"startCovRatio\":1500000000000000000,\"endCovRatio\":1800000000000000000,\"assetMap\":{\"0x5e8422345238f34275888049021821e8e08caa1f\":{\"isPause\":false,\"address\":\"0x724515010904518eCF638Cc6d693046B82548068\",\"cash\":60557257422784379622,\"liability\":52162794293656098535,\"underlyingTokenDecimals\":18,\"relativePrice\":1000000000000000000},\"0xac3e018457b222d93114458476f3e3416abbe38f\":{\"isPause\":false,\"address\":\"0x51E073D92b0c226F7B0065909440b18A85769606\",\"cash\":38310717687612156529,\"liability\":34435738368623317194,\"underlyingTokenDecimals\":18,\"relativePrice\":1071887273891919214},\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\":{\"isPause\":false,\"address\":\"0xC096FF2606152eD2A06dd12F15A3c0466Aa5A9fa\",\"cash\":31480055744644999606,\"liability\":43968771821191291731,\"underlyingTokenDecimals\":18,\"relativePrice\":1000000000000000000}}}"
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
	pools := make([]*PoolSimulator, len(poolEntites))
	for i, poolEntity := range poolEntites {
		pools[i], err = NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
