package pancakev3

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	poolEntities := []entity.Pool{
		{
			Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			SwapFee:   2500,
			Exchange:  "pancake-v3",
			Type:      "pancake-v3",
			Timestamp: 1689072352,
			Reserves: entity.PoolReserves{
				"90929743",
				"10999982374483464",
			},
			Tokens: entity.PoolTokens{
				{
					Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
					Name:      "OPENAI",
					Symbol:    "CGPT",
					Decimals:  4,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
					Name:      "Wrapped BNB",
					Symbol:    "WBNB",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
			StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
		},
		{
			Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			SwapFee:   2500,
			Exchange:  "pancake-v3",
			Type:      "pancake-v3",
			Timestamp: 1689072352,
			Reserves: entity.PoolReserves{
				"90929743",
				"10999982374483464",
			},
			Tokens: entity.PoolTokens{
				{
					Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
					Name:      "OPENAI",
					Symbol:    "CGPT",
					Decimals:  4,
					Weight:    50,
					Swappable: true,
				},
				{
					Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
					Name:      "Wrapped BNB",
					Symbol:    "WBNB",
					Decimals:  18,
					Weight:    50,
					Swappable: true,
				},
			},
			Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
			StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
		},
	}
	pools := make([]*PoolSimulator, len(poolEntities))
	var err error
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(poolEntity, valueobject.ChainIDBSC)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
