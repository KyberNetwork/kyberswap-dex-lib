package woofiv2_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/woofiv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestGetAmountOut(t *testing.T) {
	// WooPPV2: https://arbiscan.io/address/0xeff23b4be1091b53205e35f3afcd9c7182bf3062#readContract
	// IntegrationHelper: https://arbiscan.io/address/0x28D2B949024FE50627f1EbC5f0Ca3Ca721148E40#readContract
	// WooracleV2_1: https://arbiscan.io/address/0x73504eaCB100c7576146618DC306c97454CB3620#readContract
	testCases := []struct {
		name              string
		fromToken         string
		toToken           string
		fromAmount        *big.Int
		state             *woofiv2.WooFiV2State
		expectedAmountOut *big.Int
		expectedErr       error
	}{
		{
			name:       "test sellBase",
			fromToken:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			toToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			fromAmount: bignumber.NewBig10("304999404452284472"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("262177303"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("305740102740733506649"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159709047746"),
							Spread:       bignumber.NewBig10("270000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("403770676421"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: bignumber.NewBig10("486858012"),
			expectedErr:       nil,
		},
		{
			name:       "test sellQuote",
			fromToken:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			toToken:    "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			fromAmount: bignumber.NewBig10("3739458226"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("259500727"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("306097831372356871541"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159714000000"),
							Spread:       bignumber.NewBig10("250000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("403206543738"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: bignumber.NewBig10("2340162457578084112"),
			expectedErr:       nil,
		},
		{
			name:       "test swapBaseToBase",
			fromToken:  "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
			toToken:    "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			fromAmount: bignumber.NewBig10("195921323"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("262177303"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("307599458320800914127"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159801975726"),
							Spread:       bignumber.NewBig10("479000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
						Reserve:  bignumber.NewBig10("1761585197"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 8,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("2662094951911"),
							Spread:       bignumber.NewBig10("250000000000000"),
							Coeff:        bignumber.NewBig10("4920000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("3440167846485"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("422309249032"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: bignumber.NewBig10("32603174295822426732"),
			expectedErr:       nil,
		},
		{
			name:       "test not enough base2 balance",
			fromToken:  "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
			toToken:    "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			fromAmount: bignumber.NewBig10("17615851970"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("262177303"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("307599458320800914127"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159801975726"),
							Spread:       bignumber.NewBig10("479000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
						Reserve:  bignumber.NewBig10("1761585197"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 8,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("2662094951911"),
							Spread:       bignumber.NewBig10("250000000000000"),
							Coeff:        bignumber.NewBig10("4920000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("3440167846485"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("422309249032"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       woofiv2.ErrBase2BalanceNotEnough,
		},
		{
			name:       "test not enough base2 balance 2",
			fromToken:  "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
			toToken:    "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			fromAmount: bignumber.NewBig10("1761585197000000000"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("262177303"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("307599458320800914127"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159801975726"),
							Spread:       bignumber.NewBig10("479000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f": {
						Reserve:  bignumber.NewBig10("1761585197"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 8,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("2662094951911"),
							Spread:       bignumber.NewBig10("250000000000000"),
							Coeff:        bignumber.NewBig10("4920000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("3440167846485"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("422309249032"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       woofiv2.ErrBase2BalanceNotEnough,
		},
		{
			name:       "test sellBase not enough quoteAmount",
			fromToken:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			toToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			fromAmount: bignumber.NewBig10("305740102740733506649"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("262177303"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("305740102740733506649"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159709047746"),
							Spread:       bignumber.NewBig10("270000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("403770676421"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       woofiv2.ErrQuoteBalanceNotEnough,
		},
		{
			name:       "test sellBase",
			fromToken:  "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			toToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			fromAmount: bignumber.NewBig10("3049994044522844720000000000"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("262177303"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("305740102740733506649"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159709047746"),
							Spread:       bignumber.NewBig10("270000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("403770676421"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       woofiv2.ErrQuoteBalanceNotEnough,
		},
		{
			name:       "test sellQuote not enough balance",
			fromToken:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			toToken:    "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
			fromAmount: bignumber.NewBig10("37394582260000"),
			state: &woofiv2.WooFiV2State{
				QuoteToken:    "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				UnclaimedFee:  bignumber.NewBig10("259500727"),
				Timestamp:     big.NewInt(time.Now().Unix()),
				StaleDuration: bignumber.NewBig10("300"),
				Bound:         bignumber.NewBig10("1000000000000000000000000"),
				TokenInfos: map[string]*woofiv2.TokenInfo{
					"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": {
						Reserve:  bignumber.NewBig10("306097831372356871541"),
						FeeRate:  bignumber.NewBig10("25"),
						Decimals: 18,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("159714000000"),
							Spread:       bignumber.NewBig10("250000000000000"),
							Coeff:        bignumber.NewBig10("1550000000"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("180211834107"),
							CloPreferred: false,
						},
					},
					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": {
						Reserve:  bignumber.NewBig10("403206543738"),
						FeeRate:  bignumber.NewBig10("0"),
						Decimals: 6,
						State: &woofiv2.OracleState{
							Price:        bignumber.NewBig10("100000000"),
							Spread:       bignumber.NewBig10("0"),
							Coeff:        bignumber.NewBig10("0"),
							WoFeasible:   true,
							Decimals:     8,
							CloPrice:     bignumber.NewBig10("10000000"),
							CloPreferred: false,
						},
					},
				},
			},
			expectedAmountOut: nil,
			expectedErr:       woofiv2.ErrBaseBalanceNotEnough,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountOut, err := woofiv2.GetAmountOut(tc.fromToken, tc.toToken, tc.fromAmount, tc.state)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut.String(), amountOut.String())
		})
	}
}
