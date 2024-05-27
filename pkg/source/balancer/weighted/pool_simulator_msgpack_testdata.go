package balancerweighted

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*WeightedPool2Tokens {
	poolEntities := []*entity.Pool{
		{
			Address:  "adr",
			SwapFee:  0.0025,
			Reserves: []string{"5000000", "7000"},
			Tokens: entity.PoolTokens{
				&entity.PoolToken{Address: "BAL", Weight: 80},
				&entity.PoolToken{Address: "WETH", Weight: 20},
			},
			StaticExtra: "{\"vaultAddress\":\"v1\",\"poolId\":\"p1\",\"tokenDecimals\":[1,19]}",
		},
		{
			Address:  "adr",
			SwapFee:  0.0025,
			Reserves: []string{"5000000", "7000", "300000"},
			Tokens: entity.PoolTokens{
				&entity.PoolToken{Address: "BAL", Weight: 40},
				&entity.PoolToken{Address: "WETH", Weight: 10},
				&entity.PoolToken{Address: "DAI", Weight: 50},
			},
			StaticExtra: "{\"vaultAddress\":\"v1\",\"poolId\":\"p1\",\"tokenDecimals\":[1,19,1]}",
		},
	}
	var err error
	pools := make([]*WeightedPool2Tokens, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
