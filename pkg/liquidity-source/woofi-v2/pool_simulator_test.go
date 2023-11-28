package woofiv2

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
				Gas: defaultGas.Swap,
				SwapInfo: woofiV2SwapInfo{
					newPrice: number.NewUint256("115792089237316195423570985008687907853269984665640563798290"),
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
				Gas: defaultGas.Swap,
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
				Gas: defaultGas.Swap,
				SwapInfo: woofiV2SwapInfo{
					newBase1Price: number.NewUint256("115792089237316195423570985008687907853269984665639197809239"),
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
				gas:        defaultGas,
			}

			result, err := pool.CalcAmountOut(tc.params)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
