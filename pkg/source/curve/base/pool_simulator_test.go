package base

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
			out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
				TokenOut:      tc.out,
				Limit:         nil,
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

	out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "A", Amount: big.NewInt(510000)},
		TokenOut:      "B",
		Limit:         nil,
	})
	require.Nil(t, err)
	assert.Equal(t, big.NewInt(509863), out.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(153), out.Fee.Amount)
}
