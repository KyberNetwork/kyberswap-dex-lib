package nativev1

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	poolEntities := []entity.Pool{
		{
			Address:  "native_v1_0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270_0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
			Exchange: "native-v1",
			Type:     "native-v1",
			Reserves: []string{"181716_295903804_000000000", "8_489139"},
			Tokens: []*entity.PoolToken{
				{Address: "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", Decimals: 18, Swappable: true},
				{Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Decimals: 6, Swappable: true},
			},
			Extra: "{\"0to1\":[{\"q\":0.0001,\"p\":0.91245042136692},{\"q\":4.659919497201971,\"p\":0.91245042136692}," +
				"{\"q\":4.66001949720197,\"p\":0.90924546691228}],\"min0\":0.0001," +
				"\"1to0\":[{\"q\":0.0001,\"p\":1.0942398729806944},{\"q\":18277.528075741084,\"p\":1.0942398729806944}," +
				"{\"q\":25244.263002363805,\"p\":1.0939119116852096},{\"q\":32092.9359692824,\"p\":1.0937921053280593}," +
				"{\"q\":33219.273417201824,\"p\":1.0936723252106664},{\"q\":29917.17166407224,\"p\":1.0935525713244107}," +
				"{\"q\":27391.98476499627,\"p\":1.093432843660677}],\"min1\":0.0001,\"tlrnce\":0}",
		},
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
