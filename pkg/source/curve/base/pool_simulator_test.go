package base

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://etherscan.io/address/0x1005f7406f32a61bd760cfa14accd2737913d546#readContract
	// 	call balances, totalSupply to get pool params
	// 	call get_dy to get amount out
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
		expectedFeeAmount int64
	}{
		{"A", 5000, "B", 4998, 1},
		{"A", 50000, "B", 49986, 15},
		{"B", 50000, "A", 49983, 14},
		{"B", 51000, "A", 50982, 15},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
			"100",
			"1000000000000", "1000000000000",
			"1000000000000000000000000000000", "1000000000000000000000000000000"),
	})
	require.Nil(t, err)
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, 0, len(p.CanSwapTo("LP")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			assert.Equal(t, big.NewInt(tc.expectedFeeAmount), out.Fee.Amount)
		})
	}
}

func TestCalcAmountOut_interpolate_from_initialA_and_futureA(t *testing.T) {
	// if A is getting ramped up then it should interpolate A correctly
	// 100k at zero to 200k at now*2, so now should be 150k, so the same as the contract above -> get expected output from contract get_dy
	now := time.Now().Unix()
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"futureATime\": %v}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			100000, 200000,
			now*2),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
			"100",
			"1000000000000", "1000000000000",
			"1000000000000000000000000000000", "1000000000000000000000000000000"),
	})
	require.Nil(t, err)

	out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: big.NewInt(510000)},
			TokenOut:      "B",
			Limit:         nil,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, big.NewInt(509863), out.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(153), out.Fee.Amount)
}

func BenchmarkCalcAmountOut(b *testing.B) {
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
			"100",
			"1000000000000", "1000000000000",
			"1000000000000000000000000000000", "1000000000000000000000000000000"),
	})
	require.Nil(b, err)

	for i := 0; i < b.N; i++ {
		_, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: big.NewInt(5000)},
			TokenOut:      "B",
			Limit:         nil,
		})
		require.Nil(b, err)
	}
}

func TestGetDyVirtualPrice(t *testing.T) {
	// test data from https://etherscan.io/address/0x1005f7406f32a61bd760cfa14accd2737913d546#readContract
	testcases := []struct {
		i      int
		j      int
		dx     string
		expOut string
	}{
		{0, 1, "100000000000000", "140254485"},
		{0, 1, "1000000000000", "140254485"},
		{0, 1, "100000000", "99936588"},
		{0, 1, "100002233", "99938816"},
		{1, 0, "20000", "19982"},
		{1, 0, "3000200", "2997456"},
		{1, 0, "88001800", "69067695"},
		{1, 0, "100000000000000", "69244498"},
	}
	poolRedis := `{
		"address": "0x1005f7406f32a61bd760cfa14accd2737913d546",
		"reserveUsd": 209.42969262729198,
		"amplifiedTvl": 209.42969262729198,
		"exchange": "curve",
		"type": "curve-base",
		"timestamp": 1705393976,
		"reserves": ["69265278", "140296574", "208111994100559113335"],
		"tokens": [
			{ "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "weight": 1, "swappable": true },
			{ "address": "0xdac17f958d2ee523a2206206994597c13d831ec7", "weight": 1, "swappable": true }
		],
		"extra": "{\"initialA\":\"150000\",\"futureA\":\"150000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"3000000\",\"adminFee\":\"5000000000\"}",
		"staticExtra": "{\"lpToken\":\"0x1005f7406f32a61bd760cfa14accd2737913d546\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}"
	}`
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	require.Nil(t, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(t, err)

	v, dCached, err := p.GetVirtualPrice()
	require.Nil(t, err)
	assert.Equal(t, bignumber.NewBig10("1006923185919753102"), v)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			dy, err := testutil.MustConcurrentSafe[*big.Int](t, func() (any, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, bignumber.NewBig10(tc.dx), nil)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expOut), dy)

			// test using cached D
			dy, err = testutil.MustConcurrentSafe[*big.Int](t, func() (any, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, bignumber.NewBig10(tc.dx), dCached)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expOut), dy)
		})
	}
}
