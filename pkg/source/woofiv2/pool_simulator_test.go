package woofiv2_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/woofiv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPoolSimulatorCalcAmountOut(t *testing.T) {
	woofiv2Pool, err := woofiv2.NewPoolSimulator(entity.Pool{
		Address:  "0xeff23b4be1091b53205e35f3afcd9c7182bf3062",
		Exchange: "woofi-v2",
		Type:     "woofi-v2",
		Reserves: entity.PoolReserves{"244827033350648711018",
			"1311635660",
			"365949251847504077609749",
			"262911609491773128733494",
			"162656868450",
			"142357215206",
			"560370768868",
		},
		Tokens: []*entity.PoolToken{
			{
				Address:  "0x912ce59144191c1204e64559fe8253a0e49e6548",
				Decimals: 18,
			},
			{
				Address:  "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
				Decimals: 6,
			},
		},
		Extra: "{\"quoteToken\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"unclaimedFee\":207299752,\"wooracle\":\"0x73504eaCB100c7576146618DC306c97454CB3620\",\"tokenInfos\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"reserve\":1311635660,\"feeRate\":25,\"decimals\":8,\"state\":{\"price\":3433920000000,\"spread\":250000000000000,\"coeff\":2380000000,\"woFeasible\":true,\"decimals\":8}},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"reserve\":244827033350648711018,\"feeRate\":25,\"decimals\":18,\"state\":{\"price\":178714058099,\"spread\":250000000000000,\"coeff\":2000000000,\"woFeasible\":true,\"decimals\":8}},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"reserve\":262911609491773128733494,\"feeRate\":25,\"decimals\":18,\"state\":{\"price\":93859000,\"spread\":774000000000000,\"coeff\":3510000000,\"woFeasible\":true,\"decimals\":8}},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"reserve\":142357215206,\"feeRate\":5,\"decimals\":6,\"state\":{\"price\":99998984,\"spread\":50000000000000,\"coeff\":2170000000,\"woFeasible\":true,\"decimals\":8}},\"0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b\":{\"reserve\":365949251847504077609749,\"feeRate\":25,\"decimals\":18,\"state\":{\"price\":22624500,\"spread\":2750000000000000,\"coeff\":156000000000,\"woFeasible\":true,\"decimals\":8}},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"reserve\":162656868450,\"feeRate\":5,\"decimals\":6,\"state\":{\"price\":100025899,\"spread\":100000000000000,\"coeff\":1750000000,\"woFeasible\":true,\"decimals\":8}},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"reserve\":560370768868,\"feeRate\":0,\"decimals\":6,\"state\":{\"price\":100000000,\"spread\":0,\"coeff\":0,\"woFeasible\":true,\"decimals\":8}}}}",
	})

	assert.Nil(t, err)

	result, err := woofiv2Pool.CalcAmountOut(
		pool.TokenAmount{
			Token:  "0x912ce59144191c1204e64559fe8253a0e49e6548",
			Amount: bignumber.NewBig10("100000000000000000000000"),
		},
		"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
	)

	assert.Nil(t, err)
	assert.Equal(t, "93692366654", result.TokenAmountOut.Amount.String())
}
