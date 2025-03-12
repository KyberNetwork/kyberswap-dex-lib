package llamma

import (
	"fmt"
	"math"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestStatefullCalcAmountOut(t *testing.T) {
	poolStr := "{\"address\":\"0xfa96ad0a9e64261db86950e2da362f5572c5c6fd\",\"exchange\":\"curve-llamma\",\"type\":\"curve-llamma\",\"timestamp\":0," +
		"\"reserves\":[\"9356923482485339894571\",\"2801882888675171049458\"]," +
		"\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"decimals\":18,\"swappable\":true}," +
		"{\"address\":\"0xac3e018457b222d93114458476f3e3416abbe38f\",\"decimals\":18,\"swappable\":true}],\"extra\":" +
		"\"{\\\"basePrice\\\":\\\"2500000000000000000000\\\",\\\"fee\\\":\\\"10000000000000000\\\"," +
		"\\\"adminFeesX\\\":\\\"0\\\",\\\"adminFeesY\\\":\\\"0\\\",\\\"adminFee\\\":\\\"0\\\",\\\"dynamicFee\\\":\\\"10000000000000000\\\"," +
		"\\\"priceOracle\\\":\\\"2500000000000000000000\\\",\\\"activeBand\\\":0,\\\"minBand\\\":-75,\\\"maxBand\\\":981," +
		"\\\"bands\\\":null}\"," +
		"\"staticExtra\":\"{\\\"A\\\":\\\"100\\\",\\\"priceOracleAddress\\\":\\\"0x28d7880B5b67fB4a0B1c6Ed6c33c33f365113C29\\\"}\",\"blockNumber\":0}"

	bandsX := map[int64]*uint256.Int{}
	bandsY := map[int64]*uint256.Int{
		4:  uint256.MustFromDecimal("52631578947368422"),
		5:  uint256.MustFromDecimal("52631578964035091"),
		6:  uint256.MustFromDecimal("52631578969590647"),
		7:  uint256.MustFromDecimal("52631578969590642"),
		8:  uint256.MustFromDecimal("52631578969590642"),
		9:  uint256.MustFromDecimal("52631578969590642"),
		10: uint256.MustFromDecimal("52631578969590642"),
		11: uint256.MustFromDecimal("52631578952923976"),
		12: uint256.MustFromDecimal("52631578952923976"),
		13: uint256.MustFromDecimal("52631578952923976"),
		14: uint256.MustFromDecimal("52631578952923976"),
		15: uint256.MustFromDecimal("52631578947368421"),
		16: uint256.MustFromDecimal("52631578947368421"),
		17: uint256.MustFromDecimal("52631578947368421"),
		18: uint256.MustFromDecimal("1000052631578947368421"),
		19: uint256.MustFromDecimal("52631578947373201"),
		20: uint256.MustFromDecimal("52631578947373182"),
		21: uint256.MustFromDecimal("52631578947373182"),
		22: uint256.MustFromDecimal("52631578947373182"),
		23: uint256.MustFromDecimal("4761"),
		24: uint256.MustFromDecimal("4761"),
		25: uint256.MustFromDecimal("4761"),
		26: uint256.MustFromDecimal("4761"),
		27: uint256.MustFromDecimal("4761"),
		28: uint256.MustFromDecimal("4761"),
		29: uint256.MustFromDecimal("4761"),
		30: uint256.MustFromDecimal("4761"),
		31: uint256.MustFromDecimal("4761"),
		32: uint256.MustFromDecimal("4761"),
		33: uint256.MustFromDecimal("4761"),
		34: uint256.MustFromDecimal("4761"),
		35: uint256.MustFromDecimal("4761"),
		36: uint256.MustFromDecimal("4761"),
		37: uint256.MustFromDecimal("4761"),
		38: uint256.MustFromDecimal("4761"),
		39: uint256.MustFromDecimal("4761"),
	}

	var ep entity.Pool
	err := json.Unmarshal([]byte(poolStr), &ep)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(ep)
	require.Nil(t, err)
	require.NotNil(t, sim)

	sim.LogARatio = new(int256.Int).SetUint64(10050335853501431)
	sim.activeBand = 0
	sim.minBand = 0
	sim.maxBand = 39
	sim.fee = uint256.MustFromDecimal("10000000000000000")
	sim.bandsX = bandsX
	sim.bandsY = bandsY

	testCases := []struct {
		index             int
		pump              bool
		amountIn          string
		expectedAmountOut string
		err               error
	}{
		{
			index:    0,
			pump:     false,
			amountIn: "100000",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             1,
			pump:              true,
			amountIn:          "10000000000000",
			expectedAmountOut: "3619691851",
		},
		{
			index:             2,
			pump:              true,
			amountIn:          "20000000",
			expectedAmountOut: "7239",
		},
		{
			index:             3,
			pump:              false,
			amountIn:          "500000000",
			expectedAmountOut: "1328490669204",
		},
		{
			index:             4,
			pump:              true,
			amountIn:          "8000000000000",
			expectedAmountOut: "2895753478",
		},
		{
			index:             5,
			pump:              true,
			amountIn:          "90000000000",
			expectedAmountOut: "32577226",
		},
		{
			index:    6,
			pump:     true,
			amountIn: "1",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             7,
			pump:              false,
			amountIn:          "2",
			expectedAmountOut: "2709",
		},
		{
			index:             8,
			pump:              false,
			amountIn:          "999999999999999999999",
			expectedAmountOut: "16761529328088",
		},
		{
			index:    9,
			pump:     false,
			amountIn: "123456789",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             10,
			pump:              true,
			amountIn:          "987654321",
			expectedAmountOut: "357500",
		},
		{
			index:    11,
			pump:     true,
			amountIn: "1",
			err:      ErrZeroSwapAmount,
		},
		{
			index:    12,
			pump:     true,
			amountIn: "10",
			err:      ErrZeroSwapAmount,
		},
		{
			index:    13,
			pump:     true,
			amountIn: "100",
			err:      ErrZeroSwapAmount,
		},
		{
			index:    14,
			pump:     true,
			amountIn: "1000",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             15,
			pump:              true,
			amountIn:          "10000",
			expectedAmountOut: "3",
		},
		{
			index:             16,
			pump:              true,
			amountIn:          "100000",
			expectedAmountOut: "36",
		},
		{
			index:             17,
			pump:              false,
			amountIn:          "100000",
			expectedAmountOut: "265696461",
		},
		{
			index:             18,
			pump:              false,
			amountIn:          "10000",
			expectedAmountOut: "26567478",
		},
		{
			index:             19,
			pump:              false,
			amountIn:          "1000",
			expectedAmountOut: "2655122",
		},
		{
			index:             20,
			pump:              false,
			amountIn:          "100",
			expectedAmountOut: "265512",
		},
		{
			index:             21,
			pump:              false,
			amountIn:          "10",
			expectedAmountOut: "24383",
		},
		{
			index:    22,
			pump:     false,
			amountIn: "1",
			err:      ErrZeroSwapAmount,
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			tokenInIndex, tokenOutIndex := sim.getStableCoinIdx(), sim.getCollateralIdx()
			tokenIn, tokenOut := sim.GetTokens()[tokenInIndex], sim.GetTokens()[tokenOutIndex]
			if !tc.pump {
				tokenIn, tokenOut = tokenOut, tokenIn
			}

			got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig(tc.amountIn)},
				TokenOut:      tokenOut,
			})
			if tc.err != nil {
				require.Equal(t, tc.err, err)
			} else {
				require.Nil(t, err)
				assert.True(t,
					bignumber.NewBig(tc.expectedAmountOut).Cmp(got.TokenAmountOut.Amount) == 0,
					"expected: %s, got: %s", tc.expectedAmountOut, got.TokenAmountOut.Amount.String())

				sim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig10(tc.amountIn)},
					TokenAmountOut: *got.TokenAmountOut,
					SwapInfo:       got.SwapInfo,
					SwapLimit:      nil,
				})
			}
		})
	}

}

func approx(x1, x2, precision, absPrecision float64) bool {
	if precision >= 1 {
		return true
	}
	result := false
	if absPrecision != 0 {
		result = math.Abs(x2-x1) <= absPrecision
	} else {
		absPrecision = 0
	}
	if x2 == 0 {
		return math.Abs(x1) <= absPrecision
	} else if x1 == 0 {
		return math.Abs(x2) <= absPrecision
	}
	return result || (math.Abs(math.Log(x1/x2)) <= precision)
}
