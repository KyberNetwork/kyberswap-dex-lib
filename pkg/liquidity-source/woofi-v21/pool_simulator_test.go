package woofiv21

import (
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolSimulator_NewPool(t *testing.T) {
	entityPool := entity.Pool{
		Address:  "0xed9e3f98bbed560e66b89aac922e29d4596a9642",
		Exchange: string(valueobject.ExchangeWooFiV3),
		Type:     DexTypeWooFiV21,
		Reserves: []string{
			"577793740802601114533",
			"6771775566",
			"463802387551",
			"769291530352546734805110",
			"823230905302",
			"891772605638",
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
				Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				Weight:    1,
				Decimals:  6,
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
		},
		Extra:       "{\"quoteToken\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"tokenInfos\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"reserve\":\"6771775566\",\"feeRate\":25,\"maxGamma\":\"3000000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"reserve\":\"577793740802601114533\",\"feeRate\":25,\"maxGamma\":\"3000000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"reserve\":\"769291530352546734805110\",\"feeRate\":25,\"maxGamma\":\"5000000000000000\",\"maxNotionalSwap\":\"500000000000\"},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"reserve\":\"891772605638\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"reserve\":\"823230905302\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"reserve\":\"463802387551\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"}},\"wooracle\":{\"address\":\"0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec\",\"states\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"price\":\"5524710000000\",\"spread\":904000000000000,\"coeff\":1000000000,\"woFeasible\":true},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"price\":\"232859000000\",\"spread\":912000000000000,\"coeff\":1000000000,\"woFeasible\":true},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"price\":\"51806000\",\"spread\":1750000000000000,\"coeff\":7400000000,\"woFeasible\":true},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"price\":\"0\",\"spread\":0,\"coeff\":0,\"woFeasible\":false},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"price\":\"100002000\",\"spread\":450000000000000,\"coeff\":388000000,\"woFeasible\":true},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"price\":\"100001913\",\"spread\":50000000000000,\"coeff\":496000000,\"woFeasible\":true}},\"decimals\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0x912ce59144191c1204e64559fe8253a0e49e6548\":8,\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"timestamp\":1725874621,\"staleDuration\":120,\"bound\":25000000000000000},\"cloracle\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"oracleAddress\":\"0xd0c7101eacbb49f3decccc166d238410d6d46d57\",\"answer\":\"5531266217000\",\"updatedAt\":\"1725873667\",\"cloPreferred\":false},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"oracleAddress\":\"0x639fe6ab55c921f74e7fac1ee960c0b6293ba612\",\"answer\":\"232828500000\",\"updatedAt\":\"1725874681\",\"cloPreferred\":false},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"oracleAddress\":\"0xb2a824043730fe05f3da2efafa1cbbe83fa548d6\",\"answer\":\"51754572\",\"updatedAt\":\"1725874678\",\"cloPreferred\":false},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"oracleAddress\":\"0x50834f3163758fcc1df9973b6e91f0f0f0434ad3\",\"answer\":\"100002000\",\"updatedAt\":\"1725833227\",\"cloPreferred\":false},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"oracleAddress\":\"0x3f3f5df88dc9f13eac63df89ec16ef6e7e25dde7\",\"answer\":\"99984372\",\"updatedAt\":\"1725867640\",\"cloPreferred\":false},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"oracleAddress\":\"0x50834f3163758fcc1df9973b6e91f0f0f0434ad3\",\"answer\":\"100002000\",\"updatedAt\":\"1725833227\",\"cloPreferred\":false}}}",
		BlockNumber: 0,
	}

	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			Amount: bignumber.NewBig10("100000000000000000000"),
		},
		TokenOut: "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
	}

	pool, err := NewPoolSimulator(entityPool)
	require.Nil(t, err)

	pool.wooracle.Timestamp = time.Now().Unix()
	pool.wooracle.StaleDuration = 300
	pool.wooracle.Bound = 10000000000000000

	result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
		return pool.CalcAmountOut(params)
	})

	require.Nil(t, err)
	require.Equal(t, "420800752", result.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_Nil_Oracle(t *testing.T) {
	entityPool := entity.Pool{
		Address:  "0xed9e3f98bbed560e66b89aac922e29d4596a9642",
		Exchange: string(valueobject.ExchangeWooFiV3),
		Type:     DexTypeWooFiV21,
		Reserves: []string{
			"577793740802601114533",
			"6771775566",
			"463802387551",
			"769291530352546734805110",
			"823230905302",
			"891772605638",
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
				Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				Weight:    1,
				Decimals:  6,
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
		},
		Extra:       "{\"quoteToken\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"tokenInfos\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"reserve\":\"6771775566\",\"feeRate\":25,\"maxGamma\":\"3000000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"reserve\":\"577793740802601114533\",\"feeRate\":25,\"maxGamma\":\"3000000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"reserve\":\"769291530352546734805110\",\"feeRate\":25,\"maxGamma\":\"5000000000000000\",\"maxNotionalSwap\":\"500000000000\"},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"reserve\":\"891772605638\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"reserve\":\"823230905302\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"reserve\":\"463802387551\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"}},\"wooracle\":{\"address\":\"0xCf4EA1688bc23DD93D933edA535F8B72FC8934Ec\",\"states\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"price\":\"5524710000000\",\"spread\":904000000000000,\"coeff\":1000000000,\"woFeasible\":true},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"price\":\"232859000000\",\"spread\":912000000000000,\"coeff\":1000000000,\"woFeasible\":true},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"price\":\"51806000\",\"spread\":1750000000000000,\"coeff\":7400000000,\"woFeasible\":true},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"price\":\"0\",\"spread\":0,\"coeff\":0,\"woFeasible\":false},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"price\":\"100002000\",\"spread\":450000000000000,\"coeff\":388000000,\"woFeasible\":true},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"price\":\"100001913\",\"spread\":50000000000000,\"coeff\":496000000,\"woFeasible\":true}},\"decimals\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0x912ce59144191c1204e64559fe8253a0e49e6548\":8,\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"timestamp\":1725874621,\"staleDuration\":120,\"bound\":25000000000000000},\"cloracle\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"oracleAddress\":\"0xd0c7101eacbb49f3decccc166d238410d6d46d57\",\"answer\":\"5531266217000\",\"updatedAt\":\"1725873667\",\"cloPreferred\":false},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"oracleAddress\":\"0x0000000000000000000000000000000000000000\",\"answer\":\"232828500000\",\"updatedAt\":\"1725874681\",\"cloPreferred\":false},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"oracleAddress\":\"0xb2a824043730fe05f3da2efafa1cbbe83fa548d6\",\"answer\":\"51754572\",\"updatedAt\":\"1725874678\",\"cloPreferred\":false},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"oracleAddress\":\"0x50834f3163758fcc1df9973b6e91f0f0f0434ad3\",\"answer\":\"100002000\",\"updatedAt\":\"1725833227\",\"cloPreferred\":false},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"oracleAddress\":\"0x3f3f5df88dc9f13eac63df89ec16ef6e7e25dde7\",\"answer\":\"99984372\",\"updatedAt\":\"1725867640\",\"cloPreferred\":false},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"oracleAddress\":\"0x50834f3163758fcc1df9973b6e91f0f0f0434ad3\",\"answer\":\"100002000\",\"updatedAt\":\"1725833227\",\"cloPreferred\":false}}}",
		BlockNumber: 0,
	}

	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			Amount: bignumber.NewBig10("100000000000000000000"),
		},
		TokenOut: "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
	}

	pool, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	pool.wooracle.Timestamp = time.Now().Unix()

	result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
		return pool.CalcAmountOut(params)
	})

	assert.Nil(t, err)
	assert.Equal(t, "420800752", result.TokenAmountOut.Amount.String())
}

// func TestPoolSimulator_CalcAmountOut_Arithmetic_OverflowUnderflow(t *testing.T) {
// 	entityPool := entity.Pool{
// 		Address:  "0xd1778f9df3eee5473a9640f13682e3846f61febc",
// 		Exchange: string(valueobject.ExchangeWooFiV3),
// 		Type:     DexTypeWooFiV21,
// 		Reserves: []string{
// 			"301370617381821852207",
// 			"785512143",
// 			"177053835630",
// 			"97558688283555321324212",
// 			"167081703216",
// 			"152515901952",
// 		},
// 		Tokens: []*entity.PoolToken{
// 			{
// 				Address:   "0x4200000000000000000000000000000000000006",
// 				Weight:    1,
// 				Decimals:  18,
// 				Swappable: true,
// 			},
// 			{
// 				Address:   "0x68f180fcce6836688e9084f035309e29bf0a2095",
// 				Weight:    1,
// 				Decimals:  8,
// 				Swappable: true,
// 			},
// 			{
// 				Address:   "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
// 				Weight:    1,
// 				Decimals:  6,
// 				Swappable: true,
// 			},
// 			{
// 				Address:   "0x4200000000000000000000000000000000000042",
// 				Weight:    1,
// 				Decimals:  18,
// 				Swappable: true,
// 			},
// 			{
// 				Address:   "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
// 				Weight:    1,
// 				Decimals:  6,
// 				Swappable: true,
// 			},
// 			{
// 				Address:   "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
// 				Weight:    1,
// 				Decimals:  6,
// 				Swappable: true,
// 			},
// 		},
// 		Extra: "{\"quoteToken\":\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\",\"tokenInfos\":{\"0x0b2c639c533813f4aa9d7837caf62653d097ff85\":{\"reserve\":\"0x29393b216e\",\"feeRate\":5},\"0x4200000000000000000000000000000000000006\":{\"reserve\":\"0x10565b83c75fa7aa2f\",\"feeRate\":25},\"0x4200000000000000000000000000000000000042\":{\"reserve\":\"0x14a8aac659cf6a43a2b4\",\"feeRate\":25},\"0x68f180fcce6836688e9084f035309e29bf0a2095\":{\"reserve\":\"0x2ed1f6cf\",\"feeRate\":25},\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\":{\"reserve\":\"0x2382a7fa00\",\"feeRate\":0},\"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58\":{\"reserve\":\"0x26e6d87730\",\"feeRate\":5}},\"wooracle\":{\"address\":\"0xd589484d3A27B7Ce5C2C7F829EB2e1D163f95817\",\"states\":{\"0x0b2c639c533813f4aa9d7837caf62653d097ff85\":{\"price\":\"0x5f5640d\",\"spread\":50000000000000,\"coeff\":3940000000,\"woFeasible\":true},\"0x4200000000000000000000000000000000000006\":{\"price\":\"0x34d8869cc0\",\"spread\":366000000000000,\"coeff\":2260000000,\"woFeasible\":true},\"0x4200000000000000000000000000000000000042\":{\"price\":\"0xf3671b0\",\"spread\":1570000000000000,\"coeff\":3570000000,\"woFeasible\":true},\"0x68f180fcce6836688e9084f035309e29bf0a2095\":{\"price\":\"0x4030c6ec900\",\"spread\":427000000000000,\"coeff\":3950000000,\"woFeasible\":true},\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\":{\"price\":\"0x5f5e100\",\"spread\":0,\"coeff\":0,\"woFeasible\":true},\"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58\":{\"price\":\"0x5f5b671\",\"spread\":101000000000000,\"coeff\":3960000000,\"woFeasible\":true}},\"decimals\":{\"0x0b2c639c533813f4aa9d7837caf62653d097ff85\":8,\"0x4200000000000000000000000000000000000006\":8,\"0x4200000000000000000000000000000000000042\":8,\"0x68f180fcce6836688e9084f035309e29bf0a2095\":8,\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\":8,\"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58\":8}}}",
// 	}
// 	params := poolpkg.CalcAmountOutParams{
// 		TokenAmountIn: poolpkg.TokenAmount{
// 			Token:  "0x4200000000000000000000000000000000000006",
// 			Amount: bignumber.NewBig10("1000000000000000000000"),
// 		},
// 		TokenOut: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
// 	}

// 	pool, err := NewPoolSimulator(entityPool)
// 	assert.Nil(t, err)

// 	pool.wooracle.Timestamp = time.Now().Unix()
// 	pool.wooracle.StaleDuration = 300
// 	pool.wooracle.Bound = 10000000000000000

// 	_, err = testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
// 		return pool.CalcAmountOut(params)
// 	})

// 	assert.Equal(t, ErrArithmeticOverflowUnderflow, err)
// }

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name           string
		quoteToken     string
		tokenInfos     map[string]TokenInfo
		decimals       map[string]uint8
		wooracle       Wooracle
		params         poolpkg.CalcAmountOutParams
		expectedErr    error
		expectedResult *poolpkg.CalcAmountOutResult
	}{
		{
			name:       "test _sellBase",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve:         number.NewUint256("305740102740733506649"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve:         number.NewUint256("403770676421"),
					FeeRate:         0,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
			},
			decimals: map[string]uint8{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 18,
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 8,
			},
			wooracle: Wooracle{
				States: map[string]State{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Price:      number.NewUint256("159709047746"),
						Spread:     270000000000000,
						Coeff:      1550000000,
						WoFeasible: true,
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Price:      number.NewUint256("100000000"),
						Spread:     0,
						Coeff:      0,
						WoFeasible: true,
					},
				},
				Decimals: map[string]uint8{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 8,
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 8,
				},
				Timestamp:     time.Now().Unix(),
				StaleDuration: 300,
				Bound:         10000000000000000,
			},
			params: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
					Amount: bignumber.NewBig("304999404452284472"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expectedErr: nil,
			expectedResult: &poolpkg.CalcAmountOutResult{
				TokenAmountOut: &poolpkg.TokenAmount{
					Token:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					Amount: bignumber.NewBig10("48685801162"),
				},
				Fee: &poolpkg.TokenAmount{
					Token:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
					Amount: bignumber.NewBig10("12174493"),
				},
				Gas: DefaultGas.Swap,
				SwapInfo: &woofiV2SwapInfo{
					newPrice:           number.NewUint256("159708927161"),
					newMaxNotionalSwap: number.NewUint256("951288835552"),
					newMaxGamma:        number.NewUint256("2999244976951054"),
				},
			},
		},
		{
			name:       "test _sellQuote",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve:         number.NewUint256("306097831372356871541"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve:         number.NewUint256("403206543738"),
					FeeRate:         0,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
			},
			decimals: map[string]uint8{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 18,
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 6,
			},
			wooracle: Wooracle{
				States: map[string]State{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Price:      number.NewUint256("159714000000"),
						Spread:     250000000000000,
						Coeff:      1550000000,
						WoFeasible: true,
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Price:      number.NewUint256("100000000"),
						Spread:     0,
						Coeff:      0,
						WoFeasible: true,
					},
				},
				Decimals: map[string]uint8{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 8,
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 8,
				},
				Timestamp:     time.Now().Unix(),
				StaleDuration: 300,
				Bound:         10000000000000000,
			},
			params: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					Amount: bignumber.NewBig("3739458226"),
				},
				TokenOut: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			},
			expectedErr: nil,
			expectedResult: &poolpkg.CalcAmountOutResult{
				TokenAmountOut: &poolpkg.TokenAmount{
					Token:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
					Amount: bignumber.NewBig10("2340162457578084112"),
				},
				Fee: &poolpkg.TokenAmount{
					Token:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					Amount: bignumber.NewBig10("934864"),
				},
				Gas: DefaultGas.Swap,
				SwapInfo: &woofiV2SwapInfo{
					newPrice:           number.NewUint256("159714925501"),
					newMaxNotionalSwap: number.NewUint256("996261476638"),
					newMaxGamma:        number.NewUint256("2994205288788900"),
				},
			},
		},
		{
			name:       "test _swapBaseToBase",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve:         number.NewUint256("307599458320800914127"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve:         number.NewUint256("422309249032"),
					FeeRate:         0,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
					Reserve:         number.NewUint256("1761585197"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
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
			params: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
					Amount: bignumber.NewBig("195921323"),
				},
				TokenOut: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			},
			expectedErr: nil,
			expectedResult: &poolpkg.CalcAmountOutResult{
				TokenAmountOut: &poolpkg.TokenAmount{
					Token:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
					Amount: bignumber.NewBig10("32603174295822426732"),
				},
				Fee: &poolpkg.TokenAmount{
					Token:  "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
					Amount: bignumber.NewBig10("13032560"),
				},
				Gas: DefaultGas.Swap,
				SwapInfo: &woofiV2SwapInfo{
					newPrice:           number.NewUint256("2661411836801"),
					newMaxNotionalSwap: number.NewUint256("947843883507"),
					newMaxGamma:        number.NewUint256("2743391906854428"),
					base2: &woofiV2SwapInfo{
						newPrice:           number.NewUint256("159814885839"),
						newMaxNotionalSwap: number.NewUint256("947882791139"),
						newMaxGamma:        number.NewUint256("2919218326265450"),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "poolAddress",
						Exchange: string(valueobject.ExchangeWooFiV3),
						Type:     DexTypeWooFiV21,
						Tokens: []string{"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
							"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"},
					},
				},
				quoteToken: tc.quoteToken,
				tokenInfos: tc.tokenInfos,
				decimals:   tc.decimals,
				wooracle:   tc.wooracle,
				gas:        DefaultGas,
			}

			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return pool.CalcAmountOut(tc.params)
			})

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	testCases := []struct {
		name             string
		quoteToken       string
		tokenInfos       map[string]TokenInfo
		decimals         map[string]uint8
		wooracle         Wooracle
		params           poolpkg.CalcAmountOutParams
		expectedErr      error
		expectedReserves map[string]*uint256.Int
	}{
		{
			name:       "test _sellBase",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve:         number.NewUint256("305740102740733506649"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve:         number.NewUint256("403770676421"),
					FeeRate:         0,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
			},
			decimals: map[string]uint8{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 18,
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 6,
			},
			wooracle: Wooracle{
				States: map[string]State{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Price:      number.NewUint256("159709047746"),
						Spread:     270000000000000,
						Coeff:      1550000000,
						WoFeasible: true,
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Price:      number.NewUint256("100000000"),
						Spread:     0,
						Coeff:      0,
						WoFeasible: true,
					},
				},
				Decimals: map[string]uint8{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 8,
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 8,
				},
				Timestamp:     time.Now().Unix(),
				StaleDuration: 300,
				Bound:         10000000000000000,
			},
			params: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
					Amount: bignumber.NewBig("304999404452284472"),
				},
				TokenOut: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			expectedErr: nil,
			expectedReserves: map[string]*uint256.Int{
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": number.NewUint256("403283696665"),
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": number.NewUint256("306045102145185791121"),
			},
		},
		{
			name:       "test _sellQuote",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve:         number.NewUint256("306097831372356871541"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve:         number.NewUint256("403206543738"),
					FeeRate:         0,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
			},
			decimals: map[string]uint8{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 18,
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 6,
			},
			wooracle: Wooracle{
				States: map[string]State{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Price:      number.NewUint256("159714000000"),
						Spread:     250000000000000,
						Coeff:      1550000000,
						WoFeasible: true,
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Price:      number.NewUint256("100000000"),
						Spread:     0,
						Coeff:      0,
						WoFeasible: true,
					},
				},
				Decimals: map[string]uint8{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": 8,
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": 8,
				},
				Timestamp:     time.Now().Unix(),
				StaleDuration: 300,
				Bound:         10000000000000000,
			},
			params: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					Amount: bignumber.NewBig("3739458226"),
				},
				TokenOut: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			},
			expectedErr: nil,
			expectedReserves: map[string]*uint256.Int{
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": number.NewUint256("406945067100"),
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": number.NewUint256("303757668914778787429"),
			},
		},
		{
			name:       "test _swapBaseToBase",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve:         number.NewUint256("307599458320800914127"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve:         number.NewUint256("422309249032"),
					FeeRate:         0,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
				},
				"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
					Reserve:         number.NewUint256("1761585197"),
					FeeRate:         25,
					MaxNotionalSwap: number.NewUint256("1000000000000"),
					MaxGamma:        number.NewUint256("3000000000000000"),
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
			params: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
					Amount: bignumber.NewBig("195921323"),
				},
				TokenOut: "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			},
			expectedErr: nil,
			expectedReserves: map[string]*uint256.Int{
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": number.NewUint256("422296216472"),
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": number.NewUint256("274996284024978487395"),
				"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": number.NewUint256("1957506520"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "poolAddress",
						Exchange: string(valueobject.ExchangeWooFiV3),
						Type:     DexTypeWooFiV21,
						Tokens: []string{"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
							"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"},
					},
				},
				quoteToken: tc.quoteToken,
				tokenInfos: tc.tokenInfos,
				decimals:   tc.decimals,
				wooracle:   tc.wooracle,
				gas:        DefaultGas,
			}

			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return pool.CalcAmountOut(tc.params)
			})
			assert.Equal(t, tc.expectedErr, err)

			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tc.params.TokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			pool.UpdateBalance(updateBalanceParams)

			tokenIn := tc.params.TokenAmountIn.Token
			tokenOut := tc.params.TokenOut
			tokenInReserve := pool.tokenInfos[tokenIn].Reserve
			tokenOutReserve := pool.tokenInfos[tokenOut].Reserve

			assert.Equal(t, tc.expectedReserves[tokenIn], tokenInReserve)
			assert.Equal(t, tc.expectedReserves[tokenOut], tokenOutReserve)
		})
	}
}

func Test_MergeSwaps(t *testing.T) {
	var pool entity.Pool
	_ = json.Unmarshal([]byte(`{"address":"0xed9e3f98bbed560e66b89aac922e29d4596a9642","reserveUsd":1434709.233731838,"amplifiedTvl":1434709.233731838,"exchange":"woofi-v3","type":"woofi-v21","timestamp":1732040619,"reserves":["406865559957507156307876","129023588232874584509","81929754585","123360877725096143906","241126457260"],"tokens":[{"address":"0x78c1b0c915c4faa5fffa6cabf0219da63d7f4cb8","name":"","symbol":"","decimals":18,"weight":1,"swappable":true},{"address":"0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111","name":"","symbol":"","decimals":18,"weight":1,"swappable":true},{"address":"0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9","name":"","symbol":"","decimals":6,"weight":1,"swappable":true},{"address":"0xcda86a272531e8640cd7f1a92c01839911b90bb0","name":"","symbol":"","decimals":18,"weight":1,"swappable":true},{"address":"0x201eba5cc46d216ce6dc03f6a759e8e766e956ae","name":"","symbol":"","decimals":6,"weight":1,"swappable":true}],"extra":"{\"quoteToken\":\"0x201eba5cc46d216ce6dc03f6a759e8e766e956ae\",\"tokenInfos\":{\"0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9\":{\"reserve\":\"81929754585\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0x201eba5cc46d216ce6dc03f6a759e8e766e956ae\":{\"reserve\":\"241126457260\",\"feeRate\":5,\"maxGamma\":\"500000000000000\",\"maxNotionalSwap\":\"1000000000000\"},\"0x78c1b0c915c4faa5fffa6cabf0219da63d7f4cb8\":{\"reserve\":\"406865559957507156307876\",\"feeRate\":25,\"maxGamma\":\"5000000000000000\",\"maxNotionalSwap\":\"500000000000\"},\"0xcda86a272531e8640cd7f1a92c01839911b90bb0\":{\"reserve\":\"123360877725096143906\",\"feeRate\":25,\"maxGamma\":\"3000000000000000\",\"maxNotionalSwap\":\"50000000000\"},\"0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111\":{\"reserve\":\"129023588232874584509\",\"feeRate\":25,\"maxGamma\":\"3000000000000000\",\"maxNotionalSwap\":\"1000000000000\"}},\"wooracle\":{\"address\":\"0x2A375567f5E13F6bd74fDa7627Df3b1Af6BfA5a6\",\"states\":{\"0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9\":{\"price\":\"99901071\",\"spread\":101000000000000,\"coeff\":1400000000,\"woFeasible\":true},\"0x201eba5cc46d216ce6dc03f6a759e8e766e956ae\":{\"price\":\"0\",\"spread\":0,\"coeff\":0,\"woFeasible\":false},\"0x78c1b0c915c4faa5fffa6cabf0219da63d7f4cb8\":{\"price\":\"74330000\",\"spread\":994000000000000,\"coeff\":100000000000,\"woFeasible\":true},\"0xcda86a272531e8640cd7f1a92c01839911b90bb0\":{\"price\":\"326994000000\",\"spread\":755000000000000,\"coeff\":4200000000,\"woFeasible\":true},\"0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111\":{\"price\":\"312108000000\",\"spread\":755000000000000,\"coeff\":4200000000,\"woFeasible\":true}},\"decimals\":{\"0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9\":8,\"0x201eba5cc46d216ce6dc03f6a759e8e766e956ae\":8,\"0x78c1b0c915c4faa5fffa6cabf0219da63d7f4cb8\":8,\"0xcda86a272531e8640cd7f1a92c01839911b90bb0\":8,\"0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111\":8},\"timestamp\":1732040600,\"staleDuration\":9999999999,\"bound\":25000000000000000},\"cloracle\":{\"0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9\":{\"oracleAddress\":\"0x480c8bff72148e0934429a51e5bf9c122f30e1b4\",\"answer\":\"99995020\",\"updatedAt\":\"1732040399\",\"cloPreferred\":false},\"0x201eba5cc46d216ce6dc03f6a759e8e766e956ae\":{\"oracleAddress\":\"0xcced0e6b0850b1d62c53312f2a312c3caeb78611\",\"answer\":\"100113500\",\"updatedAt\":\"1732040399\",\"cloPreferred\":false},\"0x78c1b0c915c4faa5fffa6cabf0219da63d7f4cb8\":{\"oracleAddress\":\"0xd7a801aa8cd28ced2ef0c418e71d44d7744edc3f\",\"answer\":\"74014436\",\"updatedAt\":\"1732037657\",\"cloPreferred\":false},\"0xcda86a272531e8640cd7f1a92c01839911b90bb0\":{\"oracleAddress\":\"0x3708d5ee0dce068022f11dbb35b0cc2062f3afbb\",\"answer\":\"327647445614\",\"updatedAt\":\"1732040399\",\"cloPreferred\":false},\"0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111\":{\"oracleAddress\":\"0xca941f1b43cd2d7882fc6fc0457e9d76aff377e2\",\"answer\":\"312159751502\",\"updatedAt\":\"1732037657\",\"cloPreferred\":false}}}"}`),
		&pool)

	poolSim, err := NewPoolSimulator(pool)
	assert.NoError(t, err)

	_, err = poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9",
			Amount: bignumber.NewBig10("400000000000"),
		},
		TokenOut: "0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111",
	})
	assert.Error(t, err)

	tokenAmtIn250k := poolpkg.TokenAmount{
		Token:  "0x09bc4e0d864854c6afb6eb9a9cdf58ac190d0df9",
		Amount: bignumber.NewBig10("200000000000"),
	}
	res, err := poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn250k,
		TokenOut:      "0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111",
	})
	assert.NoError(t, err)
	poolSim.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn250k,
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})
	_, err = poolSim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn250k,
		TokenOut:      "0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111",
	})
	assert.Error(t, err)
}
