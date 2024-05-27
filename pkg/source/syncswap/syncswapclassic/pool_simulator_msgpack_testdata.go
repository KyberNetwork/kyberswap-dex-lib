package syncswapclassic

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	poolEntities := []entity.Pool{
		{
			Address:  "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac",
			Exchange: "syncswap",
			Type:     "syncswap-classic",
			Reserves: []string{
				"38819698878426432914729",
				"46113879614283",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x2aa69e007c32cf6637511353b89dce0b473851a9",
					Swappable: true,
				},
				{
					Address:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
					Swappable: true,
				},
			},
			Extra: "{\"swapFee0To1\":200,\"swapFee1To0\":200}",
		},
		{
			Address:  "0x1788f8dec1c2054d653f8330eedcdf3dfbeb42ac",
			Exchange: "syncswap",
			Type:     "syncswap-classic",
			Reserves: []string{
				"38819698878426432914729",
				"46113879614283",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x2aa69e007c32cf6637511353b89dce0b473851a9",
					Swappable: true,
				},
				{
					Address:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
					Swappable: true,
				},
			},
			Extra: "{\"swapFee0To1\":200,\"swapFee1To0\":200}",
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
