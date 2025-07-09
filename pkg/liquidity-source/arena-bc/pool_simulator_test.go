package arenabc

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulatorTestSuite struct {
	suite.Suite

	pools map[string]string
	sims  map[string]*PoolSimulator
}

func (ts *PoolSimulatorTestSuite) SetupSuite() {
	ts.pools = map[string]string{
		"16830": `{"address":"0x4fab166825e00567a93a6169efff4a7f52a8c8e7","exchange":"arena-bc","type":"arena-bc","timestamp":0,"reserves":["0","7300000000000000000000000000"],"tokens":[{"address":"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7","symbol":"WAVAX","decimals":18,"swappable":true},{"address":"0x4fab166825e00567a93a6169efff4a7f52a8c8e7","symbol":"KTTY","decimals":18,"swappable":true}],"extra":"{\"p\":false,\"cD\":true,\"tP\":{\"cS\":\"232210432401\",\"a\":901,\"b\":0,\"lD\":false,\"lP\":27,\"sP\":73,\"cFBP\":0,\"pA\":\"0x0e27c2b8ca8dc4feac90d6b0ea52a1f9f878b8d1\"},\"tS\":\"0\",\"tB\":\"2\",\"mTFS\":\"7300000000000000000000000000\",\"pFBP\":100,\"rFBP\":25,\"aTS\":\"10000000000000000000000000000\"}","staticExtra":"{\"cI\":43114,\"tM\":\"0x8315f1eb449dd4b779495c3a0b05e5d194446c6e\",\"tI\":16830}"}`,
		"645":   `{"address":"0x4341214c67b02d7f94d3cc84ba0f954c59623542","exchange":"arena-bc","type":"arena-bc","timestamp":0,"reserves":["1950617281041677977","6148392765000000000000000000"],"tokens":[{"address":"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7","symbol":"WAVAX","decimals":18,"swappable":true},{"address":"0x4341214c67b02d7f94d3cc84ba0f954c59623542","symbol":"LOGIC","decimals":18,"swappable":true}],"extra":"{\"p\":false,\"cD\":true,\"tP\":{\"cS\":\"232210432401\",\"a\":901,\"b\":0,\"lD\":false,\"lP\":27,\"sP\":73,\"cFBP\":0,\"pA\":\"0xcc75b6c4e9994945b252f7b5130863527bf4a942\"},\"tS\":\"1151607235000000000000000000\",\"tB\":\"1975308639029547320\",\"mTFS\":\"6148392765000000000000000000\",\"pFBP\":100,\"rFBP\":25,\"aTS\":\"10000000000000000000000000000\"}","staticExtra":"{\"cI\":43114,\"tM\":\"0x8315f1eb449dd4b779495c3a0b05e5d194446c6e\",\"tI\":645}"}`,
	}

	ts.sims = map[string]*PoolSimulator{}
	for k, p := range ts.pools {
		var ep entity.Pool
		err := json.Unmarshal([]byte(p), &ep)
		ts.Require().Nil(err)

		sim, err := NewPoolSimulator(ep)
		ts.Require().Nil(err)
		ts.Require().NotNil(sim)

		ts.sims[k] = sim
	}
}

func (ts *PoolSimulatorTestSuite) TestCalcAmountOut() {
	ts.T().Parallel()

	AVAX := "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7"
	testCases := []struct {
		name     string
		pool     string
		tokenIn  string
		tokenOut string
		amountIn string

		expectedAmountOut       string
		expectedRemainingAmount string
		expectedError           error
	}{
		{
			name:                    "buy AVAX -> KITTY",
			pool:                    "16830",
			tokenIn:                 AVAX,
			tokenOut:                "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:                "1",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "870000000000000000000",
		},
		{
			name:                    "buy AVAX -> KITTY",
			pool:                    "16830",
			tokenIn:                 AVAX,
			tokenOut:                "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:                "100",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "4241000000000000000000",
		},
		{
			name:                    "buy AVAX -> KITTY",
			pool:                    "16830",
			tokenIn:                 AVAX,
			tokenOut:                "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:                "1000000000000000000",
			expectedRemainingAmount: "1972439604",
			expectedAmountOut:       "914031268000000000000000000",
		},
		{
			name:                    "buy AVAX -> KITTY",
			pool:                    "16830",
			tokenIn:                 AVAX,
			tokenOut:                "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:                "1000000000000000000000",
			expectedRemainingAmount: "490568938852763753180",
			expectedAmountOut:       "7300000000000000000000000000",
		},
		{
			name:                    "buy AVAX -> KITTY",
			pool:                    "16830",
			tokenIn:                 AVAX,
			tokenOut:                "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:                "509431061147236246820",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "7300000000000000000000000000",
		},
		{
			name:          "sell KITTY -> AVAX",
			pool:          "16830",
			tokenOut:      AVAX,
			tokenIn:       "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:      "1000",
			expectedError: ErrZeroSwap,
		},
		{
			name:          "sell KITTY -> AVAX",
			pool:          "16830",
			tokenOut:      AVAX,
			tokenIn:       "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:      "100000000000000000",
			expectedError: ErrZeroSwap,
		},
		{
			name:          "sell KITTY -> AVAX",
			pool:          "16830",
			tokenOut:      AVAX,
			tokenIn:       "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:      "1000000000000000000",
			expectedError: ErrZeroSwap,
		},
		{
			name:          "sell KITTY -> AVAX",
			pool:          "16830",
			tokenOut:      AVAX,
			tokenIn:       "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountIn:      "1000000000000000000000",
			expectedError: ErrZeroSwap,
		},

		{
			name:          "buy AVAX -> LOGIC",
			pool:          "645",
			tokenIn:       AVAX,
			tokenOut:      "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:      "1000000000",
			expectedError: ErrZeroSwap,
		},
		{
			name:                    "buy AVAX -> LOGIC",
			pool:                    "645",
			tokenIn:                 AVAX,
			tokenOut:                "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "10000000000",
			expectedRemainingAmount: "4789890326",
			expectedAmountOut:       "1000000000000000000",
		},
		{
			name:                    "buy AVAX -> LOGIC",
			pool:                    "645",
			tokenIn:                 AVAX,
			tokenOut:                "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "5210109674",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "1000000000000000000",
		},
		{
			name:                    "buy AVAX -> LOGIC",
			pool:                    "645",
			tokenIn:                 AVAX,
			tokenOut:                "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "100000000000000000010000",
			expectedRemainingAmount: "99492568938849781179840",
			expectedAmountOut:       "6148392765000000000000000000",
		},
		{
			name:                    "buy AVAX -> LOGIC",
			pool:                    "645",
			tokenIn:                 AVAX,
			tokenOut:                "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "507431061150218830160",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "6148392765000000000000000000",
		},
		{
			name:                    "buy AVAX -> LOGIC",
			pool:                    "645",
			tokenIn:                 AVAX,
			tokenOut:                "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "99000000000000000000",
			expectedRemainingAmount: "18703864935",
			expectedAmountOut:       "3105045073000000000000000000",
		},
		{
			name:                    "buy AVAX -> LOGIC",
			pool:                    "645",
			tokenIn:                 AVAX,
			tokenOut:                "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "98999999981296135065",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "3105045073000000000000000000",
		},
		{
			name:          "sell LOGIC -> AVAX",
			pool:          "645",
			tokenOut:      AVAX,
			tokenIn:       "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:      "100000000000000000",
			expectedError: ErrZeroSwap,
		},
		{
			name:                    "sell LOGIC -> AVAX",
			pool:                    "645",
			tokenOut:                AVAX,
			tokenIn:                 "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "1000000000000000000",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "5081464982",
		},
		{
			name:                    "sell LOGIC -> AVAX",
			pool:                    "645",
			tokenOut:                AVAX,
			tokenIn:                 "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "10000000000000000000000",
			expectedRemainingAmount: "0",
			expectedAmountOut:       "50814208618811",
		},
		{
			name:                    "sell LOGIC -> AVAX and floor amountIn if not divisible by granularity scaler",
			pool:                    "645",
			tokenOut:                AVAX,
			tokenIn:                 "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "987654321111110000000",
			expectedRemainingAmount: "654321111110000000",
			expectedAmountOut:       "5015401643391",
		},
		{
			name:                    "sell LOGIC -> AVAX and floor amountIn if not divisible by granularity scaler",
			pool:                    "645",
			tokenOut:                AVAX,
			tokenIn:                 "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountIn:                "213828497326472364982313213",
			expectedRemainingAmount: "326472364982313213",
			expectedAmountOut:       "897297939412620329",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.pool, func(t *testing.T) {
			cloned := ts.sims[tc.pool].CloneState()

			res, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: bignum.NewBig(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})

			if tc.expectedError == nil {
				require.NotNil(t, res)
				require.Equal(t, tc.expectedAmountOut, res.TokenAmountOut.Amount.String())
				cloned.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignum.NewBig(tc.amountIn),
					},
					TokenAmountOut: *res.TokenAmountOut,
					SwapInfo:       res.SwapInfo,
				})
				require.Equal(t, tc.expectedRemainingAmount, res.RemainingTokenAmountIn.Amount.String())
				require.Equal(t, tc.expectedAmountOut, res.TokenAmountOut.Amount.String())
			} else {
				require.ErrorContains(t, err, tc.expectedError.Error())
			}
		})
	}
}

func (ts *PoolSimulatorTestSuite) TestCalcAmountIn() {
	ts.T().Parallel()

	AVAX := "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7"
	testCases := []struct {
		name      string
		pool      string
		tokenIn   string
		tokenOut  string
		amountOut string

		expectedAmountIn string
		expectedError    error
	}{
		{
			name:             "buy AVAX -> KITTY",
			pool:             "16830",
			tokenIn:          AVAX,
			tokenOut:         "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:        "1",
			expectedAmountIn: "1",
		},
		{
			name:             "buy AVAX -> KITTY",
			pool:             "16830",
			tokenIn:          AVAX,
			tokenOut:         "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:        "1000000000000000000",
			expectedAmountIn: "1",
		},
		{
			name:             "buy AVAX -> KITTY",
			pool:             "16830",
			tokenIn:          AVAX,
			tokenOut:         "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:        "1130000000000000000000",
			expectedAmountIn: "2",
		},
		{
			name:             "buy AVAX -> KITTY",
			pool:             "16830",
			tokenIn:          AVAX,
			tokenOut:         "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:        "509431061147236246820",
			expectedAmountIn: "1",
		},
		{
			name:             "buy AVAX -> KITTY",
			pool:             "16830",
			tokenIn:          AVAX,
			tokenOut:         "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:        "7300000000000000000000000000",
			expectedAmountIn: "509431061147236246820",
		},
		{
			name:          "buy AVAX -> KITTY",
			pool:          "16830",
			tokenIn:       AVAX,
			tokenOut:      "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:     "7300000000000000000000000001",
			expectedError: ErrBuyLimitExceeded,
		},
		{
			name:          "buy AVAX -> KITTY",
			pool:          "16830",
			tokenIn:       AVAX,
			tokenOut:      "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:     "1100030210301203021030210302103021",
			expectedError: ErrBuyLimitExceeded,
		},
		{
			name:          "sell KITTY -> AVAX",
			pool:          "16830",
			tokenOut:      AVAX,
			tokenIn:       "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:     "1000",
			expectedError: ErrSellLimitExceeded,
		},
		{
			name:          "sell KITTY -> AVAX",
			pool:          "16830",
			tokenOut:      AVAX,
			tokenIn:       "0x4fab166825e00567a93a6169efff4a7f52a8c8e7",
			amountOut:     "100000000000000000",
			expectedError: ErrSellLimitExceeded,
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "1000000000",
			expectedAmountIn: "5210109674",
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "10000000000000000000",
			expectedAmountIn: "52101097152",
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "321321321314432",
			expectedAmountIn: "5210109674",
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "100000000000000000010000",
			expectedAmountIn: "521061421389956",
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "507431061150218830160",
			expectedAmountIn: "2646736879868",
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "99000000000000000000",
			expectedAmountIn: "515800901667",
		},
		{
			name:             "buy AVAX -> LOGIC",
			pool:             "645",
			tokenIn:          AVAX,
			tokenOut:         "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "97999999981296135065",
			expectedAmountIn: "510590791106",
		},
		{
			name:             "sell LOGIC -> AVAX",
			pool:             "645",
			tokenOut:         AVAX,
			tokenIn:          "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "100000000000000000",
			expectedAmountIn: "20025576000000000000000000",
		},
		{
			name:             "sell LOGIC -> AVAX",
			pool:             "645",
			tokenOut:         AVAX,
			tokenIn:          "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "1000000000",
			expectedAmountIn: "1000000000000000000",
		},
		{
			name:          "sell LOGIC -> AVAX",
			pool:          "645",
			tokenOut:      AVAX,
			tokenIn:       "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:     "10000000000000000000000",
			expectedError: ErrSellLimitExceeded,
		},
		{
			name:             "sell LOGICr",
			pool:             "645",
			tokenOut:         AVAX,
			tokenIn:          "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "987654321111110",
			expectedAmountIn: "194397000000000000000000",
		},
		{
			name:             "sell LOGIC -> AVAX",
			pool:             "645",
			tokenOut:         AVAX,
			tokenIn:          "0x4341214c67b02d7f94d3cc84ba0f954c59623542",
			amountOut:        "18000000000000000",
			expectedAmountIn: "3553238000000000000000000",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.pool, func(t *testing.T) {
			cloned := ts.sims[tc.pool].CloneState().(*PoolSimulator)

			res, err := cloned.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  tc.tokenOut,
					Amount: bignum.NewBig(tc.amountOut),
				},
				TokenIn: tc.tokenIn,
			})

			if tc.expectedError == nil {
				require.NotNil(t, res)
				require.Equal(t, tc.expectedAmountIn, res.TokenAmountIn.Amount.String())
				cloned.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignum.NewBig(tc.amountOut),
					},
					TokenAmountOut: *res.TokenAmountIn,
					SwapInfo:       res.SwapInfo,
				})
				require.Equal(t, tc.expectedAmountIn, res.TokenAmountIn.Amount.String())
			} else {
				require.ErrorContains(t, err, tc.expectedError.Error())
			}
		})
	}
}

func TestPoolSimulatorTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PoolSimulatorTestSuite))
}
