package woofiv2

import (
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	poolEntities := []*entity.Pool{
		{
			Address:  "0xeff23b4be1091b53205e35f3afcd9c7182bf3062",
			Exchange: "woofi-v2",
			Type:     "woofi-v2",
			Reserves: []string{
				"730535084392283753085",
				"3669269999",
				"1186461856050015112014922",
				"314087914252365845476916",
				"358133881118",
				"251046788373",
				"619547557552",
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
					Weight:    1,
					Decimals:  18,
					Swappable: true,
				},
				{
					Address:   "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
					Weight:    1,
					Decimals:  8,
					Swappable: true,
				},
				{
					Address:   "0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b",
					Weight:    1,
					Decimals:  18,
					Swappable: true,
				},
				{
					Address:   "0x912ce59144191c1204e64559fe8253a0e49e6548",
					Weight:    1,
					Decimals:  18,
					Swappable: true,
				},
				{
					Address:   "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
					Weight:    1,
					Decimals:  6,
					Swappable: true,
				},
				{
					Address:   "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
					Weight:    1,
					Decimals:  6,
					Swappable: true,
				},
				{
					Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					Weight:    1,
					Decimals:  6,
					Swappable: true,
				},
			},
			Extra: "{\"quoteToken\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"tokenInfos\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"reserve\":\"0xd8117926\",\"feeRate\":25},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"reserve\":\"0x28074bb87639d60e2d\",\"feeRate\":25},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"reserve\":\"0x4282cb4e062367e7b634\",\"feeRate\":25},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"reserve\":\"0x3a7fc6c611\",\"feeRate\":5},\"0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b\":{\"reserve\":\"0xfb25a057dd40d981fe80\",\"feeRate\":25},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"reserve\":\"0x5362a22d1e\",\"feeRate\":5},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"reserve\":\"0x9087860da4\",\"feeRate\":0}},\"wooracle\":{\"address\":\"0x73504eaCB100c7576146618DC306c97454CB3620\",\"states\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"price\":\"0x3e2922e4c80\",\"spread\":376000000000000,\"coeff\":1340000000,\"woFeasible\":true},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"price\":\"0x346dce843a\",\"spread\":488000000000000,\"coeff\":1000000000,\"woFeasible\":true},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"price\":\"0x7eb9297\",\"spread\":850000000000000,\"coeff\":2620000000,\"woFeasible\":true},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"price\":\"0x5f5fb40\",\"spread\":50000000000000,\"coeff\":1830000000,\"woFeasible\":true},\"0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b\":{\"price\":\"0x29eeb68\",\"spread\":4750000000000000,\"coeff\":135000000000,\"woFeasible\":true},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"price\":\"0x5f5b858\",\"spread\":60000000000000,\"coeff\":1830000000,\"woFeasible\":true},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"price\":\"0x5f5e100\",\"spread\":0,\"coeff\":0,\"woFeasible\":true}},\"decimals\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0x912ce59144191c1204e64559fe8253a0e49e6548\":8,\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":8,\"0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"timestamp\":1703670783,\"staleDuration\":300,\"bound\":10000000000000000},\"cloracle\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"oracleAddress\":\"0x6ce185860a4963106506c203335a2910413708e9\",\"answer\":\"0x3e2ca371620\",\"updatedAt\":\"0x658bf010\",\"cloPreferred\":false},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"oracleAddress\":\"0x639fe6ab55c921f74e7fac1ee960c0b6293ba612\",\"answer\":\"0x346d1a4b1c\",\"updatedAt\":\"0x658bf3ca\",\"cloPreferred\":false},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"oracleAddress\":\"0xb2a824043730fe05f3da2efafa1cbbe83fa548d6\",\"answer\":\"0x7ea9898\",\"updatedAt\":\"0x658bf3ee\",\"cloPreferred\":false},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"oracleAddress\":\"0x50834f3163758fcc1df9973b6e91f0f0f0434ad3\",\"answer\":\"0x5f61567\",\"updatedAt\":\"0x658bcdda\",\"cloPreferred\":false},\"0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b\":{\"oracleAddress\":\"0x0000000000000000000000000000000000000000\",\"answer\":null,\"updatedAt\":null,\"cloPreferred\":false},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"oracleAddress\":\"0x3f3f5df88dc9f13eac63df89ec16ef6e7e25dde7\",\"answer\":\"0x5f5dd18\",\"updatedAt\":\"0x658b91b5\",\"cloPreferred\":false},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"oracleAddress\":\"0x50834f3163758fcc1df9973b6e91f0f0f0434ad3\",\"answer\":\"0x5f61567\",\"updatedAt\":\"0x658bcdda\",\"cloPreferred\":false}}}",
		},
	}
	pools := []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Address:  "poolAddress",
					Exchange: "woofi-v2",
					Type:     "woofi-v2",
					Tokens:   []string{"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"},
				},
			},
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve: number.NewUint256("307599458320800914127"),
					FeeRate: 25,
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve: number.NewUint256("422309249032"),
					FeeRate: 0,
				},
				"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
					Reserve: number.NewUint256("1761585197"),
					FeeRate: 25,
				},
			},
			decimals: map[string]uint8{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 18,
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 6,
				"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": 8,
			},
			wooracle: Wooracle{
				States: map[string]State{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Price:      number.NewUint256("159801975726"),
						Spread:     479000000000000,
						Coeff:      1550000000,
						WoFeasible: true,
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Price:      number.NewUint256("100000000"),
						Spread:     0,
						Coeff:      0,
						WoFeasible: true,
					},
					"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
						Price:      number.NewUint256("2662094951911"),
						Spread:     250000000000000,
						Coeff:      4920000000,
						WoFeasible: true,
					},
				},
				Decimals: map[string]uint8{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 8,
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 8,
					"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": 8,
				},
				Timestamp:     time.Now().Unix(),
				StaleDuration: 300,
				Bound:         10000000000000000,
			},
			gas: DefaultGas,
		},
	}
	for _, poolEntity := range poolEntities {
		pool, err := NewPoolSimulator(*poolEntity)
		if err != nil {
			panic(err)
		}
		pools = append(pools, pool)
	}
	return pools
}
