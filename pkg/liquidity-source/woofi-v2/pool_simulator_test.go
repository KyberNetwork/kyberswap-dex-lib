package woofiv2

import (
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator_NewPool(t *testing.T) {
	entityPool := entity.Pool{
		Address:  "0x3b3e4b4741e91af52d0e9ad8660573e951c88524",
		Exchange: "woofi-v2",
		Type:     "woofi-v2",
		Reserves: []string{
			"42419821301826468743128",
			"100926020558383543635",
			"2000733752",
			"529883163498030559696795",
			"225170288375",
			"620679347458",
		},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
				Weight:    1,
				Decimals:  18,
				Swappable: true,
			},
			{
				Address:   "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab",
				Weight:    1,
				Decimals:  18,
				Swappable: true,
			},
		},
		Extra:       "{\"quoteToken\":\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\",\"tokenInfos\":{\"0x152b9d0fdc40c096757f570a51e494bd4b943e50\":{\"reserve\":\"0x7740c638\",\"feeRate\":25},\"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab\":{\"reserve\":\"0x578a140f80838f553\",\"feeRate\":25},\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\":{\"reserve\":\"0x346d31eef7\",\"feeRate\":5},\"0xabc9547b534519ff73921b1fba6e672b5f58d083\":{\"reserve\":\"0x7035061b20231788979b\",\"feeRate\":25},\"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7\":{\"reserve\":\"0x8fb9547642a62f887d8\",\"feeRate\":25},\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\":{\"reserve\":\"0x90835f3d02\",\"feeRate\":0}},\"wooracle\":{\"address\":\"0xc13843aE0D2C5ca9E0EfB93a78828446D8173d19\",\"states\":{\"0x152b9d0fdc40c096757f570a51e494bd4b943e50\":{\"price\":\"0x3766a090400\",\"spread\":500000000000000,\"coeff\":2910510000,\"woFeasible\":true},\"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab\":{\"price\":\"0x2ff660c540\",\"spread\":500000000000000,\"coeff\":3676430000,\"woFeasible\":true},\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\":{\"price\":\"0x5f69798\",\"spread\":160022000000000,\"coeff\":2466840000,\"woFeasible\":true},\"0xabc9547b534519ff73921b1fba6e672b5f58d083\":{\"price\":\"0x1526f74\",\"spread\":2750000000000000,\"coeff\":157506000000,\"woFeasible\":true},\"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7\":{\"price\":\"0x7eb16f1c\",\"spread\":868270000000000,\"coeff\":2668470000,\"woFeasible\":true},\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\":{\"price\":\"0x5f5e100\",\"spread\":0,\"coeff\":0,\"woFeasible\":true}},\"decimals\":{\"0x152b9d0fdc40c096757f570a51e494bd4b943e50\":8,\"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab\":8,\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\":8,\"0xabc9547b534519ff73921b1fba6e672b5f58d083\":8,\"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7\":8,\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\":8}}}",
		BlockNumber: 0,
	}
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7",
			Amount: bignumber.NewBig10("10000000000000000000"),
		},
		TokenOut: "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab",
	}

	pool, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	pool.wooracle.Timestamp = time.Now().Unix()
	pool.wooracle.StaleDuration = 300
	pool.wooracle.Bound = 10000000000000000

	result, err := pool.CalcAmountOut(params)

	assert.Nil(t, err)
	assert.Equal(t, "102869361275421525", result.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_Arithmetic_OverflowUnderflow(t *testing.T) {
	entityPool := entity.Pool{
		Address:  "0xd1778f9df3eee5473a9640f13682e3846f61febc",
		Exchange: "woofi-v2",
		Type:     "woofi-v2",
		Reserves: []string{
			"301370617381821852207",
			"785512143",
			"177053835630",
			"97558688283555321324212",
			"167081703216",
			"152515901952",
		},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0x4200000000000000000000000000000000000006",
				Weight:    1,
				Decimals:  18,
				Swappable: true,
			},
			{
				Address:   "0x68f180fcce6836688e9084f035309e29bf0a2095",
				Weight:    1,
				Decimals:  8,
				Swappable: true,
			},
			{
				Address:   "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
				Weight:    1,
				Decimals:  6,
				Swappable: true,
			},
			{
				Address:   "0x4200000000000000000000000000000000000042",
				Weight:    1,
				Decimals:  18,
				Swappable: true,
			},
			{
				Address:   "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
				Weight:    1,
				Decimals:  6,
				Swappable: true,
			},
			{
				Address:   "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
				Weight:    1,
				Decimals:  6,
				Swappable: true,
			},
		},
		Extra: "{\"quoteToken\":\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\",\"tokenInfos\":{\"0x0b2c639c533813f4aa9d7837caf62653d097ff85\":{\"reserve\":\"0x29393b216e\",\"feeRate\":5},\"0x4200000000000000000000000000000000000006\":{\"reserve\":\"0x10565b83c75fa7aa2f\",\"feeRate\":25},\"0x4200000000000000000000000000000000000042\":{\"reserve\":\"0x14a8aac659cf6a43a2b4\",\"feeRate\":25},\"0x68f180fcce6836688e9084f035309e29bf0a2095\":{\"reserve\":\"0x2ed1f6cf\",\"feeRate\":25},\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\":{\"reserve\":\"0x2382a7fa00\",\"feeRate\":0},\"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58\":{\"reserve\":\"0x26e6d87730\",\"feeRate\":5}},\"wooracle\":{\"address\":\"0xd589484d3A27B7Ce5C2C7F829EB2e1D163f95817\",\"states\":{\"0x0b2c639c533813f4aa9d7837caf62653d097ff85\":{\"price\":\"0x5f5640d\",\"spread\":50000000000000,\"coeff\":3940000000,\"woFeasible\":true},\"0x4200000000000000000000000000000000000006\":{\"price\":\"0x34d8869cc0\",\"spread\":366000000000000,\"coeff\":2260000000,\"woFeasible\":true},\"0x4200000000000000000000000000000000000042\":{\"price\":\"0xf3671b0\",\"spread\":1570000000000000,\"coeff\":3570000000,\"woFeasible\":true},\"0x68f180fcce6836688e9084f035309e29bf0a2095\":{\"price\":\"0x4030c6ec900\",\"spread\":427000000000000,\"coeff\":3950000000,\"woFeasible\":true},\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\":{\"price\":\"0x5f5e100\",\"spread\":0,\"coeff\":0,\"woFeasible\":true},\"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58\":{\"price\":\"0x5f5b671\",\"spread\":101000000000000,\"coeff\":3960000000,\"woFeasible\":true}},\"decimals\":{\"0x0b2c639c533813f4aa9d7837caf62653d097ff85\":8,\"0x4200000000000000000000000000000000000006\":8,\"0x4200000000000000000000000000000000000042\":8,\"0x68f180fcce6836688e9084f035309e29bf0a2095\":8,\"0x7f5c764cbc14f9669b88837ca1490cca17c31607\":8,\"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58\":8}}}",
	}
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x4200000000000000000000000000000000000006",
			Amount: bignumber.NewBig10("1000000000000000000000"),
		},
		TokenOut: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
	}

	pool, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	pool.wooracle.Timestamp = time.Now().Unix()
	pool.wooracle.StaleDuration = 300
	pool.wooracle.Bound = 10000000000000000

	_, err = pool.CalcAmountOut(params)

	assert.Equal(t, ErrArithmeticOverflowUnderflow, err)
}

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
					Reserve: number.NewUint256("305740102740733506649"),
					FeeRate: 25,
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve: number.NewUint256("403770676421"),
					FeeRate: 0,
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
			expectedResult: &poolpkg.CalcAmountOutResult{
				TokenAmountOut: &poolpkg.TokenAmount{
					Token:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
					Amount: bignumber.NewBig10("486858012"),
				},
				Fee: &poolpkg.TokenAmount{
					Token:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
					Amount: bignumber.NewBig10("121744"),
				},
				Gas: DefaultGas.Swap,
				SwapInfo: woofiV2SwapInfo{
					newPrice: number.NewUint256("159708806577"),
				},
			},
		},
		{
			name:       "test _sellQuote",
			quoteToken: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			tokenInfos: map[string]TokenInfo{
				"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
					Reserve: number.NewUint256("306097831372356871541"),
					FeeRate: 25,
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve: number.NewUint256("403206543738"),
					FeeRate: 0,
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
				SwapInfo: woofiV2SwapInfo{
					newPrice: number.NewUint256("159715850993"),
				},
			},
		},
		{
			name:       "test _swapBaseToBase",
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
				SwapInfo: woofiV2SwapInfo{
					newBase1Price: number.NewUint256("2660728721692"),
					newBase2Price: number.NewUint256("159827793868"),
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
						Exchange: "woofi-v2",
						Type:     "woofi-v2",
						Tokens:   []string{"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"},
					},
				},
				quoteToken: tc.quoteToken,
				tokenInfos: tc.tokenInfos,
				decimals:   tc.decimals,
				wooracle:   tc.wooracle,
				gas:        DefaultGas,
			}

			result, err := pool.CalcAmountOut(tc.params)
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
					Reserve: number.NewUint256("305740102740733506649"),
					FeeRate: 25,
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve: number.NewUint256("403770676421"),
					FeeRate: 0,
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
					Reserve: number.NewUint256("306097831372356871541"),
					FeeRate: 25,
				},
				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
					Reserve: number.NewUint256("403206543738"),
					FeeRate: 0,
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
						Exchange: "woofi-v2",
						Type:     "woofi-v2",
						Tokens:   []string{"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"},
					},
				},
				quoteToken: tc.quoteToken,
				tokenInfos: tc.tokenInfos,
				decimals:   tc.decimals,
				wooracle:   tc.wooracle,
				gas:        DefaultGas,
			}

			result, err := pool.CalcAmountOut(tc.params)
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
